package modality

import (
	"encoding/json"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

type Transcoder struct{}

func NewTranscoder() *Transcoder {
	return &Transcoder{}
}

func (t *Transcoder) ProcessMessages(messages []schema.Message, physicalModelSupportsVision bool) ([]schema.Message, error) {
	if physicalModelSupportsVision {
		return messages, nil // 物理模型支持视觉，直接放行
	}

	var processed []schema.Message
	for _, msg := range messages {
		// 检查内容是否可能包含多模态 JSON 数组的特征
		if strings.Contains(msg.Content, "image_url") || strings.Contains(msg.Content, "data:image") {
			cleanText, stripped := t.SafeStripImages(msg.Content)
			if stripped {
				cleanText += "\n\n[System Note: User attached an image, but it was safely stripped because the current fallback model does not support vision.]"
			}
			processed = append(processed, schema.Message{
				Role:    msg.Role,
				Content: cleanText,
			})
		} else {
			processed = append(processed, msg)
		}
	}
	return processed, nil
}

// SafeStripImages 尝试解析多模态 JSON 结构，安全剔除图片，保留纯文本
func (t *Transcoder) SafeStripImages(rawContent string) (string, bool) {
	var contentArray []map[string]interface{}

	// 尝试作为 JSON 数组解析 (OpenAI/Anthropic 多模态标准)
	if err := json.Unmarshal([]byte(rawContent), &contentArray); err != nil {
		// 如果不是标准 JSON 数组，可能是早期 Base64 直接拼接，执行后备方案
		idx := strings.Index(rawContent, "data:image/")
		if idx != -1 {
			return rawContent[:idx], true
		}
		return rawContent, false
	}

	var textParts []string
	stripped := false

	// 遍历结构体，仅提取 type 为 text 的内容
	for _, block := range contentArray {
		if blockType, ok := block["type"].(string); ok {
			if blockType == "text" {
				if textData, ok := block["text"].(string); ok {
					textParts = append(textParts, textData)
				}
			} else if blockType == "image_url" || blockType == "image" {
				stripped = true // 标记发现并剔除了图片
			}
		}
	}

	return strings.Join(textParts, "\n"), stripped
}
