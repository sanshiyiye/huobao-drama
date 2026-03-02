package ai

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/drama-generator/backend/pkg/logger"
)

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model               string        `json:"model"`
	Messages            []ChatMessage `json:"messages"`
	Temperature         float64       `json:"temperature,omitempty"`
	MaxTokens           *int          `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int          `json:"max_completion_tokens,omitempty"`
	TopP                float64       `json:"top_p,omitempty"`
	Stream              bool          `json:"stream,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ImageGenerationRequest struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`
	Size   string `json:"size,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func NewOpenAIClient(baseURL, apiKey, model, endpoint string) *OpenAIClient {
	if endpoint == "" {
		endpoint = "/v1/chat/completions"
	}

	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	return &OpenAIClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout:   10 * time.Minute,
			Transport: transport,
		},
	}
}

func (c *OpenAIClient) ChatCompletion(messages []ChatMessage, options ...func(*ChatCompletionRequest)) (*ChatCompletionResponse, error) {
	req := &ChatCompletionRequest{
		Model:    c.Model,
		Messages: messages,
	}

	for _, option := range options {
		option(req)
	}

	return c.sendChatRequest(req)
}

func (c *OpenAIClient) sendChatRequest(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	resp, err := c.doChatRequest(req)
	if err == nil {
		return resp, nil
	}

	if shouldRetryWithMaxCompletionTokens(err, req) {
		tokens := *req.MaxTokens
		retryReq := *req
		retryReq.MaxTokens = nil
		retryReq.MaxCompletionTokens = &tokens
		logger.L().Debugw("OpenAI retrying with max_completion_tokens", "max_completion_tokens", tokens)
		return c.doChatRequest(&retryReq)
	}

	return nil, err
}

func (c *OpenAIClient) doChatRequest(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.BaseURL + c.Endpoint

	requestPreview := string(jsonData)
	if len(jsonData) > 300 {
		requestPreview = string(jsonData[:300]) + "..."
	}
	logger.L().Debugw("OpenAI request",
		"url", url,
		"base_url", c.BaseURL,
		"endpoint", c.Endpoint,
		"model", c.Model,
		"request_preview", requestPreview,
	)

	const maxAttempts = 3
	var resp *http.Response
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		httpReq, reqErr := http.NewRequest("POST", url, bytes.NewReader(jsonData))
		if reqErr != nil {
			return nil, fmt.Errorf("failed to create request: %w", reqErr)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

		resp, err = c.HTTPClient.Do(httpReq)
		if err == nil {
			break
		}

		// 只对“请求未发送成功”的连接阶段错误重试，避免重复计费。
		if attempt < maxAttempts && isSafeToRetryTransportError(err) {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			logger.L().Warnw("OpenAI transport error, retrying safely",
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"backoff_seconds", backoff.Seconds(),
				"error", err.Error(),
			)
			time.Sleep(backoff)
			continue
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("failed to send request: no response and no explicit error")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.L().Warnw("OpenAI API error response",
			"status", resp.StatusCode,
			"response_preview", truncatePreview(string(body), 500),
		)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
	}

	bodyPreview := string(body)
	if len(body) > 500 {
		bodyPreview = string(body[:500]) + "..."
	}
	logger.L().Debugw("OpenAI response",
		"status", resp.StatusCode,
		"response_preview", bodyPreview,
	)

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		errorPreview := string(body)
		if len(body) > 200 {
			errorPreview = string(body[:200])
		}
		return nil, fmt.Errorf("failed to unmarshal response: %w, body preview: %s", err, errorPreview)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// 检查 finish_reason，处理内容过滤的情况
	if len(chatResp.Choices) > 0 {
		finishReason := chatResp.Choices[0].FinishReason
		content := chatResp.Choices[0].Message.Content
		usage := chatResp.Usage

		logger.L().Debugw("OpenAI choice summary",
			"finish_reason", finishReason,
			"content_length", len(content),
			"total_tokens", usage.TotalTokens,
		)

		if finishReason == "content_filter" {
			return nil, fmt.Errorf("AI内容被安全过滤器拦截，可能因为：\n1. 请求内容触发了安全策略\n2. 生成的内容包含敏感信息\n3. 建议：调整输入内容或联系API提供商调整过滤策略")
		}

		if usage.TotalTokens == 0 && finishReason != "stop" {
			return nil, fmt.Errorf("AI返回内容为空 (finish_reason: %s)，可能的原因：\n1. 内容被过滤\n2. Token限制\n3. API异常", finishReason)
		}
	}

	return &chatResp, nil
}

func isSafeToRetryTransportError(err error) bool {
	if err == nil {
		return false
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		err = urlErr.Err
	}

	msg := strings.ToLower(err.Error())

	// 一旦进入“等待响应头”阶段，请求通常已发送，避免重试导致重复计费。
	if strings.Contains(msg, "awaiting headers") {
		return false
	}

	retryableMarkers := []string{
		"tls handshake timeout",
		"no such host",
		"connection refused",
		"network is unreachable",
		"dial tcp",
	}
	for _, marker := range retryableMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}

	// 仅在明显属于连接阶段的 timeout 才允许重试。
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return strings.Contains(msg, "handshake") || strings.Contains(msg, "dial")
	}

	return false
}

func WithTemperature(temp float64) func(*ChatCompletionRequest) {
	return func(req *ChatCompletionRequest) {
		req.Temperature = temp
	}
}

func WithMaxTokens(tokens int) func(*ChatCompletionRequest) {
	return func(req *ChatCompletionRequest) {
		req.MaxTokens = &tokens
	}
}

func WithTopP(topP float64) func(*ChatCompletionRequest) {
	return func(req *ChatCompletionRequest) {
		req.TopP = topP
	}
}

func (c *OpenAIClient) GenerateText(prompt string, systemPrompt string, options ...func(*ChatCompletionRequest)) (string, error) {
	messages := []ChatMessage{}

	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	resp, err := c.ChatCompletion(messages, options...)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) GenerateImage(prompt string, size string, n int) ([]string, error) {
	// 图片生成端点通常是 /v1/images/generations
	// 如果 c.Endpoint 是 chat 端点，我们需要将其替换
	// 这是一个简单的处理逻辑，实际可能需要更复杂的配置
	imageEndpoint := "/v1/images/generations"

	// 如果 BaseURL 是类似 api.openai.com，那么直接拼接
	url := c.BaseURL + imageEndpoint

	reqBody := ImageGenerationRequest{
		Prompt: prompt,
		N:      n,
		Size:   size,
		Model:  c.Model, // 如果是DALL-E 3，模型名很重要
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var imgResp ImageGenerationResponse
	if err := json.Unmarshal(body, &imgResp); err != nil {
		return nil, err
	}

	var urls []string
	for _, data := range imgResp.Data {
		if data.URL != "" {
			urls = append(urls, data.URL)
		} else if data.B64JSON != "" {
			// 如果返回的是base64，添加前缀
			urls = append(urls, "data:image/png;base64,"+data.B64JSON)
		}
	}

	return urls, nil
}

func (c *OpenAIClient) TestConnection() error {
	logger.L().Debugw("OpenAI TestConnection start",
		"base_url", c.BaseURL,
		"endpoint", c.Endpoint,
		"model", c.Model,
	)

	messages := []ChatMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
	}

	_, err := c.ChatCompletion(messages, WithMaxTokens(50))
	if err != nil {
		logger.L().Warnw("OpenAI TestConnection failed", "error", err)
	} else {
		logger.L().Debugw("OpenAI TestConnection succeeded")
	}
	return err
}

func shouldRetryWithMaxCompletionTokens(err error, req *ChatCompletionRequest) bool {
	if err == nil || req == nil || req.MaxTokens == nil || req.MaxCompletionTokens != nil {
		return false
	}

	msg := err.Error()
	if strings.Contains(msg, "Unsupported parameter: 'max_tokens'") {
		return true
	}
	if strings.Contains(msg, "max_tokens is not supported") {
		return true
	}
	if strings.Contains(msg, "max_completion_tokens") {
		return true
	}
	return false
}

func truncatePreview(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
