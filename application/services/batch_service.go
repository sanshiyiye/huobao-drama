package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	batchFramesConcurrency  = 1  // 一次只处理一个分镜
	batchVideosConcurrency  = 1  // 一次只处理一个分镜
	framePromptPollInterval = 3 * time.Second
	framePromptPollMaxWait  = 120 * time.Second
)

// BatchService 一键生图/一键出片批量任务
type BatchService struct {
	db               *gorm.DB
	taskService      *TaskService
	imageGenService  *ImageGenerationService
	framePromptSvc   *FramePromptService
	videoGenService  *VideoGenerationService
	videoMergeSvc    *VideoMergeService
	log              *logger.Logger
}

// EpisodeVideoTaskResult 一键章节视频任务结果
type EpisodeVideoTaskResult struct {
	Success        bool   `json:"success"`
	CurrentPhase   string `json:"current_phase"`   // frames, videos, merge, completed, failed, cancelled
	PhaseMessage   string `json:"phase_message"`
	TotalStoryboards int   `json:"total_storyboards"`
	CompletedFrames int   `json:"completed_frames"`  // 已完成生图的分镜数
	CompletedVideos int   `json:"completed_videos"`  // 已完成出片的分镜数
	FailedStoryboards []int `json:"failed_storyboards,omitempty"` // 失败的分镜编号列表
	MergeID        *uint  `json:"merge_id,omitempty"`
	Error          string `json:"error,omitempty"`
}

// EpisodeVideoTaskProgress 用于存储在 AsyncTask.Result 中的进度状态
type EpisodeVideoTaskProgress struct {
	Phase          string `json:"phase"`
	TotalStoryboards int   `json:"total_storyboards"`
	CurrentStoryboard int  `json:"current_storyboard"` // 当前处理到第几个分镜
	CompletedFrames int   `json:"completed_frames"`
	CompletedVideos int   `json:"completed_videos"`
	Cancelled      bool   `json:"cancelled"`
	FailedStoryboards []int `json:"failed_storyboards,omitempty"` // 失败的分镜编号列表
}

// NewBatchService 创建批量服务
func NewBatchService(
	db *gorm.DB,
	taskService *TaskService,
	imageGenService *ImageGenerationService,
	framePromptSvc *FramePromptService,
	videoGenService *VideoGenerationService,
	videoMergeSvc *VideoMergeService,
	log *logger.Logger,
) *BatchService {
	return &BatchService{
		db:               db,
		taskService:      taskService,
		imageGenService:  imageGenService,
		framePromptSvc:   framePromptSvc,
		videoGenService:  videoGenService,
		videoMergeSvc:    videoMergeSvc,
		log:              log,
	}
}

// BatchGenerateFramesResult 一键生图结果
type BatchGenerateFramesResult struct {
	Total     int   `json:"total"`
	Success   int   `json:"success"`
	Failed    int   `json:"failed"`
	FailedIDs []uint `json:"failed_storyboard_ids,omitempty"`
}

// BatchImageProgress 剧集首尾帧生图进度（只读，供前端轮询）
type BatchImageProgress struct {
	ExpectedTotal int `json:"expected_total"`
	Completed     int `json:"completed"`
	Failed        int `json:"failed"`
	Pending       int `json:"pending"`
	Processing    int `json:"processing"`
}

// GetBatchImageProgress 返回该剧集下首尾帧图片的生成进度，供前端轮询。
// expected_total 取「分镜数量 × 2」，因为每个分镜需要生成首帧和尾帧两张图片，这样进度计算会更准确。
func (s *BatchService) GetBatchImageProgress(episodeID string) (*BatchImageProgress, error) {
	var storyboardIDs []uint
	if err := s.db.Model(&models.Storyboard{}).Where("episode_id = ?", episodeID).Pluck("id", &storyboardIDs).Error; err != nil {
		return nil, err
	}
	if len(storyboardIDs) == 0 {
		return &BatchImageProgress{ExpectedTotal: 0, Completed: 0, Failed: 0, Pending: 0, Processing: 0}, nil
	}

	var counts []struct {
		Status string
		Total  int64
	}
	if err := s.db.Model(&models.ImageGeneration{}).
		Where("storyboard_id IN ? AND frame_type IN ? AND image_type = ?", storyboardIDs, []string{"first", "last"}, models.ImageTypeStoryboard).
		Select("status, count(*) as total").
		Group("status").
		Scan(&counts).Error; err != nil {
		return nil, err
	}

	res := &BatchImageProgress{}
	res.ExpectedTotal = len(storyboardIDs) * 2 // 每个分镜需要生成首帧和尾帧两张图片
	for _, c := range counts {
		switch c.Status {
		case string(models.ImageStatusCompleted):
			res.Completed = int(c.Total)
		case string(models.ImageStatusFailed):
			res.Failed = int(c.Total)
		case string(models.ImageStatusPending):
			res.Pending = int(c.Total)
		case string(models.ImageStatusProcessing):
			res.Processing = int(c.Total)
		}
	}

	// 确保统计的记录数不超过预期总数
	actualProcessed := res.Completed + res.Failed
	if actualProcessed > res.ExpectedTotal {
		// 如果已完成和失败的记录数超过预期总数，可能是重复生成的图片
		// 我们需要调整统计，确保总和不超过预期总数
		if res.Completed > res.ExpectedTotal {
			res.Completed = res.ExpectedTotal
			res.Failed = 0
		} else {
			res.Failed = res.ExpectedTotal - res.Completed
		}
		actualProcessed = res.ExpectedTotal
	}

	remaining := res.ExpectedTotal - actualProcessed
	if remaining < 0 {
		remaining = 0
	}
	if res.Pending+res.Processing > remaining {
		res.Pending = remaining
		res.Processing = 0
	}

	return res, nil
}

// BatchGenerateFrames 创建一键生图任务并立即返回 task_id，后台异步执行
func (s *BatchService) BatchGenerateFrames(episodeID string) (string, error) {
	task, err := s.taskService.CreateTask("batch_generate_frames", episodeID)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}
	go s.processBatchFrames(task.ID, episodeID)
	s.log.Infow("Batch generate frames task created", "task_id", task.ID, "episode_id", episodeID)
	return task.ID, nil
}

// BatchRetryFailedFrames 重试失败分镜的首尾帧生成（创建新任务）
func (s *BatchService) BatchRetryFailedFrames(episodeID string, failedStoryboardIDs []uint) (string, error) {
	task, err := s.taskService.CreateTask("batch_retry_failed_frames", episodeID)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}
	go s.processRetryFailedFrames(task.ID, episodeID, failedStoryboardIDs)
	s.log.Infow("Batch retry failed frames task created", "task_id", task.ID, "episode_id", episodeID, "failed_storyboard_ids", failedStoryboardIDs)
	return task.ID, nil
}

// ResumeBatchFramesTask 恢复一键生图任务（继续执行失败的分镜）
func (s *BatchService) ResumeBatchFramesTask(taskID string, failedStoryboardIDs []uint) (string, error) {
	oldTask, err := s.taskService.GetTask(taskID)
	if err != nil {
		return "", fmt.Errorf("原任务不存在")
	}

	var result BatchGenerateFramesResult
	if oldTask.Result != "" {
		json.Unmarshal([]byte(oldTask.Result), &result)
	}

	episodeID := oldTask.ResourceID
	
	// ⭐ 改进：恢复原任务状态，而不是创建新任务
	// 计算当前进度（基于已完成的分镜数）
	currentProgress := 0
	progressMessage := "正在继续执行..."
	
	if result.Total > 0 {
		// 已完成的分镜数 = 总数 - 失败的分镜数
		completedCount := result.Total - len(failedStoryboardIDs)
		if completedCount < 0 {
			completedCount = 0
		}
		currentProgress = (completedCount * 100) / result.Total
		progressMessage = fmt.Sprintf("继续处理 %d 个失败分镜...", len(failedStoryboardIDs))
	} else {
		// 如果没有总数信息，根据失败分镜数量估算
		// 假设至少有失败的分镜需要处理
		progressMessage = fmt.Sprintf("继续处理 %d 个分镜...", len(failedStoryboardIDs))
	}
	
	// 恢复原任务状态
	if err := s.taskService.ResumeTask(taskID, currentProgress, progressMessage); err != nil {
		return "", fmt.Errorf("恢复任务失败: %w", err)
	}

	// 使用原任务ID继续执行
	go s.processRetryFailedFrames(taskID, episodeID, failedStoryboardIDs)

	// ⭐ 改进：返回原任务ID，而不是新任务ID
	return taskID, nil
}

