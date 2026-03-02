package handlers

import (
	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CheckEpisodeFramePromptsCompletion 检查episode所有分镜的帧提示词完成情况
// GET /api/v1/episodes/:episode_id/frame-prompts-completion
func CheckEpisodeFramePromptsCompletion(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		episodeID := c.Param("episode_id")

		// 查询该episode的所有分镜
		var storyboards []models.Storyboard
		if err := db.Where("episode_id = ?", episodeID).
			Select("id").
			Find(&storyboards).Error; err != nil {
			log.Errorw("Failed to query storyboards", "error", err, "episode_id", episodeID)
			response.InternalError(c, err.Error())
			return
		}

		totalStoryboards := len(storyboards)
		if totalStoryboards == 0 {
			response.Success(c, gin.H{
				"total_storyboards":     0,
				"completed_storyboards":  0,
				"is_completed":          false,
				"completion_rate":       0.0,
			})
			return
		}

		// 每个分镜需要生成5种帧提示词：first, last, key, action, panel
		requiredFrameTypes := []string{"first", "last", "key", "action", "panel"}

		// 统计每个分镜的完成情况
		completedStoryboards := 0
		storyboardStatuses := make([]map[string]interface{}, 0, totalStoryboards)

		for _, sb := range storyboards {
			// 查询该分镜已生成的帧提示词类型
			var framePrompts []models.FramePrompt
			if err := db.Where("storyboard_id = ?", sb.ID).
				Select("frame_type").
				Find(&framePrompts).Error; err != nil {
				log.Warnw("Failed to query frame prompts for storyboard",
					"error", err, "storyboard_id", sb.ID)
				continue
			}

			// 统计已生成的类型
			generatedTypes := make(map[string]bool)
			for _, fp := range framePrompts {
				generatedTypes[fp.FrameType] = true
			}

			// 检查是否所有类型都已生成
			allGenerated := true
			for _, frameType := range requiredFrameTypes {
				if !generatedTypes[frameType] {
					allGenerated = false
					break
				}
			}

			if allGenerated {
				completedStoryboards++
			}

			// 构建已生成的类型列表
			generatedTypesList := make([]string, 0, len(generatedTypes))
			for t := range generatedTypes {
				generatedTypesList = append(generatedTypesList, t)
			}

			storyboardStatuses = append(storyboardStatuses, map[string]interface{}{
				"storyboard_id":  sb.ID,
				"completed":      allGenerated,
				"generated_types": generatedTypesList,
			})
		}

		isCompleted := completedStoryboards == totalStoryboards
		completionRate := float64(completedStoryboards) / float64(totalStoryboards) * 100.0

		response.Success(c, gin.H{
			"total_storyboards":     totalStoryboards,
			"completed_storyboards": completedStoryboards,
			"is_completed":          isCompleted,
			"completion_rate":       completionRate,
			"storyboard_statuses":   storyboardStatuses,
		})
	}
}
