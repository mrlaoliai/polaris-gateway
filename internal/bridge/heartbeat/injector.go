// 内部使用：internal/bridge/heartbeat/injector.go
// 作者：mrlaoliai
package heartbeat

import (
	"context"
	"io"
	"sync"
	"time"
)

// Injector 管理 SSE 流的心跳注入，通过互斥锁实现流控
type Injector struct {
	mu           sync.Mutex
	clientWriter io.Writer
	interval     time.Duration
	payload      []byte
	stopCh       chan struct{}
}

// NewInjector 初始化心跳注入器
func NewInjector(w io.Writer, interval time.Duration, protocolType string) *Injector {
	var payload []byte
	if protocolType == "anthropic" {
		// Anthropic 兼容的静默保活事件
		payload = []byte(": keep-alive\n\n")
	} else {
		// 通用 SSE 格式的保活空指令
		payload = []byte("data: {}\n\n")
	}

	return &Injector{
		clientWriter: w,
		interval:     interval,
		payload:      payload,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动后台独立协程执行注入
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
				h.inject()
			}
		}
	}()
}

// inject 执行原子的写入与缓冲区刷新
func (h *Injector) inject() {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, _ = h.clientWriter.Write(h.payload)
	// 如果底层连接支持主动 Flush，则强制推送至 TCP 栈
	if f, ok := h.clientWriter.(interface{ Flush() }); ok {
		f.Flush()
	}
}

// Write 拦截物理模型的真实数据流，复用同一把锁以防止数据交错
func (h *Injector) Write(p []byte) (n int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	n, err = h.clientWriter.Write(p)
	if f, ok := h.clientWriter.(interface{ Flush() }); ok {
		f.Flush()
	}
	return n, err
}

// Stop 释放系统资源
func (h *Injector) Stop() {
	close(h.stopCh)
}
