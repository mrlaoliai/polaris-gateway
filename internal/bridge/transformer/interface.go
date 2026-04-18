// 内部使用：internal/bridge/transformer/interface.go
package transformer

import (
	"context"
	"io"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

// SemanticTransformer 定义了跨厂商协议对齐的标准接口
type SemanticTransformer interface {
	// TransformRequest 将客户端原生 HTTP Payload 转换为标准化请求
	TransformRequest(payload []byte) (*schema.StandardRequest, error)

	// TransformStream 处理物理模型的 SSE 流，进行协议重写、心跳注入和 Zero-Poetry 过滤
	TransformStream(ctx context.Context, physicalStream io.Reader, clientStream io.Writer) error
}
