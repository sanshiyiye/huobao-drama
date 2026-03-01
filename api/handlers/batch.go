package handlers

import (
	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type BatchHandler struct {
	batchService *services.BatchService
	log          *logger.Logger
}

func NewBatchHandler(batchService *services.BatchService, log *logger.Logger) *BatchHandler {
	return &BatchHandler{
		batchService: batchService,
		log:          log,
	}
}

// BatchGenerateFramesRequest 一键生图请求
type BatchGenerateFramesRequest struct {
	EpisodeID string `json:"episode_id" binding:"required"`
}

// BatchGenerateFrames 一键生图：为剧集下所有分镜生成首尾帧图片，返回 task_id 供轮询
func (h *BatchHandler) BatchGenerateFrames(c *gin.Context) {
	var req BatchGenerateFramesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少 episode_id")
		return
	}

	taskID, err := h.batchService.BatchGenerateFrames(req.EpisodeID)
	if err != nil {
		h.log.Errorw("Batch generate frames failed", "error", err, "episode_id", req.EpisodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID})
}

// BatchGenerateVideosRequest 一键出片请求
type BatchGenerateVideosRequest struct {
	EpisodeID string `json:"episode_id" binding:"required"`
	Model     string `json:"model"` // 视频模型，如 DoubaoSeedance Pro1.5
}

// BatchGenerateVideos 一键出片：为剧集下所有分镜用首尾帧方式生成视频，返回 task_id 供轮询
func (h *BatchHandler) BatchGenerateVideos(c *gin.Context) {
	var req BatchGenerateVideosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少 episode_id")
		return
	}

	taskID, err := h.batchService.BatchGenerateVideos(req.EpisodeID, req.Model)
	if err != nil {
		h.log.Errorw("Batch generate videos failed", "error", err, "episode_id", req.EpisodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID})
}

// BatchRetryFailedFramesRequest 重试失败分镜请求
type BatchRetryFailedFramesRequest struct {
	EpisodeID          string `json:"episode_id" binding:"required"`
	FailedStoryboardIDs []uint `json:"failed_storyboard_ids" binding:"required"` // 失败分镜的ID列表
}

// BatchRetryFailedFrames 重试失败分镜的首尾帧生成：为失败的分镜重新生成首尾帧图片，返回 task_id 供轮询
func (h *BatchHandler) BatchRetryFailedFrames(c *gin.Context) {
	var req BatchRetryFailedFramesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少必要参数")
		return
	}

	taskID, err := h.batchService.BatchRetryFailedFrames(req.EpisodeID, req.FailedStoryboardIDs)
	if err != nil {
		h.log.Errorw("Batch retry failed frames failed", "error", err, "episode_id", req.EpisodeID, "failed_storyboard_ids", req.FailedStoryboardIDs)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID})
}

// GetBatchImageProgress 剧集首尾帧生图进度（只读），供前端轮询真实进度
func (h *BatchHandler) GetBatchImageProgress(c *gin.Context) {
	episodeID := c.Param("episode_id")
	if episodeID == "" {
		response.BadRequest(c, "缺少 episode_id")
		return
	}

	progress, err := h.batchService.GetBatchImageProgress(episodeID)
	if err != nil {
		h.log.Errorw("Get batch image progress failed", "error", err, "episode_id", episodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, progress)
}

// BatchGenerateEpisodeVideoRequest 一键章节视频请求
type BatchGenerateEpisodeVideoRequest struct {
	EpisodeID string `json:"episode_id" binding:"required"`
	DramaID   string `json:"drama_id"`
	Model     string `json:"model"`
}

// RetryEpisodeVideoPhaseRequest 重试阶段请求
type RetryEpisodeVideoPhaseRequest struct {
	TaskID string `json:"task_id" binding:"required"`
	Phase  string `json:"phase" binding:"required"` // frames, videos, merge
}

// CancelEpisodeVideoTaskRequest 取消任务请求
type CancelEpisodeVideoTaskRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

// BatchGenerateEpisodeVideo 一键章节视频：生图→出片→合成，返回 task_id
func (h *BatchHandler) BatchGenerateEpisodeVideo(c *gin.Context) {
	var req BatchGenerateEpisodeVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少必要参数")
		return
	}

	taskID, err := h.batchService.BatchGenerateEpisodeVideo(req.EpisodeID, req.Model, req.DramaID)
	if err != nil {
		h.log.Errorw("Batch generate episode video failed",
			"error", err,
			"episode_id", req.EpisodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID})
}

// RetryEpisodeVideoPhase 重试一键章节视频的某个阶段
func (h *BatchHandler) RetryEpisodeVideoPhase(c *gin.Context) {
	var req RetryEpisodeVideoPhaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少必要参数")
		return
	}

	taskID, err := h.batchService.RetryEpisodeVideoPhase(req.TaskID, req.Phase)
	if err != nil {
		h.log.Errorw("Retry episode video phase failed",
			"error", err,
			"task_id", req.TaskID,
			"phase", req.Phase)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID})
}

// CancelEpisodeVideoTask 取消一键章节视频任务
func (h *BatchHandler) CancelEpisodeVideoTask(c *gin.Context) {
	var req CancelEpisodeVideoTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少 task_id")
		return
	}

	err := h.batchService.CancelEpisodeVideoTask(req.TaskID)
	if err != nil {
		h.log.Errorw("Cancel episode video task failed",
			"error", err,
			"task_id", req.TaskID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "任务已取消"})
}

// CancelTaskRequest 取消任务请求
type CancelTaskRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

// CancelTask 取消批量任务（一键生图、一键出片）
func (h *BatchHandler) CancelTask(c *gin.Context) {
	var req CancelTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少 task_id")
		return
	}

	err := h.batchService.CancelTask(req.TaskID)
	if err != nil {
		h.log.Errorw("Cancel task failed",
			"error", err,
			"task_id", req.TaskID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "取消请求已提交"})
}