func (s *BatchService) processRetryFailedFrames(taskID string, episodeID string, failedStoryboardIDs []uint) {
	_ = s.taskService.UpdateTaskStatus(taskID, "processing", 0, "正在获取失败分镜...")

	var episode models.Episode
	if err := s.db.Where("id = ?", episodeID).First(&episode).Error; err != nil {
		_ = s.taskService.UpdateTaskStatus(taskID, "failed", 0, "剧集不存在")
		return
	}

	var storyboards []models.Storyboard
	if err := s.db.Where("id IN ?", failedStoryboardIDs).
		Preload("Background").
		Preload("Characters").
		Order("storyboard_number ASC").
		Find(&storyboards).Error; err != nil {
		_ = s.taskService.UpdateTaskStatus(taskID, "failed", 0, "获取失败分镜失败")
		return
	}

	total := len(storyboards)
	if total == 0 {
		_ = s.taskService.UpdateTaskResult(taskID, BatchGenerateFramesResult{Total: 0, Success: 0, Failed: 0})
		return
	}

	dramaID := strconv.FormatUint(uint64(episode.DramaID), 10)
	var failedIDs []uint
	var completedCount int
	var mu sync.Mutex
	sem := make(chan struct{}, batchFramesConcurrency)
	cancelled := false

	for _, sb := range storyboards {
		// 检查任务是否已取消
		task, err := s.taskService.GetTask(taskID)
		if err == nil && task.Status == "failed" && task.Error == "用户取消" {
			cancelled = true
			s.log.Infow("Task cancelled, stopping retry failed frames", "task_id", taskID)
			break
		}

		// 双重检查
		if cancelled {
			break
		}

		sem <- struct{}{}
		go func(storyboard models.Storyboard) {
			defer func() { <-sem }()
			
			// ⭐ 关键改进：goroutine启动后立即检查任务是否已取消
			task, err := s.taskService.GetTask(taskID)
			if err == nil && task.Status == "failed" && task.Error == "用户取消" {
				s.log.Infow("Task cancelled, skipping storyboard", "storyboard_id", storyboard.ID, "task_id", taskID)
				return  // 立即返回，不执行图片生成
			}
			
			// 再次检查本地取消标志
			if cancelled {
				return
			}
			
			err = s.generateFirstLastFramesForStoryboard(dramaID, &storyboard)
			mu.Lock()
			if err != nil {
				s.log.Warnw("Batch frame generation failed for storyboard", "storyboard_id", storyboard.ID, "error", err)
				failedIDs = append(failedIDs, storyboard.ID)
			}
			completedCount++
			progress := completedCount * 100 / total
			msg := fmt.Sprintf("已完成 %d/%d 个分镜", completedCount, total)
			mu.Unlock()
			_ = s.taskService.UpdateTaskStatus(taskID, "processing", progress, msg)
		}(sb)
	}

	// wait all
	for i := 0; i < batchFramesConcurrency; i++ {
		sem <- struct{}{}
	}

	// 如果已取消，不更新最终结果
	if cancelled {
		return
	}

	successCount := total - len(failedIDs)
	result := BatchGenerateFramesResult{
		Total:     total,
		Success:   successCount,
		Failed:    len(failedIDs),
		FailedIDs: failedIDs,
	}
	_ = s.taskService.UpdateTaskResult(taskID, result)
	s.log.Infow("Batch retry failed frames completed", "task_id", taskID, "total", total, "success", successCount, "failed", len(failedIDs))
}

func (s *BatchService) processBatchFrames(taskID string, episodeID string) {
	_ = s.taskService.UpdateTaskStatus(taskID, "processing", 0, "正在获取分镜列表...")

	var episode models.Episode
	if err := s.db.Where("id = ?", episodeID).First(&episode).Error; err != nil {
		_ = s.taskService.UpdateTaskStatus(taskID, "failed", 0, "剧集不存在")
		return
	}

	var storyboards []models.Storyboard
	if err := s.db.Where("episode_id = ?", episodeID).
		Preload("Background").
		Preload("Characters").
		Order("storyboard_number ASC").
		Find(&storyboards).Error; err != nil {
		_ = s.taskService.UpdateTaskStatus(taskID, "failed", 0, "获取分镜失败")
		return
	}

	total := len(storyboards)
	if total == 0 {
		_ = s.taskService.UpdateTaskResult(taskID, BatchGenerateFramesResult{Total: 0, Success: 0, Failed: 0})
		return
	}

	dramaID := strconv.FormatUint(uint64(episode.DramaID), 10)
	var failedIDs []uint
	var completedCount int
	var mu sync.Mutex
	sem := make(chan struct{}, batchFramesConcurrency)
	cancelled := false

	for _, sb := range storyboards {
		// 检查任务是否已取消
		task, err := s.taskService.GetTask(taskID)
		if err == nil && task.Status == "failed" && task.Error == "用户取消" {
			cancelled = true
			s.log.Infow("Task cancelled, stopping batch frames generation", "task_id", taskID)
			break
		}

		sem <- struct{}{}
		go func(storyboard models.Storyboard) {
			defer func() { <-sem }()
			
			// ⭐ 关键改进：goroutine启动后立即检查任务是否已取消
			task, err := s.taskService.GetTask(taskID)
			if err == nil && task.Status == "failed" && task.Error == "用户取消" {
				s.log.Infow("Task cancelled, skipping storyboard", "storyboard_id", storyboard.ID, "task_id", taskID)
				return  // 立即返回，不执行图片生成
			}
			
			// 再次检查本地取消标志
			if cancelled {
				return
			}
			
			err = s.generateFirstLastFramesForStoryboard(dramaID, &storyboard)
			mu.Lock()
			if err != nil {
				s.log.Warnw("Batch frame generation failed for storyboard", "storyboard_id", storyboard.ID, "error", err)
				failedIDs = append(failedIDs, storyboard.ID)
			}
			completedCount++
			progress := completedCount * 100 / total
			msg := fmt.Sprintf("已完成 %d/%d 个分镜", completedCount, total)
			mu.Unlock()
			
			// 只有在未取消时才更新进度
			if !cancelled {
				_ = s.taskService.UpdateTaskStatus(taskID, "processing", progress, msg)
			}
		}(sb)
	}

	// wait all
	for i := 0; i < batchFramesConcurrency; i++ {
		sem <- struct{}{}
	}

	// 如果已取消，不更新最终结果
	if cancelled {
		return
	}

	successCount := total - len(failedIDs)
	result := BatchGenerateFramesResult{
		Total:     total,
		Success:   successCount,
		Failed:    len(failedIDs),
		FailedIDs: failedIDs,
	}
	_ = s.taskService.UpdateTaskResult(taskID, result)
	s.log.Infow("Batch generate frames completed", "task_id", taskID, "total", total, "success", successCount, "failed", len(failedIDs))
}

