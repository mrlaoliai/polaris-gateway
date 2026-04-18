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

func (r *MCPRouter) ParseToolCallResponse(physicalFormat []byte, targetProtocol string) ([]byte, error) {
	if targetProtocol == "anthropic" {
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

		if len(oaiResp.Choices) > 0 && len(oaiResp.Choices[0].Message.ToolCalls) > 0 {
			tc := oaiResp.Choices[0].Message.ToolCalls[0]
			anthropicResp := map[string]interface{}{
				"type": "tool_use",
				"id":   tc.ID,
				"name": tc.Function.Name,
				// 修复：正确的嵌套字段访问路径为 tc.Function.Arguments
				"input": json.RawMessage([]byte(tc.Function.Arguments)),
			}
			return json.Marshal(anthropicResp)
		}
	}

	return physicalFormat, nil
}
