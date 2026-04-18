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

// buildPayload 将标准请求转换为 Gemini 的层级结构
func (e *GoogleExecutor) buildPayload(stdReq *schema.StandardRequest) ([]byte, error) {
	var systemContent string
	var contents []map[string]interface{}

	// 1. 上下文与角色映射 (Gemini 仅支持 user 和 model 角色)
	for _, msg := range stdReq.Messages {
		if msg.Role == "system" {
			systemContent += msg.Content + "\n"
			continue
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		})
	}

	payloadMap := map[string]interface{}{
		"contents": contents,
	}

	// 2. 注入 System Instruction
	if systemContent != "" {
		payloadMap["system_instruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": strings.TrimSpace(systemContent)},
			},
		}
	}

	// 3. 注入工具 (Function Declarations)
	if len(stdReq.Tools) > 0 {
		var funcs []map[string]interface{}
		for _, tool := range stdReq.Tools {
			funcs = append(funcs, map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			})
		}
		payloadMap["tools"] = []map[string]interface{}{
			{"function_declarations": funcs},
		}
	}

	// 4. 思维链支持 (基于 2026 年 Gemini 规范占位)
	if stdReq.Thinking {
		payloadMap["generationConfig"] = map[string]interface{}{
			"thinking_config": map[string]interface{}{"include_thoughts": true},
		}
	}

	return json.Marshal(payloadMap)
}

func (e *GoogleExecutor) ExecuteStream(ctx context.Context, stdReq *schema.StandardRequest) (io.ReadCloser, error) {
	payload, err := e.buildPayload(stdReq)
	if err != nil {
		return nil, err
	}

	// Gemini 流式请求通常使用 /streamGenerateContent?alt=sse 后缀
	finalURL := e.BaseURL
	if !e.IsVertex {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		finalURL = fmt.Sprintf("%s%skey=%s", finalURL, sep, e.APIKey)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", finalURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if e.IsVertex && e.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.BearerToken)
	}

	httpClient := &http.Client{Timeout: 0}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Google API 错误 (%d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

func (e *GoogleExecutor) Execute(ctx context.Context, stdReq *schema.StandardRequest) ([]byte, error) {
	payload, err := e.buildPayload(stdReq)
	if err != nil {
		return nil, err
	}

	finalURL := e.BaseURL
	if !e.IsVertex {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		// 非流式通常省略 stream 后缀，这里简单处理
		finalURL = fmt.Sprintf("%s%skey=%s", finalURL, sep, e.APIKey)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", finalURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if e.IsVertex && e.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.BearerToken)
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API 错误 (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
