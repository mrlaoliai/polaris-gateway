// 内部使用：internal/bridge/schema/types.go
// 作者：mrlaoliai
package schema

// StandardRequest 定义了 Polaris 网关内部的标准化请求规范
type StandardRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	// Bifrost 2.0 扩展字段
	Thinking bool   `json:"thinking,omitempty"` // 指示是否开启思维链
	Tools    []Tool `json:"tools,omitempty"`    // MCP 工具集
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  any    `json:"parameters"`
	} `json:"function"`
}

// StreamEvent 定义标准化流事件，用于 Heartbeat Injector 和 L2 Buffer 重排
type StreamEvent struct {
	ID        string `json:"id"`
	DeltaType string `json:"delta_type"` // "content", "thought", "tool_call", "heartbeat"
	Content   string `json:"content"`
}
