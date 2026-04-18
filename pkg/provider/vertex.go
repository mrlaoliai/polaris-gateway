// 内部使用：pkg/provider/vertex.go
// 作者：mrlaoliai
// 专注 Vertex AI 协议适配，支持 API Key 认证与自适应路径映射
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

type VertexExecutor struct {
	APIKey      string
	BaseURL     string // 数据库只需配置到模型名，如: https://aiplatform.googleapis.com/.../models/gemini-1.5-flash
	BearerToken string
}

func NewVertexExecutor(apiKey, baseURL string) *VertexExecutor {
	return &VertexExecutor{
		APIKey:  apiKey,
		BaseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// buildURL 根据流式状态动态构造 Vertex 特有的请求路径
func (e *VertexExecutor) buildURL(isStream bool) string {
	action := ":generateContent"
	if isStream {
		action = ":streamGenerateContent"
	}

	finalURL := e.BaseURL + action
	sep := "?"

	// Vertex 如果是流式，同样需要 alt=sse
	if isStream {
		finalURL = fmt.Sprintf("%s%salt=sse", finalURL, sep)
		sep = "&"
	}

	// 注入 API Key 认证参数
	if e.APIKey != "" {
		finalURL = fmt.Sprintf("%s%skey=%s", finalURL, sep, e.APIKey)
	}

	return finalURL
}

func (e *VertexExecutor) buildPayload(stdReq *schema.StandardRequest) ([]byte, error) {
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

func (e *VertexExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, _ := e.buildPayload(stdReq)
	url := e.buildURL(true)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	// 保留对 Bearer Token 的支持 (可选)
	if e.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.BearerToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (e *VertexExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, _ := e.buildPayload(stdReq)
	url := e.buildURL(false)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	if e.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.BearerToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