// generateFirstLastFramesForStoryboard 为单个分镜生成首帧和尾帧图片
func (s *BatchService) generateFirstLastFramesForStoryboard(dramaID string, sb *models.Storyboard) error {
	// ⭐ 改进1：检查是否已有 completed 状态的记录，如果有则跳过
	var existingFirstImg, existingLastImg models.ImageGeneration
	hasFirst := s.db.Where("storyboard_id = ? AND frame_type = ? AND status = ?", 
		sb.ID, "first", models.ImageStatusCompleted).
		Order("created_at DESC").
		First(&existingFirstImg).Error == nil
	hasLast := s.db.Where("storyboard_id = ? AND frame_type = ? AND status = ?", 
		sb.ID, "last", models.ImageStatusCompleted).
		Order("created_at DESC").
		First(&existingLastImg).Error == nil
	
	// 如果首帧和尾帧都已存在，直接返回
	if hasFirst && hasLast {
		s.log.Infow("Storyboard already has completed first and last frames, skipping", 
			"storyboard_id", sb.ID,
			"first_img_id", existingFirstImg.ID,
			"last_img_id", existingLastImg.ID)
		return nil
	}
	
	// ⭐ 改进2：取消进行中的请求（processing 或 pending 状态）
	// 取消首帧的进行中请求
	var inProgressFirst []models.ImageGeneration
	if err := s.db.Where("storyboard_id = ? AND frame_type = ? AND status IN ?", 
		sb.ID, "first", []models.ImageGenerationStatus{models.ImageStatusProcessing, models.ImageStatusPending}).
		Find(&inProgressFirst).Error; err == nil {
		for _, img := range inProgressFirst {
			cancelMsg := "用户取消（重新生成）"
			s.db.Model(&img).Updates(map[string]interface{}{
				"status":    models.ImageStatusFailed,
				"error_msg": &cancelMsg,
			})
			s.log.Infow("Cancelled in-progress first frame generation", 
				"image_gen_id", img.ID, 
				"storyboard_id", sb.ID)
		}
	}
	
	// 取消尾帧的进行中请求
	var inProgressLast []models.ImageGeneration
	if err := s.db.Where("storyboard_id = ? AND frame_type = ? AND status IN ?", 
		sb.ID, "last", []models.ImageGenerationStatus{models.ImageStatusProcessing, models.ImageStatusPending}).
		Find(&inProgressLast).Error; err == nil {
		for _, img := range inProgressLast {
			cancelMsg := "用户取消（重新生成）"
			s.db.Model(&img).Updates(map[string]interface{}{
				"status":    models.ImageStatusFailed,
				"error_msg": &cancelMsg,
			})
			s.log.Infow("Cancelled in-progress last frame generation", 
				"image_gen_id", img.ID, 
				"storyboard_id", sb.ID)
		}
	}
	
	// ⭐ 改进3：可选地清理旧的失败记录（只保留最近的一条失败记录）
	// 清理首帧的旧失败记录
	var failedFirst []models.ImageGeneration
	if err := s.db.Where("storyboard_id = ? AND frame_type = ? AND status = ?", 
		sb.ID, "first", models.ImageStatusFailed).
		Order("created_at DESC").
		Find(&failedFirst).Error; err == nil && len(failedFirst) > 1 {
		// 保留最新的一条，删除其他的
		for i := 1; i < len(failedFirst); i++ {
			s.db.Delete(&failedFirst[i])
			s.log.Infow("Deleted old failed first frame record", 
				"image_gen_id", failedFirst[i].ID, 
				"storyboard_id", sb.ID)
		}
	}
	
	// 清理尾帧的旧失败记录
	var failedLast []models.ImageGeneration
	if err := s.db.Where("storyboard_id = ? AND frame_type = ? AND status = ?", 
		sb.ID, "last", models.ImageStatusFailed).
		Order("created_at DESC").
		Find(&failedLast).Error; err == nil && len(failedLast) > 1 {
		// 保留最新的一条，删除其他的
		for i := 1; i < len(failedLast); i++ {
			s.db.Delete(&failedLast[i])
			s.log.Infow("Deleted old failed last frame record", 
				"image_gen_id", failedLast[i].ID, 
				"storyboard_id", sb.ID)
		}
	}
	
	firstPrompt, lastPrompt, err := s.getOrGenerateFramePrompts(sb)
	if err != nil {
		return err
	}
	refs := s.buildReferenceImages(sb)

	ftFirst := "first"
	ftLast := "last"
	reqFirst := &GenerateImageRequest{
		StoryboardID:    &sb.ID,
		DramaID:         dramaID,
		ImageType:       string(models.ImageTypeStoryboard),
		FrameType:       &ftFirst,
		Prompt:          firstPrompt,
		ReferenceImages: refs,
	}
	reqLast := &GenerateImageRequest{
		StoryboardID:    &sb.ID,
		DramaID:         dramaID,
		ImageType:       string(models.ImageTypeStoryboard),
		FrameType:       &ftLast,
		Prompt:          lastPrompt,
		ReferenceImages: refs,
	}

	// 生成首帧图片并等待完成（如果还没有）
	if !hasFirst {
		firstImgGen, err := s.imageGenService.GenerateImage(reqFirst)
		if err != nil {
			return fmt.Errorf("首帧: %w", err)
		}
		if err := s.waitForImageGenerationComplete(firstImgGen.ID); err != nil {
			return fmt.Errorf("首帧生成失败: %w", err)
		}
	}

	// 生成尾帧图片并等待完成（如果还没有）
	if !hasLast {
		lastImgGen, err := s.imageGenService.GenerateImage(reqLast)
		if err != nil {
			return fmt.Errorf("尾帧: %w", err)
		}
		if err := s.waitForImageGenerationComplete(lastImgGen.ID); err != nil {
			return fmt.Errorf("尾帧生成失败: %w", err)
		}
	}

	return nil
}

// waitForImageGenerationComplete 等待图片生成完成
func (s *BatchService) waitForImageGenerationComplete(imageGenID uint) error {
	const pollInterval = 3 * time.Second
	const maxWait = 120 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		var imgGen models.ImageGeneration
		if err := s.db.First(&imgGen, imageGenID).Error; err != nil {
			return err
		}

		switch imgGen.Status {
		case models.ImageStatusCompleted:
			return nil
		case models.ImageStatusFailed:
			return fmt.Errorf("图片生成失败")
		case models.ImageStatusPending, models.ImageStatusProcessing:
			// 继续等待
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("图片生成超时")
}

func (s *BatchService) getOrGenerateFramePrompts(sb *models.Storyboard) (firstPrompt, lastPrompt string, err error) {
	var fps []models.FramePrompt
	if err := s.db.Where("storyboard_id = ? AND frame_type IN ?", sb.ID, []string{"first", "last"}).Find(&fps).Error; err != nil {
		return "", "", err
	}
	for _, fp := range fps {
		if fp.FrameType == "first" {
			firstPrompt = fp.Prompt
		} else if fp.FrameType == "last" {
			lastPrompt = fp.Prompt
		}
	}
	fallback := ""
	if sb.Action != nil {
		fallback = *sb.Action
	}
	if sb.Description != nil && fallback == "" {
		fallback = *sb.Description
	}
	if firstPrompt == "" {
		firstPrompt, err = s.ensureFramePrompt(strconv.FormatUint(uint64(sb.ID), 10), "first", fallback)
		if err != nil {
			return "", "", fmt.Errorf("首帧提示词: %w", err)
		}
	}
	if lastPrompt == "" {
		lastPrompt, err = s.ensureFramePrompt(strconv.FormatUint(uint64(sb.ID), 10), "last", fallback)
		if err != nil {
			return "", "", fmt.Errorf("尾帧提示词: %w", err)
		}
	}
	return firstPrompt, lastPrompt, nil
}

func (s *BatchService) ensureFramePrompt(storyboardID, frameType, fallback string) (string, error) {
	taskID, err := s.framePromptSvc.GenerateFramePrompt(GenerateFramePromptRequest{
		StoryboardID: storyboardID,
		FrameType:    FrameType(frameType),
	}, "")
	if err != nil {
		return fallback, err
	}
	deadline := time.Now().Add(framePromptPollMaxWait)
	for time.Now().Before(deadline) {
		time.Sleep(framePromptPollInterval)
		task, err := s.taskService.GetTask(taskID)
		if err != nil {
			continue
		}
		if task.Status == "failed" {
			return fallback, nil
		}
		if task.Status == "completed" && task.Result != "" {
			var resultMap map[string]interface{}
			if json.Unmarshal([]byte(task.Result), &resultMap) != nil {
				continue
			}
			resp, _ := resultMap["response"].(map[string]interface{})
			if resp == nil {
				continue
			}
			sf, _ := resp["single_frame"].(map[string]interface{})
			if sf != nil {
				if p, ok := sf["prompt"].(string); ok && p != "" {
					return p, nil
				}
			}
		}
	}
	return fallback, nil
}

func (s *BatchService) buildReferenceImages(sb *models.Storyboard) []string {
	var refs []string
	if sb.Background != nil && sb.Background.LocalPath != nil && *sb.Background.LocalPath != "" {
		refs = append(refs, *sb.Background.LocalPath)
	}
	for _, c := range sb.Characters {
		if c.LocalPath != nil && *c.LocalPath != "" {
			refs = append(refs, *c.LocalPath)
		}
	}
	return refs
}

// BatchGenerateVideosResult 一键出片结果
type BatchGenerateVideosResult struct {
	Total    int   `json:"total"`
	Success  int   `json:"success"`
	Failed   int   `json:"failed"`
	FailedIDs []uint `json:"failed_storyboard_ids,omitempty"`
}

// BatchGenerateVideos 创建一键出片任务并立即返回 task_id
func (s *BatchService) BatchGenerateVideos(episodeID string, model string) (string, error) {
	task, err := s.taskService.CreateTask("batch_generate_videos", episodeID)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}
	go s.processBatchVideos(task.ID, episodeID, model)
	s.log.Infow("Batch generate videos task created", "task_id", task.ID, "episode_id", episodeID, "model", model)
	return task.ID, nil
}

