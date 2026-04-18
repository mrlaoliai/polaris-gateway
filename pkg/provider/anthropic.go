// 内部使用：pkg/provider/anthropic.go
// 作者：mrlaoliai
// 专注 Anthropic Claude API 的协议适配，确保符合 Linter 规范
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

type AnthropicExecutor struct {
	APIKey  string
	BaseURL string
}

func NewAnthropicExecutor(apiKey, baseURL string) *AnthropicExecutor {
	return &AnthropicExecutor{
		APIKey:  apiKey,
		BaseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func (e *AnthropicExecutor) buildPayload(stdReq *schema.StandardRequest) ([]byte, error) {
	// 将标准请求转换为 Anthropic 官方格式
	payload := map[string]interface{}{
		"model":      stdReq.Model,
		"messages":   stdReq.Messages,
		"max_tokens": stdReq.MaxTokens,
		"stream":     stdReq.Stream,
	}

	// 注入 Thinking 配置 (Anthropic 扩展)
	if stdReq.Thinking {
		payload["thinking"] = map[string]interface{}{
			"type": "enabled",
			// 根据实际需求动态调整预算，此处设为固定或从配置读取
			"budget_tokens": 4000,
		}
	}

	return json.Marshal(payload)
}

func (e *AnthropicExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, err := e.buildPayload(stdReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	// 注入标准 Anthropic 认证 Header
	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (e *AnthropicExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, err := e.buildPayload(stdReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	// 修正 Linter: 显式忽略 Body 关闭错误
	defer func() { _ = resp.Body.Close() }()

	return io.ReadAll(resp.Body)
}
