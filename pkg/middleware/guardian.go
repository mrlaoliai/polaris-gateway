// 基础中间件：pkg/middleware/guardian.go
// 作者：mrlaoliai
package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
)

type Guardian struct {
	db *sql.DB
}

func NewGuardian(db *sql.DB) *Guardian {
	return &Guardian{db: db}
}

// AuthAndQuotaMiddleware 拦截器：校验逻辑 Key 并检查额度
func (g *Guardian) AuthAndQuotaMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		logicalKey := strings.TrimPrefix(authHeader, "Bearer ")

		// State-in-DB: 查询网关凭证
		var keyID int
		var dailyLimit, usedTokens int
		err := g.db.QueryRow(
			"SELECT id, daily_limit, used_tokens FROM gateway_keys WHERE key_value = ?",
			logicalKey,
		).Scan(&keyID, &dailyLimit, &usedTokens)

		if err == sql.ErrNoRows {
			http.Error(w, "Invalid Gateway Key", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// 额度校验 (-1 代表无限)
		if dailyLimit != -1 && usedTokens >= dailyLimit {
			http.Error(w, "Daily Quota Exceeded", http.StatusTooManyRequests)
			return
		}

		// 将 Key ID 注入上下文，供后续审计 (usage_stats) 扣费使用
		ctx := context.WithValue(r.Context(), "gateway_key_id", keyID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
