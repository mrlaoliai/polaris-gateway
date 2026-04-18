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
		Version: "2023-06-01", // 2026 年默认版本
	}
}

func (e *AnthropicExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	// 将标准请求转回 Anthropic 原生格式
	payload, _ := json.Marshal(map[string]interface{}{
		"model":      stdReq.Model,
		"messages":   stdReq.Messages,
		"max_tokens": 4096,
		"stream":     true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", e.Version)

	httpClient := &http.Client{Timeout: 0} // 流式请求不设总超时

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Anthropic API 错误 (%d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

func (e *AnthropicExecutor) Execute(ctx context.Context, req *schema.StandardRequest) ([]byte, error) {
	// 实现略，逻辑同上但 stream = false
	return nil, nil
}
