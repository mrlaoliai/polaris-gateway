package provider

import (
	"context"
	"io"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/schema"
)

// Executor 是物理厂商 API 的最终执行者
type Executor interface {
	// Execute 发送非流式请求
	Execute(ctx context.Context, req *schema.StandardRequest) ([]byte, error)
	// ExecuteStream 发送流式请求并返回原始响应体
	ExecuteStream(ctx context.Context, req *schema.StandardRequest) (io.ReadCloser, error)
}
