package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

// FileInfo 表示文件信息
type FileInfo struct {
	ID             uint      `json:"id"`
	Filename       string    `json:"filename"`
	Path           string    `json:"path"`
	Size           int64     `json:"size"`
	IsAssociated   bool      `json:"is_associated"`
	AssociatedType string    `json:"associated_type,omitempty"` // 关联类型：image, video, character等
	AssociatedID   uint      `json:"associated_id,omitempty"`   // 关联ID
	CreatedAt      time.Time `json:"created_at"`
}

// FileManagementService 提供文件管理功能
type FileManagementService struct {
	db         *gorm.DB
	cfg        *config.Config
	log        *logger.Logger
	localStore *storage.LocalStorage
}

// NewFileManagementService 创建新的文件管理服务实例
func NewFileManagementService(db *gorm.DB, cfg *config.Config, log *logger.Logger, localStore *storage.LocalStorage) *FileManagementService {
	return &FileManagementService{
		db:         db,
		cfg:        cfg,
		log:        log,
		localStore: localStore,
	}
}

// ListImages 列出所有图片文件
func (s *FileManagementService) ListImages() ([]*FileInfo, error) {
	var images []*FileInfo
	imagesPath := filepath.Join(s.cfg.Storage.LocalPath, "images")

	// 检查目录是否存在
	if _, err := os.Stat(imagesPath); os.IsNotExist(err) {
		return images, nil
	}

	// 遍历目录
	err := filepath.Walk(imagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isImageFile(info.Name()) {
			fileInfo, err := s.createFileInfo(path, info)
			if err != nil {
				s.log.Errorw("Failed to get file info", "path", path, "error", err)
				return nil // 继续处理其他文件
			}
			images = append(images, fileInfo)
		}
		return nil
	})

	if err != nil {
		s.log.Errorw("Failed to list images", "error", err)
		return nil, err
	}

	return images, nil
}

// ListVideos 列出所有视频文件
func (s *FileManagementService) ListVideos() ([]*FileInfo, error) {
	var videos []*FileInfo
	videosPath := filepath.Join(s.cfg.Storage.LocalPath, "videos")

	// 检查目录是否存在
	if _, err := os.Stat(videosPath); os.IsNotExist(err) {
		return videos, nil
	}

	// 遍历目录
	err := filepath.Walk(videosPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isVideoFile(info.Name()) {
			fileInfo, err := s.createFileInfo(path, info)
			if err != nil {
				s.log.Errorw("Failed to get file info", "path", path, "error", err)
				return nil // 继续处理其他文件
			}
			videos = append(videos, fileInfo)
		}
		return nil
	})

	if err != nil {
		s.log.Errorw("Failed to list videos", "error", err)
		return nil, err
	}

	return videos, nil
}

// DeleteFile 删除文件
func (s *FileManagementService) DeleteFile(filePath string) error {
	fullPath := filepath.Join(s.cfg.Storage.LocalPath, strings.TrimPrefix(filePath, "/"))
	s.log.Infow("Deleting file", "path", fullPath)

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", fullPath)
	}

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		s.log.Errorw("Failed to delete file", "path", fullPath, "error", err)
		return err
	}

	return nil
}

// DeleteImage 删除图片文件
func (s *FileManagementService) DeleteImage(filename string) error {
	return s.DeleteFile(filepath.Join("images", filename))
}

// DeleteVideo 删除视频文件
func (s *FileManagementService) DeleteVideo(filename string) error {
	return s.DeleteFile(filepath.Join("videos", filename))
}

// createFileInfo 创建文件信息
func (s *FileManagementService) createFileInfo(path string, info os.FileInfo) (*FileInfo, error) {
	fileInfo := &FileInfo{
		Filename:     info.Name(),
		Path:         path,
		Size:         info.Size(),
		CreatedAt:    info.ModTime(),
		IsAssociated: false,
	}

	// 检查文件是否与数据库中的记录关联
	if isImageFile(info.Name()) {
		fileInfo.IsAssociated, fileInfo.AssociatedType, fileInfo.AssociatedID = s.isImageAssociated(info.Name())
	} else if isVideoFile(info.Name()) {
		fileInfo.IsAssociated, fileInfo.AssociatedType, fileInfo.AssociatedID = s.isVideoAssociated(info.Name())
	}

	return fileInfo, nil
}

