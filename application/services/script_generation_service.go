package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"gorm.io/gorm"
)

type ScriptGenerationService struct {
	db          *gorm.DB
	aiService   *AIService
	log         *logger.Logger
	config      *config.Config
	promptI18n  *PromptI18n
	taskService *TaskService
}

func NewScriptGenerationService(db *gorm.DB, cfg *config.Config, log *logger.Logger) *ScriptGenerationService {
	return &ScriptGenerationService{
		db:          db,
		aiService:   NewAIService(db, log),
		log:         log,
		config:      cfg,
		promptI18n:  NewPromptI18n(cfg),
		taskService: NewTaskService(db, log),
	}
}

type GenerateCharactersRequest struct {
	DramaID     string  `json:"drama_id" binding:"required"`
	EpisodeID   uint    `json:"episode_id"`
	Outline     string  `json:"outline"`
	Count       int     `json:"count"`
	Temperature float64 `json:"temperature"`
	Model       string  `json:"model"` // 指定使用的文本模型
}

func (s *ScriptGenerationService) GenerateCharacters(req *GenerateCharactersRequest) (string, error) {
	var drama models.Drama
	if err := s.db.Where("id = ? ", req.DramaID).First(&drama).Error; err != nil {
		return "", fmt.Errorf("drama not found")
	}

	// 创建任务
	task, err := s.taskService.CreateTask("character_generation", req.DramaID)
	if err != nil {
		s.log.Errorw("Failed to create character generation task", "error", err)
		return "", fmt.Errorf("创建任务失败: %w", err)
	}

	// 异步处理角色生成
	go s.processCharacterGeneration(task.ID, req)

	s.log.Infow("Character generation task created", "task_id", task.ID, "drama_id", req.DramaID)
	return task.ID, nil
}

