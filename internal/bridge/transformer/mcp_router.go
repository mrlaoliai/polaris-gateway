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
		// 校验工具意图，这里可以无缝接入 V3 文档中的 Dynamic Tool Pruning 逻辑
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
// 例如：OpenAI 的 tool_calls 数组 -> Anthropic 的 tool_use 内容块
func (r *MCPRouter) ParseToolCallResponse(physicalFormat []byte, targetProtocol string) ([]byte, error) {
	if targetProtocol == "anthropic" {
		// 解析 OpenAI 的返回
		var oaiResp struct {
			Choices []struct {
				Message struct {
					ToolCalls []struct {
						ID       string `json:"id"`
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

		// 转换为 Anthropic 格式
		if len(oaiResp.Choices) > 0 && len(oaiResp.Choices[0].Message.ToolCalls) > 0 {
			tc := oaiResp.Choices[0].Message.ToolCalls[0]
			anthropicResp := map[string]interface{}{
				"type": "tool_use",
				"id":   tc.ID,
				"name": tc.Function.Name,
				// 修复：必须先转换为 []byte 才能包装为 RawMessage
				"input": json.RawMessage([]byte(tc.Arguments)),
			}
			return json.Marshal(anthropicResp)
		}
	}

	return physicalFormat, nil
}
