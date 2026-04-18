package transformer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/heartbeat"
	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
)

type AnthropicTransformer struct {
	targetModel string
	mcpRouter   *MCPRouter
}

func NewAnthropicTransformer(target string) *AnthropicTransformer {
	return &AnthropicTransformer{
		targetModel: target,
		mcpRouter:   NewMCPRouter(),
	}
}

func (t *AnthropicTransformer) TransformRequest(payload []byte) (*schema.StandardRequest, error) {
	var claudeReq struct {
		Model    string `json:"model"`
		System   string `json:"system"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
		Tools  []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"input_schema"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(payload, &claudeReq); err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	stdReq := &schema.StandardRequest{
		Model:    t.targetModel,
		Stream:   claudeReq.Stream,
		Thinking: true,
	}

	if claudeReq.System != "" {
		stdReq.Messages = append(stdReq.Messages, schema.Message{Role: "system", Content: claudeReq.System})
	}

	for _, msg := range claudeReq.Messages {
		stdReq.Messages = append(stdReq.Messages, schema.Message{Role: msg.Role, Content: msg.Content})
	}

	for _, tool := range claudeReq.Tools {
		stdReq.Tools = append(stdReq.Tools, schema.Tool{
			Type: "function",
			Function: struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Parameters  any    `json:"parameters"`
			}{
				Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema,
			},
		})
	}

	return stdReq, nil
}

func (t *AnthropicTransformer) TransformStream(ctx context.Context, physicalStream io.Reader, clientStream io.Writer) error {
	// [深度集成] 初始化心跳，防止推理阶段超时
	injector := heartbeat.NewInjector(clientStream, 15*time.Second, "anthropic")
	injector.Start(ctx)
	defer injector.Stop()

	scanner := bufio.NewScanner(physicalStream)
	zpFilter := middleware.NewZeroPoetryProcessor()

	// 1. 发送握手包
	msgStart := fmt.Sprintf("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"polaris_tx\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"%s\",\"content\":[]}}\n\n", t.targetModel)
	_, _ = injector.Write([]byte(msgStart))

	var inThinking, inText bool
	var blockIndex int

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") || strings.Contains(line, "[DONE]") {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					ReasoningContent string `json:"reasoning_content"`
					Content          string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err != nil || len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// 2. 处理 Thinking 块 (影子签名)
		if delta.ReasoningContent != "" {
			if !inThinking {
				injector.Write([]byte(fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\"}}\n\n", blockIndex)))
				inThinking = true
			}
			safeT, _ := json.Marshal(delta.ReasoningContent)
			injector.Write([]byte(fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":%d,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":%s}}\n\n", blockIndex, string(safeT))))
		}

		// 3. 处理 Text 块
		if delta.Content != "" {
			if inThinking {
				injector.Write([]byte(fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)))
				inThinking = false
				blockIndex++
			}
			if !inText {
				injector.Write([]byte(fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n", blockIndex)))
				inText = true
			}
			clean := zpFilter.Process(delta.Content)
			if clean != "" {
				safeC, _ := json.Marshal(clean)
				injector.Write([]byte(fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":%d,\"delta\":{\"type\":\"text_delta\",\"text\":%s}}\n\n", blockIndex, string(safeC))))
			}
		}

		// 4. 终结逻辑
		if chunk.Choices[0].FinishReason != nil {
			if inThinking || inText {
				injector.Write([]byte(fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)))
			}
			reason := "end_turn"
			if *chunk.Choices[0].FinishReason == "tool_calls" {
				reason = "tool_use"
			}
			injector.Write([]byte(fmt.Sprintf("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"%s\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n", reason)))
		}
	}
	return scanner.Err()
}
