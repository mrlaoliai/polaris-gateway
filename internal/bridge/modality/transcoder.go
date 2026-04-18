// 内部使用：internal/bridge/modality/transcoder.go
// 作者：mrlaoliai
package modality

import (
	"fmt"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

// Transcoder 负责跨模型的多模态负载转换
type Transcoder struct{}

func NewTranscoder() *Transcoder {
	return &Transcoder{}
}

// ProcessMessages 扫描并处理消息中的视觉元素
func (t *Transcoder) ProcessMessages(messages []schema.Message, physicalModelSupportsVision bool) ([]schema.Message, error) {
	var processed []schema.Message

	for _, msg := range messages {
		// 简单的启发式检查：如果内容包含 Base64 图片标识
		if strings.Contains(msg.Content, "data:image/") {
			if !physicalModelSupportsVision {
				// 触发 Multi-Modal Fallback：剥离图片，保留文本，并在末尾追加系统提示
				cleanText := t.StripImagePayload(msg.Content)
				fallbackMsg := fmt.Sprintf("%s\n[System Note: User attached an image, but it was stripped because the current routing target does not support vision.]", cleanText)

				processed = append(processed, schema.Message{
					Role:    msg.Role,
					Content: fallbackMsg,
				})
				continue
			}

			// 如果支持视觉，则执行具体的协议图片格式对齐（如 URL 转 Base64，或重构 JSON 树）
			// 此处省略具体的 Base64 重新编码逻辑，保持架构清晰
		}
		processed = append(processed, msg)
	}

	return processed, nil
}

// StripImagePayload 是一个防御性函数，用于清洗大体积的 Base64 负载
func (t *Transcoder) StripImagePayload(rawContent string) string {
	// 简单的截断逻辑推演：定位 "data:image/" 并移除到结束引号或括号
	// 实际生产中应解析具体的 JSON 结构块
	idx := strings.Index(rawContent, "data:image/")
	if idx != -1 {
		return rawContent[:idx] + "[Image Payload Removed]"
	}
	return rawContent
}