func (s *BatchService) processBatchVideos(taskID string, episodeID string, model string) {
	_ = s.taskService.UpdateTaskStatus(taskID, "processing", 0, "正在获取分镜列表...")

	var episode models.Episode
	if err := s.db.Where("id = ?", episodeID).First(&episode).Error; err != nil {
		_ = s.taskService.UpdateTaskStatus(taskID, "failed", 0, "剧集不存在")
		return
	}

	var storyboards []models.Storyboard
	if err := s.db.Where("episode_id = ?", episodeID).Order("storyboard_number ASC").Find(&storyboards).Error; err != nil {
		_ = s.taskService.UpdateTaskStatus(taskID, "failed", 0, "获取分镜失败")
		return
	}

	total := len(storyboards)
	if total == 0 {
		_ = s.taskService.UpdateTaskResult(taskID, BatchGenerateVideosResult{Total: 0, Success: 0, Failed: 0})
		return
	}

	dramaID := strconv.FormatUint(uint64(episode.DramaID), 10)
	provider := "doubao"
	if model != "" {
		provider = extractProviderFromModelName(model)
	}
	duration := 5

	var failedIDs []uint
	var completedCount int
	var mu sync.Mutex
	sem := make(chan struct{}, batchVideosConcurrency)
	cancelled := false

	for _, sb := range storyboards {
		// 检查任务是否已取消
		task, err := s.taskService.GetTask(taskID)
		if err == nil && task.Status == "failed" && task.Error == "用户取消" {
			cancelled = true
			s.log.Infow("Task cancelled, stopping batch videos generation", "task_id", taskID)
			break
		}

		sem <- struct{}{}
		go func(storyboard models.Storyboard) {
			defer func() { <-sem }()
			
			// ⭐ 关键改进：goroutine启动后立即检查任务是否已取消
			task, err := s.taskService.GetTask(taskID)
			if err == nil && task.Status == "failed" && task.Error == "用户取消" {
				s.log.Infow("Task cancelled, skipping storyboard", "storyboard_id", storyboard.ID, "task_id", taskID)
				return  // 立即返回，不执行视频生成
			}
			
			// 再次检查本地取消标志
			if cancelled {
				return
			}
			
			// ⭐ 改进：检查是否已有正在处理的视频生成任务
			if existingVideoGen, exists := s.hasProcessingVideoGeneration(storyboard.ID); exists {
				s.log.Infow("Found existing video generation, waiting", 
					"storyboard_id", storyboard.ID, 
					"video_gen_id", existingVideoGen.ID,
					"task_id", existingVideoGen.TaskID)
				
				if err := s.waitForVideoGeneration(existingVideoGen.ID); err == nil {
					// 任务完成
					mu.Lock()
					completedCount++
					progress := completedCount * 100 / total
					msg := fmt.Sprintf("已完成 %d/%d 个分镜", completedCount, total)
					mu.Unlock()
					
					if !cancelled {
						_ = s.taskService.UpdateTaskStatus(taskID, "processing", progress, msg)
					}
					return
				}
				// 任务失败，继续执行下面的逻辑
				s.log.Warnw("Existing video generation task failed, will create new one", 
					"storyboard_id", storyboard.ID, 
					"video_gen_id", existingVideoGen.ID)
			}
			
			err = s.generateVideoForStoryboard(dramaID, &storyboard, model, provider, duration)
			mu.Lock()
			if err != nil {
				s.log.Warnw("Batch video generation failed for storyboard", "storyboard_id", storyboard.ID, "error", err)
				failedIDs = append(failedIDs, storyboard.ID)
			}
			completedCount++
			progress := completedCount * 100 / total
			msg := fmt.Sprintf("已完成 %d/%d 个分镜", completedCount, total)
			mu.Unlock()
			
			// 只有在未取消时才更新进度
			if !cancelled {
				_ = s.taskService.UpdateTaskStatus(taskID, "processing", progress, msg)
			}
		}(sb)
	}

	for i := 0; i < batchVideosConcurrency; i++ {
		sem <- struct{}{}
	}

	// 如果已取消，不更新最终结果
	if cancelled {
		return
	}

	successCount := total - len(failedIDs)
	result := BatchGenerateVideosResult{
		Total:     total,
		Success:   successCount,
		Failed:    len(failedIDs),
		FailedIDs: failedIDs,
	}
	_ = s.taskService.UpdateTaskResult(taskID, result)
	s.log.Infow("Batch generate videos completed", "task_id", taskID, "total", total, "success", successCount, "failed", len(failedIDs))
}

func extractProviderFromModelName(model string) string {
	if model == "" {
		return "doubao"
	}
	if len(model) >= 6 && (model[:6] == "openai" || model[:6] == "OpenAI") {
		return "openai"
	}
	if len(model) >= 7 && model[:7] == "chatfire" {
		return "chatfire"
	}
	return "doubao"
}

// hasProcessingVideoGeneration 检查是否有正在处理的视频生成任务
// 判断条件：
// 1. 状态是 processing（说明已经提交到API了）
// 2. 有 task_id 且不为空（说明API已经返回了任务ID）
// 3. 创建时间在最近1小时内（避免处理过期的任务）
func (s *BatchService) hasProcessingVideoGeneration(storyboardID uint) (*models.VideoGeneration, bool) {
	var videoGen models.VideoGeneration
	
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	
	err := s.db.Where("storyboard_id = ? AND status = ? AND task_id IS NOT NULL AND task_id != '' AND created_at > ?", 
		storyboardID, 
		models.VideoStatusProcessing,
		oneHourAgo).
		Order("created_at DESC").
		First(&videoGen).Error
	
	if err == nil {
		return &videoGen, true
	}
	
	return nil, false
}

// waitForVideoGeneration 等待视频生成完成（与后台 goroutine 超时时间对齐）
// 超时时间：50分钟（与 pollTaskStatus 的超时时间一致）
func (s *BatchService) waitForVideoGeneration(videoGenID uint) error {
	const maxWaitTime = 50 * time.Minute  // 与 pollTaskStatus 的超时时间一致
	const pollInterval = 5 * time.Second
	deadline := time.Now().Add(maxWaitTime)
	
	for time.Now().Before(deadline) {
		var videoGen models.VideoGeneration
		if err := s.db.First(&videoGen, videoGenID).Error; err != nil {
			return err
		}
		
		if videoGen.Status == models.VideoStatusCompleted {
			return nil
		}
		
		if videoGen.Status == models.VideoStatusFailed {
			errorMsg := "视频生成失败"
			if videoGen.ErrorMsg != nil {
				errorMsg = fmt.Sprintf("视频生成失败: %s", *videoGen.ErrorMsg)
			}
			return fmt.Errorf(errorMsg)
		}
		
		// 如果状态仍然是 processing，继续等待
		time.Sleep(pollInterval)
	}
	
	return fmt.Errorf("视频生成超时（等待了 %d 分钟）", int(maxWaitTime.Minutes()))
}

