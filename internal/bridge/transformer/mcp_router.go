// 内部使用：internal/bridge/transformer/mcp_router.go
// 作者：mrlaoliai
package transformer

import (
	"encoding/json"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

// MCPRouter 负责在不同的厂商工具调用格式之间进行对齐
type MCPRouter struct{}

func NewMCPRouter() *MCPRouter {
	return &MCPRouter{}
}

// FormatToOpenAI 将标准 MCP 工具定义转换为 OpenAI 的 Function Calling 格式
func (r *MCPRouter) FormatToOpenAI(tools []schema.Tool) ([]map[string]interface{}, error) {
	var openAITools []map[string]interface{}
	for _, tool := range tools {
		openAITools = append(openAITools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		})
	}
	return openAITools, nil
}

// ParseToolCallResponse 将底层物理模型返回的工具调用指令转回客户端预期的格式
func (r *MCPRouter) ParseToolCallResponse(physicalFormat []byte, targetProtocol string) ([]byte, error) {
	if targetProtocol == "anthropic" {
		var oaiResp struct {
			Choices []struct {
				Message struct {
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(physicalFormat, &oaiResp); err != nil {
			return nil, err
		}

		// 修复：支持多个并行工具调用转换
		if len(oaiResp.Choices) > 0 {
			toolCalls := oaiResp.Choices[0].Message.ToolCalls
			if len(toolCalls) > 0 {
				var anthropicBlocks []map[string]interface{}
				for _, tc := range toolCalls {
					anthropicBlocks = append(anthropicBlocks, map[string]interface{}{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": json.RawMessage([]byte(tc.Function.Arguments)),
					})
				}
				return json.Marshal(anthropicBlocks)
			}
		}
	}
	// TODO: 增加 Gemini (function_calls) 到 OpenAI/Anthropic 的互转
	return physicalFormat, nil
}
