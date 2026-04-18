// 内部使用：pkg/provider/google.go
// 作者：mrlaoliai
// 专注 Google AI Studio (Gemini API) 的协议适配与自适应路径映射
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

type GoogleExecutor struct {
	APIKey  string
	BaseURL string // 数据库只需配置到模型名，如: https://generativelanguage.googleapis.com/v1beta/models/gemini-pro
}

func NewGoogleExecutor(apiKey, baseURL string) *GoogleExecutor {
	return &GoogleExecutor{
		APIKey:  apiKey,
		BaseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// buildURL 根据流式状态动态构造完整路径
func (e *GoogleExecutor) buildURL(isStream bool) string {
	action := ":generateContent"
	if isStream {
		action = ":streamGenerateContent"
	}

	finalURL := e.BaseURL + action
	sep := "?"

	// Google AI Studio 流式请求必须携带 alt=sse 参数
	if isStream {
		finalURL = fmt.Sprintf("%s%salt=sse", finalURL, sep)
		sep = "&"
	}

	return fmt.Sprintf("%s%skey=%s", finalURL, sep, e.APIKey)
}

func (e *GoogleExecutor) buildPayload(stdReq *schema.StandardRequest) ([]byte, error) {
	var contents []map[string]interface{}
	var systemParts []map[string]interface{}

	for _, msg := range stdReq.Messages {
		if msg.Role == "system" {
			systemParts = append(systemParts, map[string]interface{}{"text": msg.Content})
			continue
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]interface{}{{"text": msg.Content}},
		})
	}

	payload := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     stdReq.Temperature,
			"maxOutputTokens": stdReq.MaxTokens,
		},
	}

	if len(systemParts) > 0 {
		payload["system_instruction"] = map[string]interface{}{"parts": systemParts}
	}

	if stdReq.Thinking {
		payload["generationConfig"].(map[string]interface{})["thinking_config"] = map[string]interface{}{
			"include_thoughts": true,
		}
	}

	return json.Marshal(payload)
}

func (e *GoogleExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, err := e.buildPayload(stdReq)
	if err != nil {
		return nil, err
	}
	url := e.buildURL(true)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (e *GoogleExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, err := e.buildPayload(stdReq)
	if err != nil {
		return nil, err
	}
	url := e.buildURL(false)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}