func (s *BatchService) generateVideoForStoryboard(dramaID string, sb *models.Storyboard, model, provider string, duration int) error {
	// ⭐ 改进：检查是否已有正在处理的任务
	if existingVideoGen, exists := s.hasProcessingVideoGeneration(sb.ID); exists {
		s.log.Infow("Found existing video generation task, waiting for completion", 
			"storyboard_id", sb.ID, 
			"video_gen_id", existingVideoGen.ID,
			"task_id", existingVideoGen.TaskID,
			"created_at", existingVideoGen.CreatedAt)
		
		// 等待现有任务完成
		if err := s.waitForVideoGeneration(existingVideoGen.ID); err == nil {
			return nil  // 任务已完成
		} else {
			// 任务失败或超时，继续执行下面的逻辑创建新任务
			s.log.Warnw("Existing video generation task failed or timed out, will create new one", 
				"storyboard_id", sb.ID, 
				"video_gen_id", existingVideoGen.ID,
				"error", err)
		}
	}
	
	var firstImg, lastImg models.ImageGeneration
	if err := s.db.Where("storyboard_id = ? AND frame_type = ? AND status = ?", sb.ID, "first", models.ImageStatusCompleted).
		Order("created_at DESC").First(&firstImg).Error; err != nil {
		return fmt.Errorf("缺少首帧图片")
	}
	if err := s.db.Where("storyboard_id = ? AND frame_type = ? AND status = ?", sb.ID, "last", models.ImageStatusCompleted).
		Order("created_at DESC").First(&lastImg).Error; err != nil {
		return fmt.Errorf("缺少尾帧图片")
	}

	prompt := ""
	if sb.VideoPrompt != nil {
		prompt = *sb.VideoPrompt
	}
	if prompt == "" && sb.Action != nil {
		prompt = *sb.Action
	}
	if prompt == "" && sb.Description != nil {
		prompt = *sb.Description
	}
	if len(prompt) < 5 {
		return fmt.Errorf("分镜描述过短")
	}

	req := &GenerateVideoRequest{
		DramaID:               dramaID,
		StoryboardID:          &sb.ID,
		ReferenceMode:         "first_last",
		FirstFrameLocalPath:   firstImg.LocalPath,
		FirstFrameURL:         firstImg.ImageURL,
		LastFrameLocalPath:   lastImg.LocalPath,
		LastFrameURL:         lastImg.ImageURL,
		Prompt:                prompt,
		Provider:              provider,
		Model:                 model,
		Duration:              &duration,
	}
	if req.FirstFrameLocalPath == nil && req.FirstFrameURL != nil {
		req.FirstFrameLocalPath = nil
	}
	if req.LastFrameLocalPath == nil && req.LastFrameURL != nil {
		req.LastFrameLocalPath = nil
	}

	// 调用视频生成服务
	videoGen, err := s.videoGenService.GenerateVideo(req)
	if err != nil {
		return err
	}

	// ⭐ 改进：等待时间与后台 goroutine 的超时时间对齐（50分钟）
	const maxWaitTime = 50 * time.Minute  // 与 pollTaskStatus 的超时时间一致
	const pollInterval = 5 * time.Second
	deadline := time.Now().Add(maxWaitTime)

	for time.Now().Before(deadline) {
		var currentVideoGen models.VideoGeneration
		if err := s.db.First(&currentVideoGen, videoGen.ID).Error; err != nil {
			return err
		}

		if currentVideoGen.Status == models.VideoStatusCompleted {
			return nil
		}

		if currentVideoGen.Status == models.VideoStatusFailed {
			errorMsg := "视频生成失败"
			if currentVideoGen.ErrorMsg != nil {
				errorMsg = fmt.Sprintf("视频生成失败: %s", *currentVideoGen.ErrorMsg)
			}
			return fmt.Errorf(errorMsg)
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("视频生成超时（等待了 %d 分钟）", int(maxWaitTime.Minutes()))
}

// BatchGenerateEpisodeVideo 创建一键章节视频任务（生图→出片→合成）
func (s *BatchService) BatchGenerateEpisodeVideo(episodeID string, model string, dramaID string) (string, error) {
	task, err := s.taskService.CreateTask("batch_generate_episode_video", episodeID)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}

	go s.processEpisodeVideo(task.ID, episodeID, model, dramaID)

	s.log.Infow("Batch generate episode video task created",
		"task_id", task.ID,
		"episode_id", episodeID,
		"model", model)

	return task.ID, nil
}

// RetryEpisodeVideoPhase 重试一键章节视频的某个阶段
// resetProgress: true 表示全部重试（重置进度），false 表示继续执行（从失败处继续）
func (s *BatchService) RetryEpisodeVideoPhase(taskID string, phase string, resetProgress bool) (string, error) {
	oldTask, err := s.taskService.GetTask(taskID)
	if err != nil {
		return "", fmt.Errorf("原任务不存在")
	}

	var progress EpisodeVideoTaskProgress
	if oldTask.Result != "" {
		json.Unmarshal([]byte(oldTask.Result), &progress)
	}

	episodeID := oldTask.ResourceID
	
	// ⭐ 改进：恢复原任务状态，而不是创建新任务
	currentProgress := 0
	progressMessage := "正在继续执行..."
	
	if resetProgress {
		// 全部重试：重置进度为0，重置已完成的分镜数
		currentProgress = 0
		progress.CompletedFrames = 0
		progress.CompletedVideos = 0
		progress.FailedStoryboards = []int{}
		progress.Phase = "frames"
		progressMessage = "阶段1/3：重新开始生图..."
	} else {
		// 继续执行：基于已完成的部分计算进度
		if phase == "frames" {
			if progress.CompletedFrames > 0 && progress.TotalStoryboards > 0 {
				// 阶段1：已完成的分镜数 / 总分镜数 * 33%
				// 使用浮点数计算后再取整，确保精度
				currentProgress = int(float64(progress.CompletedFrames) * 33.0 / float64(progress.TotalStoryboards))
			}
			progressMessage = "阶段1/3：继续生图..."
		} else if phase == "videos" {
			if progress.CompletedVideos > 0 && progress.TotalStoryboards > 0 {
				// 阶段2：33% + (已完成的分镜数 / 总分镜数 * 33%)
				// 使用浮点数计算后再取整，确保精度
				currentProgress = 33 + int(float64(progress.CompletedVideos) * 33.0 / float64(progress.TotalStoryboards))
			} else if progress.CompletedFrames > 0 && progress.TotalStoryboards > 0 {
				// 阶段1已完成，但阶段2还没开始，从33%开始
				currentProgress = 33
			} else {
				currentProgress = 33  // 阶段1已完成（即使没有 CompletedFrames 信息）
			}
			progressMessage = "阶段2/3：继续出片..."
		} else if phase == "merge" {
			// 阶段3：如果阶段2已完成，从66%开始；否则根据已完成的分镜数计算
			if progress.CompletedVideos > 0 && progress.TotalStoryboards > 0 {
				// 检查是否所有分镜都已完成出片
				if progress.CompletedVideos >= progress.TotalStoryboards {
					currentProgress = 66  // 阶段1和2已完成
				} else {
					// 阶段2未完全完成，计算当前进度
					currentProgress = 33 + int(float64(progress.CompletedVideos) * 33.0 / float64(progress.TotalStoryboards))
				}
			} else {
				currentProgress = 66  // 假设阶段1和2已完成
			}
			progressMessage = "阶段3/3：继续合成..."
		}
	}
	
	// 恢复原任务状态
	if err := s.taskService.ResumeTask(taskID, currentProgress, progressMessage); err != nil {
		return "", fmt.Errorf("恢复任务失败: %w", err)
	}

	go func() {
		if resetProgress || phase == "frames" {
			// 全部重试：从头开始
			// 继续执行阶段1：从失败分镜开始
			if !resetProgress && (progress.CompletedFrames > 0 || len(progress.FailedStoryboards) > 0) {
				s.processEpisodeVideoFromFrames(taskID, episodeID, "", "", &progress)
			} else {
				s.processEpisodeVideo(taskID, episodeID, "", "")
			}
		} else if phase == "videos" {
			s.processEpisodeVideoFromVideos(taskID, episodeID, "", "", &progress)
		} else if phase == "merge" {
			s.processEpisodeVideoFromMerge(taskID, episodeID, "", "")
		}
	}()

	// ⭐ 改进：返回原任务ID，而不是新任务ID
	return taskID, nil
}

// CancelEpisodeVideoTask 取消一键章节视频任务
func (s *BatchService) CancelEpisodeVideoTask(taskID string) error {
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.Status != "pending" && task.Status != "processing" {
		return fmt.Errorf("task is not cancellable")
	}

	var progress EpisodeVideoTaskProgress
	if task.Result != "" {
		json.Unmarshal([]byte(task.Result), &progress)
	}
	progress.Cancelled = true

	result := EpisodeVideoTaskResult{
		Success:         false,
		CurrentPhase:    "cancelled",
		PhaseMessage:    "已取消",
		TotalStoryboards: progress.TotalStoryboards,
		CompletedFrames: progress.CompletedFrames,
		CompletedVideos: progress.CompletedVideos,
	}
	resultJSON, _ := json.Marshal(result)

	s.db.Model(&models.AsyncTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":   "completed",
		"progress": 0,
		"message":  "已取消",
		"result":   string(resultJSON),
	})

	return nil
}

