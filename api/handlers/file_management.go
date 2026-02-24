package handlers

import (
	"strings"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FileManagementHandler 处理文件管理API请求
type FileManagementHandler struct {
	fileService *services.FileManagementService
	log         *logger.Logger
}

// NewFileManagementHandler 创建新的文件管理处理器实例
func NewFileManagementHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger, localStore *storage.LocalStorage) *FileManagementHandler {
	return &FileManagementHandler{
		fileService: services.NewFileManagementService(db, cfg, log, localStore),
		log:         log,
	}
}

// ListImages 列出所有图片文件
func (h *FileManagementHandler) ListImages(c *gin.Context) {
	images, err := h.fileService.ListImages()
	if err != nil {
		h.log.Errorw("Failed to list images", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	// 转换为响应格式
	var responseImages []map[string]interface{}
	for _, img := range images {
		// 生成可访问的URL
		url := strings.Replace(img.Path, "data/storage/", "/static/", 1)
		responseImages = append(responseImages, map[string]interface{}{
			"id":               img.ID,
			"filename":         img.Filename,
			"path":             img.Path,
			"url":              url,
			"size":             img.Size,
			"is_associated":    img.IsAssociated,
			"associated_type":  img.AssociatedType,
			"associated_id":    img.AssociatedID,
			"created_at":       img.CreatedAt,
		})
	}

	response.Success(c, responseImages)
}

// ListVideos 列出所有视频文件
func (h *FileManagementHandler) ListVideos(c *gin.Context) {
	videos, err := h.fileService.ListVideos()
	if err != nil {
		h.log.Errorw("Failed to list videos", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	// 转换为响应格式
	var responseVideos []map[string]interface{}
	for _, vid := range videos {
		// 生成可访问的URL
		url := strings.Replace(vid.Path, "data/storage/", "/static/", 1)
		responseVideos = append(responseVideos, map[string]interface{}{
			"id":               vid.ID,
			"filename":         vid.Filename,
			"path":             vid.Path,
			"url":              url,
			"size":             vid.Size,
			"is_associated":    vid.IsAssociated,
			"associated_type":  vid.AssociatedType,
			"associated_id":    vid.AssociatedID,
			"created_at":       vid.CreatedAt,
		})
	}

	response.Success(c, responseVideos)
}

// DeleteFile 删除文件
func (h *FileManagementHandler) DeleteFile(c *gin.Context) {
	filePath := c.Param("path")
	if filePath == "" {
		response.BadRequest(c, "文件路径不能为空")
		return
	}

	// 防止路径遍历
	if strings.Contains(filePath, "..") || strings.Contains(filePath, "\\") {
		response.BadRequest(c, "无效的文件路径")
		return
	}

	err := h.fileService.DeleteFile(filePath)
	if err != nil {
		h.log.Errorw("Failed to delete file", "path", filePath, "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteImage 删除图片文件
func (h *FileManagementHandler) DeleteImage(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		response.BadRequest(c, "文件名不能为空")
		return
	}

	err := h.fileService.DeleteImage(filename)
	if err != nil {
		h.log.Errorw("Failed to delete image", "filename", filename, "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteVideo 删除视频文件
func (h *FileManagementHandler) DeleteVideo(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		response.BadRequest(c, "文件名不能为空")
		return
	}

	err := h.fileService.DeleteVideo(filename)
	if err != nil {
		h.log.Errorw("Failed to delete video", "filename", filename, "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// CleanupUnassociatedFiles 清理未关联的文件
func (h *FileManagementHandler) CleanupUnassociatedFiles(c *gin.Context) {
	deletedFiles, err := h.fileService.CleanupUnassociatedFiles()
	if err != nil {
		h.log.Errorw("Failed to cleanup unassociated files", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, map[string]interface{}{
		"deleted_files": deletedFiles,
		"count":         len(deletedFiles),
	})
}
