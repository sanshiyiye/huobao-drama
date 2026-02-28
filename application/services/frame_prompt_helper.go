package services

import (
	"encoding/json"
	"regexp"
	"strings"
)

// parseFramePromptJSON 解析AI返回的JSON格式提示词
// 如果AI返回的是纯文本而不是JSON，会直接使用该文本作为prompt
func (s *FramePromptService) parseFramePromptJSON(aiResponse string) *SingleFramePrompt {
	// 清理可能的markdown代码块标记
	cleaned := strings.TrimSpace(aiResponse)

	// 移除 ```json 和 ``` 标记
	re := regexp.MustCompile("(?s)```json\\s*(.+?)\\s*```")
	if matches := re.FindStringSubmatch(cleaned); len(matches) > 1 {
		cleaned = strings.TrimSpace(matches[1])
	} else {
		// 移除单独的 ``` 标记
		cleaned = strings.Trim(cleaned, "`")
		cleaned = strings.TrimSpace(cleaned)
	}

	// 检查是否是纯文本（没有JSON结构）
	// 如果响应不是以 { 或 [ 开头，很可能是纯文本
	if !strings.HasPrefix(cleaned, "{") && !strings.HasPrefix(cleaned, "[") {
		// AI返回了纯文本而不是JSON，直接使用
		s.log.Infow("AI returned plain text instead of JSON, using as prompt", "response_length", len(cleaned))
		return &SingleFramePrompt{
			Prompt:      cleaned,
			Description: "AI生成的提示词",
		}
	}

	// 尝试解析JSON
	var result SingleFramePrompt
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// JSON解析失败，检查是否是纯文本（响应中不包含JSON结构）
		if !strings.Contains(cleaned, "{") && !strings.Contains(cleaned, "[") {
			// 响应看起来是纯文本，直接使用
			s.log.Infow("Failed to parse JSON, but response appears to be plain text, using as prompt", 
				"response_length", len(cleaned), "error", err)
			return &SingleFramePrompt{
				Prompt:      cleaned,
				Description: "AI生成的提示词",
			}
		}
		// 真正的JSON解析错误，记录警告
		s.log.Warnw("Failed to parse JSON", "error", err, "cleaned_response_preview", 
			func() string {
				if len(cleaned) > 200 {
					return cleaned[:200] + "..."
				}
				return cleaned
			}())
		return nil
	}

	// 验证必需字段
	if result.Prompt == "" {
		s.log.Warnw("Parsed JSON missing prompt field", "response_preview", 
			func() string {
				if len(cleaned) > 200 {
					return cleaned[:200] + "..."
				}
				return cleaned
			}())
		return nil
	}

	return &result
}
