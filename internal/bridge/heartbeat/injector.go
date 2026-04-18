// 内部使用：internal/bridge/heartbeat/injector.go
// 作者：mrlaoliai
package heartbeat

import (
	"context"
	"io"
	"sync"
	"time"
)

// Injector 负责在 SSE 流中原子化地注入保活心跳
type Injector struct {
	mu           sync.Mutex
	clientWriter io.Writer
	interval     time.Duration
	payload      []byte
	stopCh       chan struct{}
	stopOnce     sync.Once // 确保 Stop 操作的幂等性
}

// NewInjector 根据协议类型初始化注入器
func NewInjector(w io.Writer, interval time.Duration, protocolType string) *Injector {
	var payload []byte
	if protocolType == "anthropic" {
		// Anthropic 兼容的 SSE 规范：冒号开头代表注释，客户端会忽略但连接保持活跃
		payload = []byte(": keep-alive\n\n")
	} else {
		// 通用 SSE 格式的空数据包
		payload = []byte("data: {}\n\n")
	}

	return &Injector{
		clientWriter: w,
		interval:     interval,
		payload:      payload,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动异步注入协程
func (h *Injector) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.stopCh:
				return
			case <-ticker.C:
				if err := h.inject(); err != nil {
					// 如果写入失败（通常是连接已断开），则自动停止注入
					return
				}
			}
		}
	}()
}

// inject 执行原子的心跳写入
func (h *Injector) inject() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.clientWriter.Write(h.payload)
	if err != nil {
		return err
	}

	// 执行强制刷新
	if f, ok := h.clientWriter.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}

// Write 由外部 Transformer 调用，用于写入真实的模型数据
// 此方法与心跳注入共用同一把互斥锁，确保数据块不被切断
func (h *Injector) Write(p []byte) (n int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	n, err = h.clientWriter.Write(p)
	if f, ok := h.clientWriter.(interface{ Flush() }); ok {
		f.Flush()
	}
	return n, err
}

// Stop 安全停止注入器
func (h *Injector) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}
