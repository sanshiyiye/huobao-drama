package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
)

type OpenAIImageClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

type DALLERequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	Size    string   `json:"size,omitempty"`
	Quality string   `json:"quality,omitempty"`
	N       int      `json:"n"`
	Image   []string `json:"image,omitempty"`
}

type DALLEResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}

func NewOpenAIImageClient(baseURL, apiKey, model, endpoint string) *OpenAIImageClient {
	if endpoint == "" {
		endpoint = "/v1/images/generations"
	}
	return &OpenAIImageClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *OpenAIImageClient) GenerateImage(prompt string, opts ...ImageOption) (*ImageResult, error) {
	options := &ImageOptions{
		Size:    "1920x1920",
		Quality: "standard",
	}

	for _, opt := range opts {
		opt(options)
	}

	model := c.Model
	if options.Model != "" {
		model = options.Model
	}

	reqBody := DALLERequest{
		Model:   model,
		Prompt:  prompt,
		Size:    options.Size,
		Quality: options.Quality,
		N:       1,
		Image:   options.ReferenceImages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + c.Endpoint
	logBody := utils.TruncateBase64InJSON(jsonData)
	logger.L().Debugw("OpenAI image request",
		"url", url,
		"model", model,
		"request_preview", logBody,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.L().Warnw("OpenAI image API error",
			"status", resp.StatusCode,
			"response_preview", truncateBody(utils.TruncateBase64InJSON(body), 500),
		)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	logger.L().Debugw("OpenAI image response", "response_preview", truncateBody(utils.TruncateBase64InJSON(body), 500))

	var result DALLEResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no image generated, response: %s", string(body))
	}

	return &ImageResult{
		Status:    "completed",
		ImageURL:  result.Data[0].URL,
		Completed: true,
	}, nil
}

func (c *OpenAIImageClient) GetTaskStatus(taskID string) (*ImageResult, error) {
	return nil, fmt.Errorf("not supported for OpenAI/DALL-E")
}

func truncateBody(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
