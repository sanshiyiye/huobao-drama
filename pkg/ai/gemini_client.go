package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/drama-generator/backend/pkg/logger"
)

type GeminiClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

type GeminiTextRequest struct {
	Contents          []GeminiContent    `json:"contents"`
	SystemInstruction *GeminiInstruction `json:"systemInstruction,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiTextResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		Index         int    `json:"index"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func NewGeminiClient(baseURL, apiKey, model, endpoint string) *GeminiClient {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	if endpoint == "" {
		endpoint = "/v1beta/models/{model}:generateContent"
	}
	if model == "" {
		model = "gemini-3-pro"
	}
	return &GeminiClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *GeminiClient) GenerateText(prompt string, systemPrompt string, options ...func(*ChatCompletionRequest)) (string, error) {
	model := c.Model

	// 构建请求体
	reqBody := GeminiTextRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: prompt}},
				Role:  "user",
			},
		},
	}

	// 使用 systemInstruction 字段处理系统提示
	if systemPrompt != "" {
		reqBody.SystemInstruction = &GeminiInstruction{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// 替换端点中的 {model} 占位符
	endpoint := c.BaseURL + c.Endpoint
	endpoint = strings.ReplaceAll(endpoint, "{model}", model)
	url := fmt.Sprintf("%s?key=%s", endpoint, c.APIKey)

	safeURL := strings.Replace(url, c.APIKey, "***", 1)
	requestPreview := string(jsonData)
	if len(jsonData) > 300 {
		requestPreview = string(jsonData[:300]) + "..."
	}
	logger.L().Debugw("Gemini request",
		"url", safeURL,
		"model", model,
		"request_preview", requestPreview,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.L().Warnw("Gemini API error response",
			"status", resp.StatusCode,
			"response_preview", truncatePreview(string(body), 500),
		)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	bodyPreview := string(body)
	if len(body) > 500 {
		bodyPreview = string(body[:500]) + "..."
	}
	logger.L().Debugw("Gemini response",
		"status", resp.StatusCode,
		"response_preview", bodyPreview,
	)

	var result GeminiTextResponse
	if err := json.Unmarshal(body, &result); err != nil {
		errorPreview := string(body)
		if len(body) > 200 {
			errorPreview = string(body[:200])
		}
		return "", fmt.Errorf("parse response: %w, body preview: %s", err, errorPreview)
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no parts in response")
	}

	responseText := result.Candidates[0].Content.Parts[0].Text
	logger.L().Debugw("Gemini generated text", "content_length", len(responseText))

	return responseText, nil
}

func (c *GeminiClient) GenerateImage(prompt string, size string, n int) ([]string, error) {
	return nil, fmt.Errorf("GenerateImage not implemented for Gemini client")
}

func (c *GeminiClient) TestConnection() error {
	logger.L().Debugw("Gemini TestConnection start",
		"base_url", c.BaseURL,
		"model", c.Model,
		"endpoint", c.Endpoint,
	)
	_, err := c.GenerateText("Hello", "")
	if err != nil {
		logger.L().Warnw("Gemini TestConnection failed", "error", err)
	} else {
		logger.L().Debugw("Gemini TestConnection succeeded")
	}
	return err
}