// processEpisodeVideo 异步执行一键章节视频流程（完整流程）
func (s *BatchService) processEpisodeVideo(taskID string, episodeID string, model string, dramaID string) {
	var episode models.Episode
	if err := s.db.Preload("Storyboards").Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("剧集不存在"))
		return
	}

	storyboards := episode.Storyboards
	if len(storyboards) == 0 {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("没有分镜"))
		return
	}

	sort.Slice(storyboards, func(i, j int) bool {
		return storyboards[i].StoryboardNumber < storyboards[j].StoryboardNumber
	})

	total := len(storyboards)
	dramaIDStr := strconv.FormatUint(uint64(episode.DramaID), 10)
	if dramaID != "" {
		dramaIDStr = dramaID
	}

	progress := EpisodeVideoTaskProgress{
		Phase:             "frames",
		TotalStoryboards:  total,
		CurrentStoryboard: 0,
		CompletedFrames:   0,
		CompletedVideos:   0,
		Cancelled:         false,
		FailedStoryboards: []int{},
	}

	s.updateEpisodeVideoProgress(taskID, &progress, "阶段1/3：正在生图...", 0)

	failedInFrames := false
	for i, sb := range storyboards {
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		progress.CurrentStoryboard = sb.StoryboardNumber
		msg := fmt.Sprintf("阶段1/3：正在生图（%d/%d）- 分镜%d", i+1, total, sb.StoryboardNumber)
		s.updateEpisodeVideoProgress(taskID, &progress, msg, (i*100)/(total*3))

		err := s.generateFirstLastFramesForStoryboard(dramaIDStr, &sb)
		if err != nil {
			s.log.Warnw("Frame generation failed for storyboard",
				"storyboard_id", sb.ID,
				"storyboard_number", sb.StoryboardNumber,
				"error", err)
			progress.FailedStoryboards = append(progress.FailedStoryboards, sb.StoryboardNumber)
			failedInFrames = true
			// ⭐ 改进：不 break，继续处理剩余分镜
			continue
		}

		progress.CompletedFrames = i + 1
	}

	// ⭐ 改进：即使有失败，如果已完成部分分镜，也继续后续阶段
	if failedInFrames && progress.CompletedFrames == 0 {
		// 如果所有分镜都失败了，直接返回错误
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("所有分镜生图失败"))
		return
	}

	s.updateEpisodeVideoProgress(taskID, &progress, "生图完成，等待3秒...", 33)
	time.Sleep(3 * time.Second)

	if s.checkEpisodeVideoCancelled(taskID, &progress) {
		s.updateEpisodeVideoCancelled(taskID, &progress)
		return
	}

	progress.Phase = "videos"
	s.updateEpisodeVideoProgress(taskID, &progress, "阶段2/3：正在出片...", 33)

	provider := "doubao"
	if model != "" {
		provider = extractProviderFromModelName(model)
	}
	duration := 5
	failedInVideos := false

	for i, sb := range storyboards {
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		progress.CurrentStoryboard = sb.StoryboardNumber
		msg := fmt.Sprintf("阶段2/3：正在出片（%d/%d）- 分镜%d", i+1, total, sb.StoryboardNumber)
		overallProgress := 33 + (i*33)/(total*3)
		s.updateEpisodeVideoProgress(taskID, &progress, msg, overallProgress)

		// ⭐ 改进：检查是否已有正在处理的视频生成任务
		if existingVideoGen, exists := s.hasProcessingVideoGeneration(sb.ID); exists {
			s.log.Infow("Found existing video generation, waiting", 
				"storyboard_id", sb.ID, 
				"video_gen_id", existingVideoGen.ID)
			
			if err := s.waitForVideoGeneration(existingVideoGen.ID); err == nil {
				// 任务完成
				progress.CompletedVideos = i + 1
				continue
			}
			// 任务失败，继续执行下面的逻辑
			s.log.Warnw("Existing video generation task failed, will create new one", 
				"storyboard_id", sb.ID, 
				"video_gen_id", existingVideoGen.ID)
		}

		err := s.generateVideoForStoryboard(dramaIDStr, &sb, model, provider, duration)
		if err != nil {
			s.log.Warnw("Video generation failed for storyboard",
				"storyboard_id", sb.ID,
				"storyboard_number", sb.StoryboardNumber,
				"error", err)
			progress.FailedStoryboards = append(progress.FailedStoryboards, sb.StoryboardNumber)
			failedInVideos = true
			// ⭐ 改进：不 break，继续处理剩余分镜
			continue
		}

		progress.CompletedVideos = i + 1
	}

	// ⭐ 改进：即使有失败，如果已完成部分分镜，也继续阶段3
	if failedInVideos && progress.CompletedVideos == 0 {
		// 如果所有分镜都失败了，直接返回错误
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("所有分镜出片失败"))
		return
	}

	s.updateEpisodeVideoProgress(taskID, &progress, "出片完成，等待3秒...", 66)
	time.Sleep(3 * time.Second)

	if s.checkEpisodeVideoCancelled(taskID, &progress) {
		s.updateEpisodeVideoCancelled(taskID, &progress)
		return
	}

	progress.Phase = "merge"
	s.updateEpisodeVideoProgress(taskID, &progress, "阶段3/3：正在合成...", 66)

	mergeReq := &OneClickMergeRequest{
		EpisodeID: episodeID,
		DramaID:   dramaIDStr,
		Title:     fmt.Sprintf("%s - 第%d集", episode.Drama.Title, episode.EpisodeNum),
	}

	videoMerge, err := s.videoMergeSvc.OneClickMerge(mergeReq)
	if err != nil {
		s.updateEpisodeVideoError(taskID, &progress, fmt.Sprintf("合成失败: %v", err))
		return
	}

	mergePollInterval := 3 * time.Second

	for {
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		var merge models.VideoMerge
		if err := s.db.First(&merge, videoMerge.ID).Error; err == nil {
			if merge.Status == models.VideoMergeStatusCompleted {
				progress.Phase = "completed"
				result := EpisodeVideoTaskResult{
					Success:         true,
					CurrentPhase:    "completed",
					PhaseMessage:    "全部完成！",
					TotalStoryboards: total,
					CompletedFrames: progress.CompletedFrames,
					CompletedVideos: progress.CompletedVideos,
					MergeID:         &videoMerge.ID,
				}
				s.taskService.UpdateTaskResult(taskID, result)
				s.taskService.UpdateTaskStatus(taskID, "completed", 100, "全部完成！")
				return
			}

			if merge.Status == models.VideoMergeStatusFailed {
				s.updateEpisodeVideoError(taskID, &progress,
					fmt.Sprintf("合成失败: %s", merge.ErrorMsg))
				return
			}
		}

		s.updateEpisodeVideoProgress(taskID, &progress, "阶段3/3：正在合成视频...", 80)
		time.Sleep(mergePollInterval)
	}
}

// processEpisodeVideoFromFrames 从生图阶段开始重试（从失败分镜开始，然后继续后续阶段）
func (s *BatchService) processEpisodeVideoFromFrames(taskID string, episodeID string, model string, dramaID string, previousProgress *EpisodeVideoTaskProgress) {
	var episode models.Episode
	if err := s.db.Preload("Storyboards").Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("剧集不存在"))
		return
	}

	storyboards := episode.Storyboards
	if len(storyboards) == 0 {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("没有分镜"))
		return
	}

	sort.Slice(storyboards, func(i, j int) bool {
		return storyboards[i].StoryboardNumber < storyboards[j].StoryboardNumber
	})

	total := len(storyboards)
	dramaIDStr := strconv.FormatUint(uint64(episode.DramaID), 10)
	if dramaID != "" {
		dramaIDStr = dramaID
	}

	progress := EpisodeVideoTaskProgress{
		Phase:             "frames",
		TotalStoryboards:  total,
		CurrentStoryboard: 0,
		CompletedFrames:   previousProgress.CompletedFrames,  // ⭐ 保留已完成的分镜数
		CompletedVideos:   previousProgress.CompletedVideos,
		Cancelled:         false,
		FailedStoryboards: []int{},
	}

	s.updateEpisodeVideoProgress(taskID, &progress, "阶段1/3：正在生图...", 0)

	failedInFrames := false
	// ⭐ 改进：从已完成的分镜数开始，处理剩余分镜
	for i, sb := range storyboards {
		// 跳过已完成的分镜
		if i < previousProgress.CompletedFrames {
			continue
		}
		
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		progress.CurrentStoryboard = sb.StoryboardNumber
		msg := fmt.Sprintf("阶段1/3：正在生图（%d/%d）- 分镜%d", i+1, total, sb.StoryboardNumber)
		s.updateEpisodeVideoProgress(taskID, &progress, msg, (i*100)/(total*3))

		err := s.generateFirstLastFramesForStoryboard(dramaIDStr, &sb)
		if err != nil {
			s.log.Warnw("Frame generation failed for storyboard",
				"storyboard_id", sb.ID,
				"storyboard_number", sb.StoryboardNumber,
				"error", err)
			progress.FailedStoryboards = append(progress.FailedStoryboards, sb.StoryboardNumber)
			failedInFrames = true
			// ⭐ 改进：不 break，继续处理剩余分镜
			continue
		}

		// 更新已完成数量（基于 previousProgress.CompletedFrames）
		progress.CompletedFrames = previousProgress.CompletedFrames + (i - previousProgress.CompletedFrames + 1)
	}

	// ⭐ 改进：即使有失败，如果已完成部分分镜，也继续后续阶段
	if failedInFrames && progress.CompletedFrames == previousProgress.CompletedFrames {
		// 如果所有剩余分镜都失败了，直接返回错误
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("所有剩余分镜生图失败"))
		return
	}

	// 阶段1完成，继续阶段2和阶段3
	// 复用 processEpisodeVideoFromVideos 的逻辑，但需要传入更新后的 progress
	progress.Phase = "videos"
	s.processEpisodeVideoFromVideos(taskID, episodeID, model, dramaID, &progress)
}

