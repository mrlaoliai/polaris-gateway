// 内部使用：pkg/provider/anthropic.go
// 作者：mrlaoliai
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

type AnthropicExecutor struct {
	APIKey  string
	BaseURL string
	Version string
}

func NewAnthropicExecutor(apiKey, baseURL string) *AnthropicExecutor {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1/messages"
	}
	return &AnthropicExecutor{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Version: "2023-06-01",
	}
}

func (e *AnthropicExecutor) buildPayload(stdReq *schema.StandardRequest, stream bool) ([]byte, error) {
	var system string
	var messages []map[string]interface{}

	for _, m := range stdReq.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, map[string]interface{}{"role": m.Role, "content": m.Content})
	}

	// 修复：动态从 stdReq 获取参数，不再硬编码
	maxTokens := stdReq.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192 // 默认安全值
	}

	payload := map[string]interface{}{
		"model":          stdReq.Model,
		"messages":       messages,
		"max_tokens":     maxTokens,
		"temperature":    stdReq.Temperature,
		"stop_sequences": stdReq.StopSequences, // 修复：支持停止词
		"stream":         stream,
	}

	if system != "" {
		payload["system"] = system
	}

	return json.Marshal(payload)
}

func (e *AnthropicExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, _ := e.buildPayload(stdReq, true)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))

	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", e.Version)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic api error: %s", string(body))
	}
	return resp.Body, nil
}

func (e *AnthropicExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, _ := e.buildPayload(stdReq, false)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))

	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", e.Version)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
