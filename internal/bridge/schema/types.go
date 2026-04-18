// 内部使用：internal/bridge/schema/types.go
// 作者：mrlaoliai
package schema

// StandardRequest 定义了 Polaris 网关内部的标准化请求规范
type StandardRequest struct {
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	Temperature   float32   `json:"temperature,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
	MaxTokens     int       `json:"max_tokens,omitempty"`     // [新增] 显式控制生成长度
	StopSequences []string  `json:"stop_sequences,omitempty"` // [新增] 支持停止词配置

	// Bifrost 2.0 扩展字段
	Thinking bool   `json:"thinking,omitempty"` // 指示是否开启思维链 (Shadow Signature)
	Tools    []Tool `json:"tools,omitempty"`    // MCP 协议兼容工具集
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
	Index     int    `json:"index"`      // [新增] 序列索引，用于 L2 溢出缓冲在高并发下的数据对齐
	DeltaType string `json:"delta_type"` // "text_delta", "thinking_delta", "tool_use", "heartbeat"
	Content   string `json:"content"`
}