// processCharacterGeneration 异步处理角色生成
func (s *ScriptGenerationService) processCharacterGeneration(taskID string, req *GenerateCharactersRequest) {
	// 更新任务状态为处理中
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "正在生成角色...")

	count := req.Count
	if count == 0 {
		count = 5
	}

	// 获取 drama 的 style 信息
	var drama models.Drama
	if err := s.db.Where("id = ? ", req.DramaID).First(&drama).Error; err != nil {
		s.log.Errorw("Drama not found during character generation", "error", err, "drama_id", req.DramaID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "剧本信息不存在")
		return
	}

	// 分析剧情背景信息（从大纲或剧本中提取）
	var context string
	if req.Outline != "" {
		context = s.promptI18n.AnalyzeScriptContext(req.Outline)
	} else {
		// 如果没有大纲，从剧本描述中提取
		var dramaInfo string
		if drama.Description != nil {
			dramaInfo = *drama.Description
		}
		context = s.promptI18n.AnalyzeScriptContext(dramaInfo)
	}

	systemPrompt := s.promptI18n.GetCharacterExtractionPromptTest(drama.Style, drama.PlotStyle, context)

	outlineText := req.Outline
	if outlineText == "" {
		outlineText = s.promptI18n.FormatUserPrompt("drama_info_template", drama.Title, drama.Description, drama.Genre)
	}

	userPrompt := s.promptI18n.FormatUserPrompt("character_request", outlineText, count)

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// 如果指定了模型，使用指定的模型；否则使用默认配置
	var text string
	var err error
	if req.Model != "" {
		s.log.Infow("Using specified model for character generation", "model", req.Model, "task_id", taskID)
		client, getErr := s.aiService.GetAIClientForModel("text", req.Model)
		if getErr != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", req.Model, "error", getErr, "task_id", taskID)
			text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
		} else {
			text, err = client.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
		}
	} else {
		text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
	}

	if err != nil {
		s.log.Errorw("Failed to generate characters", "error", err, "task_id", taskID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "AI生成失败: "+err.Error())
		return
	}

	s.log.Infow("AI response received for character generation", "length", len(text), "preview", text[:minInt(200, len(text))], "task_id", taskID)

	// AI 返回的格式可能是对象包含 characters 字段或者直接是数组
	var result []struct {
		Name        string `json:"name"`
		Role        string `json:"role"`
		Age         string `json:"age"`
		Gender      string `json:"gender"`
		Description string `json:"description"`
		Personality string `json:"personality"`
		Appearance  string `json:"appearance"`
		VoiceStyle  string `json:"voice_style"`
	}

	// 首先尝试解析为对象包含 characters 字段的格式
	var response struct {
		Characters []struct {
			Name        string `json:"name"`
			Role        string `json:"role"`
			Age         string `json:"age"`
			Gender      string `json:"gender"`
			Description string `json:"description"`
			Personality string `json:"personality"`
			Appearance  string `json:"appearance"`
			VoiceStyle  string `json:"voice_style"`
		} `json:"characters"`
	}

	if err := utils.SafeParseAIJSON(text, &response); err == nil && len(response.Characters) > 0 {
		result = response.Characters
	} else {
		// 如果解析失败，尝试直接解析为数组格式
		if err := utils.SafeParseAIJSON(text, &result); err != nil {
			s.log.Errorw("Failed to parse characters JSON", "error", err, "raw_response", text[:minInt(500, len(text))], "task_id", taskID)
			s.taskService.UpdateTaskStatus(taskID, "failed", 0, "解析AI返回结果失败")
			return
		}
	}

	// 后处理：从 appearance 字段中去除年龄和性别信息
	// 这样无论AI如何生成，我们都能确保 appearance 字段是纯净的
	for i := range result {
		// 去除年龄信息
		agePatterns := []string{
			"婴儿", "幼儿", "儿童", "青少年", "青年", "成年", "中年", "年长者", "老年",
			"infant", "toddler", "child", "teenager", "youth", "adult", "middle-aged", "elderly", "senior",
		}
		for _, pattern := range agePatterns {
			result[i].Appearance = strings.ReplaceAll(result[i].Appearance, pattern, "")
		}

		// 去除性别信息
		genderPatterns := []string{
			"男", "女", "其他",
			"male", "female", "other",
		}
		for _, pattern := range genderPatterns {
			result[i].Appearance = strings.ReplaceAll(result[i].Appearance, pattern, "")
		}

		// 去除可能残留的标点符号和多余空格
		result[i].Appearance = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(result[i].Appearance), " ")
	}

	// 调试日志：打印解析后的角色数据
	s.log.Infow("Extracted characters from AI", "characters", result, "task_id", taskID)

	var characters []models.Character
	for _, char := range result {
		// 检查角色是否已存在
		var existingChar models.Character
		err := s.db.Where("drama_id = ? AND name = ?", req.DramaID, char.Name).First(&existingChar).Error
		if err == nil {
			// 角色已存在，更新 age 和 gender 字段（如果 AI 返回了这些数据）
			s.log.Infow("Character already exists, updating age and gender", "drama_id", req.DramaID, "name", char.Name, "task_id", taskID)
			if char.Age != "" {
				existingChar.Age = &char.Age
			}
			if char.Gender != "" {
				existingChar.Gender = &char.Gender
			}
			if err := s.db.Save(&existingChar).Error; err != nil {
				s.log.Errorw("Failed to update character age and gender", "error", err, "task_id", taskID, "character_id", existingChar.ID)
			}
			characters = append(characters, existingChar)
			continue
		}

		// 角色不存在，创建新角色
		dramaID, _ := strconv.ParseUint(req.DramaID, 10, 32)
		character := models.Character{
			DramaID:     uint(dramaID),
			Name:        char.Name,
			Role:        &char.Role,
			Age:         &char.Age,
			Gender:      &char.Gender,
			Description: &char.Description,
			Personality: &char.Personality,
			Appearance:  &char.Appearance,
			VoiceStyle:  &char.VoiceStyle,
		}

		if err := s.db.Create(&character).Error; err != nil {
			s.log.Errorw("Failed to create character", "error", err, "task_id", taskID)
			continue
		}

		characters = append(characters, character)
	}

	// 如果提供了 EpisodeID，建立 episode_characters 关联关系
	if req.EpisodeID > 0 {
		var episode models.Episode
		if err := s.db.First(&episode, req.EpisodeID).Error; err == nil {
			// 使用 GORM 的 Association 建立多对多关联
			if err := s.db.Model(&episode).Association("Characters").Append(characters); err != nil {
				s.log.Errorw("Failed to associate characters with episode", "error", err, "episode_id", req.EpisodeID, "task_id", taskID)
			} else {
				s.log.Infow("Characters associated with episode", "episode_id", req.EpisodeID, "character_count", len(characters), "task_id", taskID)
			}
		} else {
			s.log.Errorw("Episode not found for association", "episode_id", req.EpisodeID, "error", err, "task_id", taskID)
		}
	}

	// 更新任务状态为完成
	resultData := map[string]interface{}{
		"characters": characters,
		"count":      len(characters),
	}
	s.taskService.UpdateTaskResult(taskID, resultData)

	s.log.Infow("Character generation completed", "task_id", taskID, "drama_id", req.DramaID, "character_count", len(characters))
}

// GenerateScenesForEpisode 已废弃，使用 StoryboardService.GenerateStoryboard 替代
// ParseScript 已废弃，使用 GenerateCharacters 替代

// minInt 返回两个整数中较小的一个
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