// processEpisodeVideoFromVideos 从视频阶段开始重试（跳过已成功生图的分镜）
func (s *BatchService) processEpisodeVideoFromVideos(taskID string, episodeID string, model string, dramaID string, previousProgress *EpisodeVideoTaskProgress) {
	var episode models.Episode
	if err := s.db.Preload("Storyboards").Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("剧集不存在"))
		return
	}

	storyboards := episode.Storyboards
	if len(storyboards) == 0 {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("没有分镜"))
		return
	}

	sort.Slice(storyboards, func(i, j int) bool {
		return storyboards[i].StoryboardNumber < storyboards[j].StoryboardNumber
	})

	total := len(storyboards)
	dramaIDStr := strconv.FormatUint(uint64(episode.DramaID), 10)
	if dramaID != "" {
		dramaIDStr = dramaID
	}

	progress := EpisodeVideoTaskProgress{
		Phase:             "videos",
		TotalStoryboards:  total,
		CurrentStoryboard: 0,
		CompletedFrames:   previousProgress.CompletedFrames,
		CompletedVideos:   previousProgress.CompletedVideos,  // ⭐ 改进：保留已完成的分镜数
		Cancelled:         false,
		FailedStoryboards: []int{},
	}

	provider := "doubao"
	if model != "" {
		provider = extractProviderFromModelName(model)
	}
	duration := 5
	failedInVideos := false

	// ⭐ 改进：从已完成的分镜数开始，处理剩余分镜
	for i, sb := range storyboards {
		// 跳过已完成的分镜
		if i < previousProgress.CompletedVideos {
			continue
		}
		
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		progress.CurrentStoryboard = sb.StoryboardNumber
		msg := fmt.Sprintf("阶段2/3：正在出片（%d/%d）- 分镜%d", i+1, total, sb.StoryboardNumber)
		overallProgress := 33 + (i*33)/(total*3)
		s.updateEpisodeVideoProgress(taskID, &progress, msg, overallProgress)

		// ⭐ 改进：检查是否已有正在处理的视频生成任务
		if existingVideoGen, exists := s.hasProcessingVideoGeneration(sb.ID); exists {
			s.log.Infow("Found existing video generation, waiting", 
				"storyboard_id", sb.ID, 
				"video_gen_id", existingVideoGen.ID)
			
			if err := s.waitForVideoGeneration(existingVideoGen.ID); err == nil {
				// 任务完成，更新已完成数量（基于 previousProgress.CompletedVideos）
				progress.CompletedVideos = previousProgress.CompletedVideos + (i - previousProgress.CompletedVideos + 1)
				continue
			}
			// 任务失败，继续执行下面的逻辑
			s.log.Warnw("Existing video generation task failed, will create new one", 
				"storyboard_id", sb.ID, 
				"video_gen_id", existingVideoGen.ID)
		}

		err := s.generateVideoForStoryboard(dramaIDStr, &sb, model, provider, duration)
		if err != nil {
			s.log.Warnw("Video generation failed for storyboard",
				"storyboard_id", sb.ID,
				"storyboard_number", sb.StoryboardNumber,
				"error", err)
			progress.FailedStoryboards = append(progress.FailedStoryboards, sb.StoryboardNumber)
			failedInVideos = true
			// ⭐ 改进：不 break，继续处理剩余分镜
			continue
		}

		// 更新已完成数量（基于 previousProgress.CompletedVideos）
		progress.CompletedVideos = previousProgress.CompletedVideos + (i - previousProgress.CompletedVideos + 1)
	}

	// ⭐ 改进：即使有失败，如果已完成部分分镜，也继续阶段3
	if failedInVideos && progress.CompletedVideos == previousProgress.CompletedVideos {
		// 如果所有剩余分镜都失败了，直接返回错误
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("所有剩余分镜出片失败"))
		return
	}

	s.updateEpisodeVideoProgress(taskID, &progress, "出片完成，等待3秒...", 66)
	time.Sleep(3 * time.Second)

	if s.checkEpisodeVideoCancelled(taskID, &progress) {
		s.updateEpisodeVideoCancelled(taskID, &progress)
		return
	}

	progress.Phase = "merge"
	s.updateEpisodeVideoProgress(taskID, &progress, "阶段3/3：正在合成...", 66)

	mergeReq := &OneClickMergeRequest{
		EpisodeID: episodeID,
		DramaID:   dramaIDStr,
		Title:     fmt.Sprintf("%s - 第%d集", episode.Drama.Title, episode.EpisodeNum),
	}

	videoMerge, err := s.videoMergeSvc.OneClickMerge(mergeReq)
	if err != nil {
		s.updateEpisodeVideoError(taskID, &progress, fmt.Sprintf("合成失败: %v", err))
		return
	}

	mergePollInterval := 3 * time.Second

	for {
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		var merge models.VideoMerge
		if err := s.db.First(&merge, videoMerge.ID).Error; err == nil {
			if merge.Status == models.VideoMergeStatusCompleted {
				progress.Phase = "completed"
				result := EpisodeVideoTaskResult{
					Success:         true,
					CurrentPhase:    "completed",
					PhaseMessage:    "全部完成！",
					TotalStoryboards: total,
					CompletedFrames: progress.CompletedFrames,
					CompletedVideos: progress.CompletedVideos,
					MergeID:         &videoMerge.ID,
				}
				s.taskService.UpdateTaskResult(taskID, result)
				s.taskService.UpdateTaskStatus(taskID, "completed", 100, "全部完成！")
				return
			}

			if merge.Status == models.VideoMergeStatusFailed {
				s.updateEpisodeVideoError(taskID, &progress,
					fmt.Sprintf("合成失败: %s", merge.ErrorMsg))
				return
			}
		}

		s.updateEpisodeVideoProgress(taskID, &progress, "阶段3/3：正在合成视频...", 80)
		time.Sleep(mergePollInterval)
	}
}

// processEpisodeVideoFromMerge 从合成阶段开始重试
func (s *BatchService) processEpisodeVideoFromMerge(taskID string, episodeID string, model string, dramaID string) {
	var episode models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("剧集不存在"))
		return
	}

	dramaIDStr := strconv.FormatUint(uint64(episode.DramaID), 10)
	if dramaID != "" {
		dramaIDStr = dramaID
	}

	progress := EpisodeVideoTaskProgress{
		Phase:             "merge",
		TotalStoryboards:  0,
		CurrentStoryboard: 0,
		CompletedFrames:   0,
		CompletedVideos:   0,
		Cancelled:         false,
		FailedStoryboards: []int{},
	}

	s.updateEpisodeVideoProgress(taskID, &progress, "阶段3/3：正在合成...", 66)

	mergeReq := &OneClickMergeRequest{
		EpisodeID: episodeID,
		DramaID:   dramaIDStr,
		Title:     fmt.Sprintf("%s - 第%d集", episode.Drama.Title, episode.EpisodeNum),
	}

	videoMerge, err := s.videoMergeSvc.OneClickMerge(mergeReq)
	if err != nil {
		s.updateEpisodeVideoError(taskID, &progress, fmt.Sprintf("合成失败: %v", err))
		return
	}

	mergePollInterval := 3 * time.Second

	for {
		if s.checkEpisodeVideoCancelled(taskID, &progress) {
			s.updateEpisodeVideoCancelled(taskID, &progress)
			return
		}

		var merge models.VideoMerge
		if err := s.db.First(&merge, videoMerge.ID).Error; err == nil {
			if merge.Status == models.VideoMergeStatusCompleted {
				progress.Phase = "completed"
				result := EpisodeVideoTaskResult{
					Success:         true,
					CurrentPhase:    "completed",
					PhaseMessage:    "全部完成！",
					TotalStoryboards: progress.TotalStoryboards,
					CompletedFrames: progress.CompletedFrames,
					CompletedVideos: progress.CompletedVideos,
					MergeID:         &videoMerge.ID,
				}
				s.taskService.UpdateTaskResult(taskID, result)
				s.taskService.UpdateTaskStatus(taskID, "completed", 100, "全部完成！")
				return
			}

			if merge.Status == models.VideoMergeStatusFailed {
				s.updateEpisodeVideoError(taskID, &progress,
					fmt.Sprintf("合成失败: %s", merge.ErrorMsg))
				return
			}
		}

		s.updateEpisodeVideoProgress(taskID, &progress, "阶段3/3：正在合成视频...", 80)
		time.Sleep(mergePollInterval)
	}
}

// updateEpisodeVideoProgress 更新一键章节视频进度
func (s *BatchService) updateEpisodeVideoProgress(taskID string, progress *EpisodeVideoTaskProgress, message string, overallProgress int) {
	resultJSON, _ := json.Marshal(progress)
	s.db.Model(&models.AsyncTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":   "processing",
		"progress": overallProgress,
		"message":  message,
		"result":   string(resultJSON),
	})
}