// isImageAssociated 检查图片文件是否与数据库中的记录关联
func (s *FileManagementService) isImageAssociated(filename string) (bool, string, uint) {
	// 检查图片生成记录（同时检查 image_url 和 local_path）
	var imageGen models.ImageGeneration
	if err := s.db.Where("image_url LIKE ? OR local_path LIKE ?", "%"+filename+"%", "%"+filename+"%").First(&imageGen).Error; err == nil {
		return true, "image", imageGen.ID
	}

	// 检查角色图片（同时检查 image_url 和 local_path）
	var character models.Character
	if err := s.db.Where("image_url LIKE ? OR local_path LIKE ?", "%"+filename+"%", "%"+filename+"%").First(&character).Error; err == nil {
		return true, "character", character.ID
	}

	// 检查场景图片（同时检查 image_url 和 local_path）
	var scene models.Scene
	if err := s.db.Where("image_url LIKE ? OR local_path LIKE ?", "%"+filename+"%", "%"+filename+"%").First(&scene).Error; err == nil {
		return true, "scene", scene.ID
	}

	// 检查道具图片（同时检查 image_url 和 local_path）
	var prop models.Prop
	if err := s.db.Where("image_url LIKE ? OR local_path LIKE ?", "%"+filename+"%", "%"+filename+"%").First(&prop).Error; err == nil {
		return true, "prop", prop.ID
	}

	return false, "", 0
}

// isVideoAssociated 检查视频文件是否与数据库中的记录关联
func (s *FileManagementService) isVideoAssociated(filename string) (bool, string, uint) {
	// 检查视频生成记录（同时检查 video_url 和 local_path）
	var videoGen models.VideoGeneration
	if err := s.db.Where("video_url LIKE ? OR local_path LIKE ?", "%"+filename+"%", "%"+filename+"%").First(&videoGen).Error; err == nil {
		return true, "video", videoGen.ID
	}

	// 检查视频合成记录（同时检查 merged_url 和 local_path）
	var videoMerge models.VideoMerge
	if err := s.db.Where("merged_url LIKE ? OR local_path LIKE ?", "%"+filename+"%", "%"+filename+"%").First(&videoMerge).Error; err == nil {
		return true, "video_merge", videoMerge.ID
	}

	return false, "", 0
}

// isImageFile 检查文件是否是图片文件
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}

// isVideoFile 检查文件是否是视频文件
func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp4" || ext == ".avi" || ext == ".mov" || ext == ".mkv" || ext == ".webm"
}

// CleanupUnassociatedFiles 清理未关联的文件
func (s *FileManagementService) CleanupUnassociatedFiles() ([]string, error) {
	var deletedFiles []string

	// 清理未关联的图片文件
	images, err := s.ListImages()
	if err != nil {
		s.log.Errorw("Failed to list images for cleanup", "error", err)
	} else {
		for _, img := range images {
			if !img.IsAssociated {
				if err := s.DeleteFile(strings.TrimPrefix(img.Path, s.cfg.Storage.LocalPath)); err == nil {
					deletedFiles = append(deletedFiles, img.Path)
					s.log.Infow("Deleted unassociated file", "path", img.Path)
				}
			}
		}
	}

	// 清理未关联的视频文件
	videos, err := s.ListVideos()
	if err != nil {
		s.log.Errorw("Failed to list videos for cleanup", "error", err)
	} else {
		for _, vid := range videos {
			if !vid.IsAssociated {
				if err := s.DeleteFile(strings.TrimPrefix(vid.Path, s.cfg.Storage.LocalPath)); err == nil {
					deletedFiles = append(deletedFiles, vid.Path)
					s.log.Infow("Deleted unassociated file", "path", vid.Path)
				}
			}
		}
	}

	return deletedFiles, nil
}
