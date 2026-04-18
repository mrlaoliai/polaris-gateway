// 内部使用：internal/bridge/transformer/anthropic.go
// 作者：mrlaoliai
// 设计哲学：协议转换中台，支持 Thinking 过程流式注入，采用 Sticky Error 模式满足 Linter
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
	"github.com/mrlaoliai/polaris-gateway/internal/bridge/modality"
	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
)

type AnthropicTransformer struct {
	targetModel string
	mcpRouter   *MCPRouter
	transcoder  *modality.Transcoder
}

func NewAnthropicTransformer(target string) *AnthropicTransformer {
	return &AnthropicTransformer{
		targetModel: target,
		mcpRouter:   NewMCPRouter(),
		transcoder:  modality.NewTranscoder(),
	}
}

func (t *AnthropicTransformer) TransformRequest(payload []byte) (*schema.StandardRequest, error) {
	var claudeReq struct {
		Model    string           `json:"model"`
		System   string           `json:"system"`
		Messages []schema.Message `json:"messages"`
		Stream   bool             `json:"stream"`
		Tools    []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"input_schema"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(payload, &claudeReq); err != nil {
		return nil, err
	}

	processedMsgs, _ := t.transcoder.ProcessMessages(claudeReq.Messages, false)

	stdReq := &schema.StandardRequest{
		Model:    t.targetModel,
		Stream:   claudeReq.Stream,
		Thinking: true,
	}

	if claudeReq.System != "" {
		stdReq.Messages = append(stdReq.Messages, schema.Message{Role: "system", Content: claudeReq.System})
	}
	stdReq.Messages = append(stdReq.Messages, processedMsgs...)

	for _, tool := range claudeReq.Tools {
		stdReq.Tools = append(stdReq.Tools, schema.Tool{
			Type: "function",
			Function: struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Parameters  any    `json:"parameters"`
			}{Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema},
		})
	}

	return stdReq, nil
}

func (t *AnthropicTransformer) TransformStream(ctx context.Context, physicalStream io.Reader, clientStream io.Writer) error {
	injector := heartbeat.NewInjector(clientStream, 15*time.Second, "anthropic")
	injector.Start(ctx)
	defer injector.Stop()

	scanner := bufio.NewScanner(physicalStream)
	zpFilter := middleware.NewZeroPoetryProcessor()

	// 1. 发送握手包
	msgStart := fmt.Sprintf("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"polaris_tx\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"%s\",\"content\":[]}}\n\n", t.targetModel)
	if _, err := injector.Write([]byte(msgStart)); err != nil {
		return err
	}

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

		// 2. Thinking 块处理
		if delta.ReasoningContent != "" {
			if !inThinking {
				msg := fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\"}}\n\n", blockIndex)
				if _, err := injector.Write([]byte(msg)); err != nil {
					return err
				}
				inThinking = true
			}
			safeT, _ := json.Marshal(delta.ReasoningContent)
			msg := fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":%d,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":%s}}\n\n", blockIndex, string(safeT))
			if _, err := injector.Write([]byte(msg)); err != nil {
				return err
			}
		}

		// 3. Text 块处理
		if delta.Content != "" {
			if inThinking {
				msg := fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
				if _, err := injector.Write([]byte(msg)); err != nil {
					return err
				}
				inThinking = false
				blockIndex++
			}
			if !inText {
				msg := fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n", blockIndex)
				if _, err := injector.Write([]byte(msg)); err != nil {
					return err
				}
				inText = true
			}
			clean := zpFilter.Process(delta.Content)
			if clean != "" {
				safeC, _ := json.Marshal(clean)
				msg := fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":%d,\"delta\":{\"type\":\"text_delta\",\"text\":%s}}\n\n", blockIndex, string(safeC))
				if _, err := injector.Write([]byte(msg)); err != nil {
					return err
				}
			}
		}

		// 4. 终结逻辑
		if chunk.Choices[0].FinishReason != nil {
			if inThinking || inText {
				msg := fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
				if _, err := injector.Write([]byte(msg)); err != nil {
					return err
				}
			}
			reason := "end_turn"
			if *chunk.Choices[0].FinishReason == "tool_calls" {
				reason = "tool_use"
			}
			msg := fmt.Sprintf("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"%s\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n", reason)
			if _, err := injector.Write([]byte(msg)); err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// 最终检查 Injector 捕获的粘性错误
	return injector.Err()
}
