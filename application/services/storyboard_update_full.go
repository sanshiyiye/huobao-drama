package services

import (
	"fmt"

	"github.com/drama-generator/backend/domain/models"
)

// UpdateStoryboard 更新分镜的所有字段，并重新生成提示词
func (s *StoryboardService) UpdateStoryboard(storyboardID string, updates map[string]interface{}) error {
	// 查找分镜
	var storyboard models.Storyboard
	if err := s.db.First(&storyboard, storyboardID).Error; err != nil {
		return fmt.Errorf("storyboard not found: %w", err)
	}

	// 构建用于重新生成提示词的Storyboard结构
	sb := Storyboard{
		ShotNumber: storyboard.StoryboardNumber,
	}

	// 从updates中提取字段并更新
	updateData := make(map[string]interface{})

	if val, ok := updates["title"].(string); ok && val != "" {
		updateData["title"] = val
		sb.Title = val
	}
	if val, ok := updates["shot_type"].(string); ok && val != "" {
		updateData["shot_type"] = val
		sb.ShotType = val
	}
	if val, ok := updates["angle"].(string); ok && val != "" {
		updateData["angle"] = val
		sb.Angle = val
	}
	if val, ok := updates["movement"].(string); ok && val != "" {
		updateData["movement"] = val
		sb.Movement = val
	}
	if val, ok := updates["location"].(string); ok && val != "" {
		updateData["location"] = val
		sb.Location = val
	}
	if val, ok := updates["time"].(string); ok && val != "" {
		updateData["time"] = val
		sb.Time = val
	}
	if val, ok := updates["action"].(string); ok && val != "" {
		updateData["action"] = val
		sb.Action = val
	}
	if val, ok := updates["dialogue"].(string); ok && val != "" {
		updateData["dialogue"] = val
		sb.Dialogue = val
	}
	if val, ok := updates["result"].(string); ok && val != "" {
		updateData["result"] = val
		sb.Result = val
	}
	if val, ok := updates["atmosphere"].(string); ok && val != "" {
		updateData["atmosphere"] = val
		sb.Atmosphere = val
	}
	if val, ok := updates["description"].(string); ok && val != "" {
		updateData["description"] = val
	}
	if val, ok := updates["bgm_prompt"].(string); ok && val != "" {
		updateData["bgm_prompt"] = val
		sb.BgmPrompt = val
	}
	if val, ok := updates["sound_effect"].(string); ok && val != "" {
		updateData["sound_effect"] = val
		sb.SoundEffect = val
	}
	if val, ok := updates["duration"].(float64); ok {
		updateData["duration"] = int(val)
		sb.Duration = int(val)
	}
	if val, ok := updates["scene_id"].(float64); ok {
		sceneID := uint(val)
		updateData["scene_id"] = sceneID
	}

	// 使用当前数据库值填充缺失字段（用于生成提示词）
	if sb.Title == "" && storyboard.Title != nil {
		sb.Title = *storyboard.Title
	}
	if sb.ShotType == "" && storyboard.ShotType != nil {
		sb.ShotType = *storyboard.ShotType
	}
	if sb.Angle == "" && storyboard.Angle != nil {
		sb.Angle = *storyboard.Angle
	}
	if sb.Movement == "" && storyboard.Movement != nil {
		sb.Movement = *storyboard.Movement
	}
	if sb.Location == "" && storyboard.Location != nil {
		sb.Location = *storyboard.Location
	}
	if sb.Time == "" && storyboard.Time != nil {
		sb.Time = *storyboard.Time
	}
	if sb.Action == "" && storyboard.Action != nil {
		sb.Action = *storyboard.Action
	}
	if sb.Dialogue == "" && storyboard.Dialogue != nil {
		sb.Dialogue = *storyboard.Dialogue
	}
	if sb.Result == "" && storyboard.Result != nil {
		sb.Result = *storyboard.Result
	}
	if sb.Atmosphere == "" && storyboard.Atmosphere != nil {
		sb.Atmosphere = *storyboard.Atmosphere
	}
	if sb.BgmPrompt == "" && storyboard.BgmPrompt != nil {
		sb.BgmPrompt = *storyboard.BgmPrompt
	}
	if sb.SoundEffect == "" && storyboard.SoundEffect != nil {
		sb.SoundEffect = *storyboard.SoundEffect
	}
	if sb.Duration == 0 {
		sb.Duration = storyboard.Duration
	}

	// 处理角色关联更新
	if characters, ok := updates["characters"]; ok {
		// 检查characters是否是数组类型
		if characterList, isSlice := characters.([]interface{}); isSlice {
			// 转换为[]uint类型
			var characterIDs []uint
			for _, char := range characterList {
				if charID, isNumber := char.(float64); isNumber { // JSON解析后的数字类型是float64
					characterIDs = append(characterIDs, uint(charID))
				}
			}

			// 获取分镜所属的剧集和项目信息，以验证角色属于该项目
			var episode models.Episode
			if err := s.db.First(&episode, storyboard.EpisodeID).Error; err == nil {
				var validCharacters []models.Character
				if len(characterIDs) > 0 {
					var characters []models.Character
					if err := s.db.Where("id IN ?", characterIDs).Find(&characters).Error; err == nil {
						for _, char := range characters {
							if char.DramaID == episode.DramaID {
								validCharacters = append(validCharacters, char)
							}
						}
					}
				}

				// 更新角色关联
				if err := s.db.Model(&storyboard).Association("Characters").Replace(validCharacters); err != nil {
					s.log.Warnw("Failed to update storyboard characters", "error", err, "storyboard_id", storyboardID)
				} else {
					s.log.Infow("Storyboard characters updated",
						"storyboard_id", storyboardID,
						"character_count", len(validCharacters),
						"character_ids", func() []uint {
							var ids []uint
							for _, c := range validCharacters {
								ids = append(ids, c.ID)
							}
							return ids
						}())
				}
			}
		}
	}

	// 不自动重新生成image_prompt和video_prompt，
	// 确保与ProfessionalEditor.vue中通过"提取提示词"按钮生成的提示词一致
	// 用户需要在ProfessionalEditor.vue中点击"提取提示词"按钮来获取最新的提示词

	// 更新数据库
	if err := s.db.Model(&storyboard).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update storyboard: %w", err)
	}

	s.log.Infow("Storyboard updated successfully",
		"storyboard_id", storyboardID,
		"fields_updated", len(updateData))

	return nil
}
