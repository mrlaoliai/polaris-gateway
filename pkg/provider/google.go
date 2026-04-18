// 内部使用：pkg/provider/google.go
// 作者：mrlaoliai
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
	APIKey      string
	BaseURL     string
	IsVertex    bool
	BearerToken string
}

func NewGoogleExecutor(apiKey, baseURL string, isVertex bool) *GoogleExecutor {
	return &GoogleExecutor{
		APIKey:   apiKey,
		BaseURL:  baseURL,
		IsVertex: isVertex,
	}
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
			"maxOutputTokens": stdReq.MaxTokens,     // 修复：映射最大长度
			"stopSequences":   stdReq.StopSequences, // 修复：映射停止词
		},
	}

	if len(systemParts) > 0 {
		payload["system_instruction"] = map[string]interface{}{"parts": systemParts}
	}

	// 注入思维链配置
	if stdReq.Thinking {
		payload["generationConfig"].(map[string]interface{})["thinking_config"] = map[string]interface{}{
			"include_thoughts": true,
		}
	}

	return json.Marshal(payload)
}

func (e *GoogleExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, _ := e.buildPayload(stdReq)

	finalURL := e.BaseURL
	// 修复：Gemini API 必须显式要求 alt=sse，否则返回的是原生 JSON 数组，Transformer 无法扫描
	sep := "?"
	if strings.Contains(finalURL, "?") {
		sep = "&"
	}
	finalURL = fmt.Sprintf("%s%salt=sse", finalURL, sep)

	if !e.IsVertex {
		finalURL = fmt.Sprintf("%s&key=%s", finalURL, e.APIKey)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", finalURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if e.IsVertex {
		req.Header.Set("Authorization", "Bearer "+e.BearerToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (e *GoogleExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, _ := e.buildPayload(stdReq)
	finalURL := e.BaseURL
	if !e.IsVertex {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		finalURL = fmt.Sprintf("%s%skey=%s", finalURL, sep, e.APIKey)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", finalURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
