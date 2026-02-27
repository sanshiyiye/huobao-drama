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
	batchFramesConcurrency  = 2
	batchVideosConcurrency  = 2
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

// BatchRetryFailedFrames 重试失败分镜的首尾帧生成
func (s *BatchService) BatchRetryFailedFrames(episodeID string, failedStoryboardIDs []uint) (string, error) {
	task, err := s.taskService.CreateTask("batch_retry_failed_frames", episodeID)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}
	go s.processRetryFailedFrames(task.ID, episodeID, failedStoryboardIDs)
	s.log.Infow("Batch retry failed frames task created", "task_id", task.ID, "episode_id", episodeID, "failed_storyboard_ids", failedStoryboardIDs)
	return task.ID, nil
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

	for _, sb := range storyboards {
		sem <- struct{}{}
		go func(storyboard models.Storyboard) {
			defer func() { <-sem }()
			err := s.generateFirstLastFramesForStoryboard(dramaID, &storyboard)
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

	for _, sb := range storyboards {
		sem <- struct{}{}
		go func(storyboard models.Storyboard) {
			defer func() { <-sem }()
			err := s.generateFirstLastFramesForStoryboard(dramaID, &storyboard)
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

	// 生成首帧图片并等待完成
	firstImgGen, err := s.imageGenService.GenerateImage(reqFirst)
	if err != nil {
		return fmt.Errorf("首帧: %w", err)
	}
	if err := s.waitForImageGenerationComplete(firstImgGen.ID); err != nil {
		return fmt.Errorf("首帧生成失败: %w", err)
	}

	// 生成尾帧图片并等待完成
	lastImgGen, err := s.imageGenService.GenerateImage(reqLast)
	if err != nil {
		return fmt.Errorf("尾帧: %w", err)
	}
	if err := s.waitForImageGenerationComplete(lastImgGen.ID); err != nil {
		return fmt.Errorf("尾帧生成失败: %w", err)
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

	for _, sb := range storyboards {
		sem <- struct{}{}
		go func(storyboard models.Storyboard) {
			defer func() { <-sem }()
			err := s.generateVideoForStoryboard(dramaID, &storyboard, model, provider, duration)
			mu.Lock()
			if err != nil {
				s.log.Warnw("Batch video generation failed for storyboard", "storyboard_id", storyboard.ID, "error", err)
				failedIDs = append(failedIDs, storyboard.ID)
			}
			completedCount++
			progress := completedCount * 100 / total
			msg := fmt.Sprintf("已完成 %d/%d 个分镜", completedCount, total)
			mu.Unlock()
			_ = s.taskService.UpdateTaskStatus(taskID, "processing", progress, msg)
		}(sb)
	}

	for i := 0; i < batchVideosConcurrency; i++ {
		sem <- struct{}{}
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

func (s *BatchService) generateVideoForStoryboard(dramaID string, sb *models.Storyboard, model, provider string, duration int) error {
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

	// 等待视频生成完成
	const maxWaitTime = 5 * time.Minute
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
			return fmt.Errorf("视频生成失败")
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("视频生成超时")
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

// RetryEpisodeVideoPhase 重试一键章节视频的某个阶段（仅重试失败的分镜）
func (s *BatchService) RetryEpisodeVideoPhase(taskID string, phase string) (string, error) {
	oldTask, err := s.taskService.GetTask(taskID)
	if err != nil {
		return "", fmt.Errorf("原任务不存在")
	}

	var progress EpisodeVideoTaskProgress
	if oldTask.Result != "" {
		json.Unmarshal([]byte(oldTask.Result), &progress)
	}

	episodeID := oldTask.ResourceID
	newTask, err := s.taskService.CreateTask("batch_generate_episode_video", episodeID)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}

	go func() {
		if phase == "frames" {
			s.processEpisodeVideo(newTask.ID, episodeID, "", "")
		} else if phase == "videos" {
			s.processEpisodeVideoFromVideos(newTask.ID, episodeID, "", "", &progress)
		} else if phase == "merge" {
			s.processEpisodeVideoFromMerge(newTask.ID, episodeID, "", "")
		}
	}()

	return newTask.ID, nil
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
			break
		}

		progress.CompletedFrames = i + 1
	}

	if failedInFrames {
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("分镜%d生图失败", progress.FailedStoryboards[len(progress.FailedStoryboards)-1]))
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

		err := s.generateVideoForStoryboard(dramaIDStr, &sb, model, provider, duration)
		if err != nil {
			s.log.Warnw("Video generation failed for storyboard",
				"storyboard_id", sb.ID,
				"storyboard_number", sb.StoryboardNumber,
				"error", err)
			progress.FailedStoryboards = append(progress.FailedStoryboards, sb.StoryboardNumber)
			failedInVideos = true
			break
		}

		progress.CompletedVideos = i + 1
	}

	if failedInVideos {
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("分镜%d出片失败", progress.FailedStoryboards[len(progress.FailedStoryboards)-1]))
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
		CompletedVideos:   0,
		Cancelled:         false,
		FailedStoryboards: []int{},
	}

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

		err := s.generateVideoForStoryboard(dramaIDStr, &sb, model, provider, duration)
		if err != nil {
			s.log.Warnw("Video generation failed for storyboard",
				"storyboard_id", sb.ID,
				"storyboard_number", sb.StoryboardNumber,
				"error", err)
			progress.FailedStoryboards = append(progress.FailedStoryboards, sb.StoryboardNumber)
			failedInVideos = true
			break
		}

		progress.CompletedVideos = i + 1
	}

	if failedInVideos {
		s.updateEpisodeVideoError(taskID, &progress,
			fmt.Sprintf("分镜%d出片失败", progress.FailedStoryboards[len(progress.FailedStoryboards)-1]))
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
