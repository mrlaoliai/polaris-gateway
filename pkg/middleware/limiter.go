// 内部使用：pkg/middleware/limiter.go
package middleware

import (
	"net/http"
)

// ConcurrentLimiter 限制全局同时处理的请求数量
type ConcurrentLimiter struct {
	sem chan struct{}
}

func NewConcurrentLimiter(limit int) *ConcurrentLimiter {
	return &ConcurrentLimiter{
		sem: make(chan struct{}, limit),
	}
}

func (l *ConcurrentLimiter) LimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case l.sem <- struct{}{}:
			// 成功获取令牌
			defer func() { <-l.sem }() // 请求结束释放令牌
			next.ServeHTTP(w, r)
		default:
			// 令牌桶已满，拒绝访问
			http.Error(w, "Server Busy: Max Concurrency Reached", http.StatusTooManyRequests)
		}
	}
}