// updateEpisodeVideoError 更新一键章节视频错误
func (s *BatchService) updateEpisodeVideoError(taskID string, progress *EpisodeVideoTaskProgress, errMsg string) {
	progress.Phase = "failed"
	result := EpisodeVideoTaskResult{
		Success:           false,
		CurrentPhase:      progress.Phase,
		PhaseMessage:      errMsg,
		TotalStoryboards:  progress.TotalStoryboards,
		CompletedFrames:   progress.CompletedFrames,
		CompletedVideos:   progress.CompletedVideos,
		FailedStoryboards: progress.FailedStoryboards,
		Error:             errMsg,
	}
	s.taskService.UpdateTaskResult(taskID, result)
	s.taskService.UpdateTaskError(taskID, fmt.Errorf(errMsg))
}

// updateEpisodeVideoCancelled 更新一键章节视频为已取消
func (s *BatchService) updateEpisodeVideoCancelled(taskID string, progress *EpisodeVideoTaskProgress) {
	progress.Phase = "cancelled"
	result := EpisodeVideoTaskResult{
		Success:         false,
		CurrentPhase:    "cancelled",
		PhaseMessage:    "已取消",
		TotalStoryboards: progress.TotalStoryboards,
		CompletedFrames: progress.CompletedFrames,
		CompletedVideos: progress.CompletedVideos,
	}
	s.taskService.UpdateTaskResult(taskID, result)
	s.taskService.UpdateTaskStatus(taskID, "completed", 0, "已取消")
}

// checkEpisodeVideoCancelled 检查是否已取消
func (s *BatchService) checkEpisodeVideoCancelled(taskID string, progress *EpisodeVideoTaskProgress) bool {
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		return false
	}

	if task.Result != "" {
		var storedProgress EpisodeVideoTaskProgress
		if json.Unmarshal([]byte(task.Result), &storedProgress) == nil {
			if storedProgress.Cancelled {
				progress.Cancelled = true
				return true
			}
		}
	}

	return false
}

// CancelTask 取消批量任务（一键生图、一键出片）
func (s *BatchService) CancelTask(taskID string) error {
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		s.log.Errorw("Failed to get task for cancellation", "task_id", taskID, "error", err)
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务类型
	if task.Type != "batch_generate_frames" && task.Type != "batch_generate_videos" && task.Type != "batch_retry_failed_frames" {
		s.log.Warnw("Attempted to cancel unsupported task type", "task_id", taskID, "task_type", task.Type)
		return fmt.Errorf("不支持取消此类型的任务（类型: %s）", task.Type)
	}

	// 只允许取消 pending 或 processing 状态的任务
	if task.Status != "pending" && task.Status != "processing" {
		s.log.Warnw("Attempted to cancel task with invalid status", "task_id", taskID, "status", task.Status)
		return fmt.Errorf("任务状态不允许取消（当前状态: %s）", task.Status)
	}

	// ⭐ 修复：保存当前进度信息，以便后续可以继续执行
	var result interface{}
	
	// 如果任务已有结果，尝试解析并保留
	if task.Result != "" {
		var existingResult interface{}
		if json.Unmarshal([]byte(task.Result), &existingResult) == nil {
			result = existingResult
		}
	}
	
	// 如果没有结果，尝试从数据库查询已完成的分镜，构建结果
	if result == nil && task.ResourceID != "" {
		episodeID := task.ResourceID
		
		// 获取该剧集的所有分镜ID
		var storyboardIDs []uint
		if err := s.db.Model(&models.Storyboard{}).Where("episode_id = ?", episodeID).Pluck("id", &storyboardIDs).Error; err == nil && len(storyboardIDs) > 0 {
			if task.Type == "batch_generate_frames" || task.Type == "batch_retry_failed_frames" {
				// 查询已完成生图的分镜（有首帧和尾帧图片）
				var completedStoryboardIDs []uint
				if err := s.db.Model(&models.ImageGeneration{}).
					Where("storyboard_id IN ? AND frame_type IN ? AND image_type = ? AND status = ?", 
						storyboardIDs, []string{"first", "last"}, models.ImageTypeStoryboard, models.ImageStatusCompleted).
					Group("storyboard_id").
					Having("COUNT(DISTINCT frame_type) = 2"). // 既有首帧又有尾帧
					Pluck("storyboard_id", &completedStoryboardIDs).Error; err == nil {
					
					// 计算失败的分镜ID（所有分镜 - 已完成的分镜）
					failedIDs := make([]uint, 0)
					completedMap := make(map[uint]bool)
					for _, id := range completedStoryboardIDs {
						completedMap[id] = true
					}
					for _, id := range storyboardIDs {
						if !completedMap[id] {
							failedIDs = append(failedIDs, id)
						}
					}
					
					result = BatchGenerateFramesResult{
						Total:     len(storyboardIDs),
						Success:   len(completedStoryboardIDs),
						Failed:    len(failedIDs),
						FailedIDs: failedIDs,
					}
				} else {
					// ⭐ 方案1：查询失败，但至少保存所有分镜ID作为需要重新检查的分镜
					// 前端继续执行时，会重新检查每个分镜的状态，只处理真正失败的分镜
					s.log.Warnw("Failed to query completed storyboards, will recheck on continue", 
						"task_id", taskID, 
						"episode_id", episodeID,
						"storyboard_count", len(storyboardIDs),
						"error", err)
					result = BatchGenerateFramesResult{
						Total:     len(storyboardIDs),
						Success:   0,  // 未知，需要重新检查
						Failed:    len(storyboardIDs),
						FailedIDs: storyboardIDs,  // 所有分镜都需要重新检查
					}
				}
			} else if task.Type == "batch_generate_videos" {
				// 查询已完成出片的分镜
				var completedStoryboardIDs []uint
				if err := s.db.Model(&models.VideoGeneration{}).
					Where("storyboard_id IN ? AND status = ?", storyboardIDs, models.VideoStatusCompleted).
					Pluck("storyboard_id", &completedStoryboardIDs).Error; err == nil {
					
					// 计算失败的分镜ID
					failedIDs := make([]uint, 0)
					completedMap := make(map[uint]bool)
					for _, id := range completedStoryboardIDs {
						completedMap[id] = true
					}
					for _, id := range storyboardIDs {
						if !completedMap[id] {
							failedIDs = append(failedIDs, id)
						}
					}
					
					result = BatchGenerateVideosResult{
						Total:     len(storyboardIDs),
						Success:   len(completedStoryboardIDs),
						Failed:    len(failedIDs),
						FailedIDs: failedIDs,
					}
				} else {
					// ⭐ 方案1：查询失败，但至少保存所有分镜ID作为需要重新检查的分镜
					// 前端继续执行时，会重新检查每个分镜的状态，只处理真正失败的分镜
					s.log.Warnw("Failed to query completed videos, will recheck on continue", 
						"task_id", taskID, 
						"episode_id", episodeID,
						"storyboard_count", len(storyboardIDs),
						"error", err)
					result = BatchGenerateVideosResult{
						Total:     len(storyboardIDs),
						Success:   0,  // 未知，需要重新检查
						Failed:    len(storyboardIDs),
						FailedIDs: storyboardIDs,  // 所有分镜都需要重新检查
					}
				}
			}
		}
	}
	
	// 如果仍然没有结果，创建空的结果结构
	if result == nil {
		if task.Type == "batch_generate_frames" || task.Type == "batch_retry_failed_frames" {
			result = BatchGenerateFramesResult{
				Total:     0,
				Success:   0,
				Failed:    0,
				FailedIDs: []uint{},
			}
		} else if task.Type == "batch_generate_videos" {
			result = BatchGenerateVideosResult{
				Total:     0,
				Success:   0,
				Failed:    0,
				FailedIDs: []uint{},
			}
		}
	}

	// 更新任务状态为已取消，并保存结果
	now := time.Now()
	resultJSON, _ := json.Marshal(result)
	
	// ⭐ 改进：保留当前进度，而不是重置为0
	// 这样用户可以看到取消时的进度，继续执行时也能正确显示
	currentProgress := task.Progress
	if currentProgress < 0 {
		currentProgress = 0
	}
	if currentProgress > 100 {
		currentProgress = 100
	}
	
	if err := s.db.Model(&models.AsyncTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":       "failed",
		"progress":     currentProgress,  // ⭐ 保留当前进度，而不是设置为0
		"message":      "已取消",
		"error":        "用户取消",
		"result":       string(resultJSON), // ⭐ 保存结果
		"completed_at": &now,
		"updated_at":   time.Now(),
	}).Error; err != nil {
		s.log.Errorw("Failed to update task status to cancelled", "task_id", taskID, "error", err)
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	s.log.Infow("Task cancelled successfully", "task_id", taskID, "task_type", task.Type)
	return nil
}
