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

// buildPayload 负责将标准的 StandardRequest 映射为真实的 Anthropic 负载
func (e *AnthropicExecutor) buildPayload(stdReq *schema.StandardRequest, stream bool) ([]byte, error) {
	var systemPrompt string
	var messages []map[string]interface{}

	// 1. 拆分上下文：Anthropic 强制要求 system 独立
	for _, msg := range stdReq.Messages {
		if msg.Role == "system" {
			systemPrompt += msg.Content + "\n"
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	// 2. 组装基础请求
	payloadMap := map[string]interface{}{
		"model":      stdReq.Model,
		"messages":   messages,
		"max_tokens": 8192, // 提高上限以支持长代码生成
		"stream":     stream,
	}

	if systemPrompt != "" {
		payloadMap["system"] = systemPrompt
	}

	// 3. 映射 MCP 工具调用
	if len(stdReq.Tools) > 0 {
		var anthropicTools []map[string]interface{}
		for _, tool := range stdReq.Tools {
			anthropicTools = append(anthropicTools, map[string]interface{}{
				"name":         tool.Function.Name,
				"description":  tool.Function.Description,
				"input_schema": tool.Function.Parameters, // Anthropic 规范
			})
		}
		payloadMap["tools"] = anthropicTools
	}

	return json.Marshal(payloadMap)
}

func (e *AnthropicExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, err := e.buildPayload(stdReq, true)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", e.Version)

	httpClient := &http.Client{Timeout: 0}
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

func (e *AnthropicExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, err := e.buildPayload(stdReq, false)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", e.Version)

	httpClient := &http.Client{} // 非流式请求可以使用默认的隐藏超时
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API 错误 (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
