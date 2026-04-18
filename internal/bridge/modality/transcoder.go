// 内部使用：internal/bridge/modality/transcoder.go
// 作者：mrlaoliai
package modality

import (
	"encoding/json"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

// Transcoder 负责跨厂商、跨模型的多模态负载转换与降级
type Transcoder struct{}

func NewTranscoder() *Transcoder {
	return &Transcoder{}
}

// ProcessMessages 针对消息队列执行多模态对齐或降级
func (t *Transcoder) ProcessMessages(messages []schema.Message, physicalModelSupportsVision bool) ([]schema.Message, error) {
	var processed []schema.Message

	for _, msg := range messages {
		// 快速探测：如果内容中完全不含多模态特征，直接透传，避免开销
		if !strings.Contains(msg.Content, "image") && !strings.Contains(msg.Content, "data:") {
			processed = append(processed, msg)
			continue
		}

		// 场景 A：物理模型支持视觉 -> 执行格式对齐（存根）
		if physicalModelSupportsVision {
			// 此处未来可扩展 OpenAI -> Anthropic 的图片 JSON 树转换逻辑
			processed = append(processed, msg)
			continue
		}

		// 场景 B：物理模型不支持视觉 -> 执行安全剥离
		cleanText, stripped := t.SafeStripImages(msg.Content)
		if stripped {
			// 追加系统标记，告知大模型图片已被网关过滤
			cleanText += "\n\n[System Note: User attached an image, but it was stripped because the current target model does not support vision.]"
		}

		processed = append(processed, schema.Message{
			Role:    msg.Role,
			Content: cleanText,
		})
	}

	return processed, nil
}

// SafeStripImages 尝试深度解析多模态结构，提取纯文本
func (t *Transcoder) SafeStripImages(rawContent string) (string, bool) {
	if rawContent == "" {
		return "", false
	}

	// 1. 尝试作为 JSON 数组解析 (OpenAI/Anthropic 标准)
	var contentArray []map[string]interface{}
	if err := json.Unmarshal([]byte(rawContent), &contentArray); err == nil {
		var textParts []string
		stripped := false

		for _, block := range contentArray {
			contentType, _ := block["type"].(string)
			if contentType == "text" {
				if text, ok := block["text"].(string); ok {
					textParts = append(textParts, text)
				}
			} else if contentType == "image_url" || contentType == "image" {
				stripped = true
			}
		}
		return strings.Join(textParts, "\n"), stripped
	}

	// 2. 尝试作为单 JSON 对象解析
	var contentObj map[string]interface{}
	if err := json.Unmarshal([]byte(rawContent), &contentObj); err == nil {
		if contentType, _ := contentObj["type"].(string); contentType == "text" {
			return contentObj["text"].(string), false
		} else {
			return "", true
		}
	}

	// 3. 兜底方案：如果是纯文本但含有 Base64 注入，执行暴力清理
	// 针对非标准客户端直接在文本中嵌入 data:image 的情况
	if strings.Contains(rawContent, "data:image/") {
		// 这里执行一个简单的字符串查找截断，保护内存
		idx := strings.Index(rawContent, "data:image/")
		return strings.TrimSpace(rawContent[:idx]), true
	}

	return rawContent, false
}
