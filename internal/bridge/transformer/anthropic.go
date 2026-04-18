package transformer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
)

// AnthropicTransformer 处理 Claude API 客户端与网关标准协议的双向转换
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

// TransformRequest: 接收 Claude Code 客户端发来的原生 JSON，转为标准请求
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
		return nil, fmt.Errorf("解析 Anthropic 协议失败: %w", err)
	}

	stdReq := &schema.StandardRequest{
		Model:    t.targetModel, // 物理路由映射，例如 deepseek-v4-reasoning
		Stream:   claudeReq.Stream,
		Thinking: true, // 强制开启思维链支持
	}

	// 处理 System Prompt
	if claudeReq.System != "" {
		stdReq.Messages = append(stdReq.Messages, schema.Message{
			Role:    "system",
			Content: claudeReq.System,
		})
	}

	// 处理历史对话
	for _, msg := range claudeReq.Messages {
		stdReq.Messages = append(stdReq.Messages, schema.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 处理 MCP 工具挂载
	for _, tool := range claudeReq.Tools {
		stdReq.Tools = append(stdReq.Tools, schema.Tool{
			Type: "function",
			Function: struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Parameters  any    `json:"parameters"`
			}{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return stdReq, nil
}

// TransformStream: 核心状态机。读取物理模型流 -> 触发过滤与 L2 缓冲 -> 转换为 Claude SSE 流
func (t *AnthropicTransformer) TransformStream(ctx context.Context, physicalStream io.Reader, clientStream io.Writer) error {
	scanner := bufio.NewScanner(physicalStream)

	// 1. 下发 Anthropic 握手包
	messageStart := fmt.Sprintf("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_polaris_trace\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"%s\",\"content\":[]}}\n\n", t.targetModel)
	_, _ = clientStream.Write([]byte(messageStart))

	var inThinkingBlock bool
	var inTextBlock bool
	var blockIndex int

	// 实例化 Zero-Poetry 过滤器
	zpFilter := middleware.NewZeroPoetryProcessor()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "[DONE]" {
			break
		}

		// 解析 OpenAI/DeepSeek 兼容的 Delta 数据
		var chunk struct {
			Choices []struct {
				Delta struct {
					ReasoningContent string `json:"reasoning_content"`
					Content          string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil || len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// 2. 拦截并处理 "影子签名" (Thinking Session)
		if delta.ReasoningContent != "" {
			if !inThinkingBlock {
				startEvent := fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\"}}\n\n", blockIndex)
				clientStream.Write([]byte(startEvent))
				inThinkingBlock = true
			}

			safeThinking, _ := json.Marshal(delta.ReasoningContent)
			deltaEvent := fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":%d,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":%s}}\n\n", blockIndex, string(safeThinking))
			clientStream.Write([]byte(deltaEvent))
		}

		// 3. 处理常规回复内容
		if delta.Content != "" {
			// 如果刚刚结束 Thinking，需要先闭合思维块
			if inThinkingBlock {
				stopEvent := fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
				clientStream.Write([]byte(stopEvent))
				inThinkingBlock = false
				blockIndex++
			}

			if !inTextBlock {
				startEvent := fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n", blockIndex)
				clientStream.Write([]byte(startEvent))
				inTextBlock = true
			}

			// 执行 Zero-Poetry 口癖清洗
			cleanText := zpFilter.Process(delta.Content)
			if cleanText != "" {
				safeContent, _ := json.Marshal(cleanText)
				deltaEvent := fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":%d,\"delta\":{\"type\":\"text_delta\",\"text\":%s}}\n\n", blockIndex, string(safeContent))
				clientStream.Write([]byte(deltaEvent))
			}
		}

		// 4. 处理终结状态
		if chunk.Choices[0].FinishReason != nil {
			if inThinkingBlock || inTextBlock {
				stopEvent := fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
				clientStream.Write([]byte(stopEvent))
			}

			reason := *chunk.Choices[0].FinishReason
			anthropicReason := "end_turn"
			if reason == "tool_calls" {
				anthropicReason = "tool_use"
			}

			msgStop := fmt.Sprintf("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"%s\",\"stop_sequence\":null}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n", anthropicReason)
			clientStream.Write([]byte(msgStop))
		}
	}

	return scanner.Err()
}
