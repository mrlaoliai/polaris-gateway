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

type GoogleExecutor struct {
	APIKey      string
	BaseURL     string
	IsVertex    bool
	BearerToken string // Vertex 可能需要 OAuth2 令牌
}

func NewGoogleExecutor(apiKey, baseURL string, isVertex bool) *GoogleExecutor {
	return &GoogleExecutor{
		APIKey:   apiKey,
		BaseURL:  baseURL,
		IsVertex: isVertex,
	}
}

func (e *GoogleExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	// Gemini/Vertex 的内容封装逻辑
	// 即使接口地址不同，内容体我们按你确认的“一致性”来处理
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": stdReq.Messages[len(stdReq.Messages)-1].Content},
				},
			},
		},
		// 对应 V3 中的 Thinking 支持
		"generationConfig": map[string]interface{}{
			"thinking_config": map[string]interface{}{"include_thoughts": stdReq.Thinking},
		},
	})

	// 处理接口地址差异
	finalURL := e.BaseURL
	if !e.IsVertex {
		// 标准 Gemini 模式通常在 URL 中带 Key
		finalURL = fmt.Sprintf("%s?key=%s", e.BaseURL, e.APIKey)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", finalURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// 如果是 Vertex，可能需要注入 Bearer Token
	if e.IsVertex && e.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.BearerToken)
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (e *GoogleExecutor) Execute(ctx context.Context, req *schema.StandardRequest) ([]byte, error) {
	return nil, nil
}
