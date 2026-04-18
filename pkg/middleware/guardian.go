// pkg/middleware/guardian.go
package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
)

// 定义未导出的自定义类型作为 Context Key，彻底杜绝 SA1029 碰撞风险
type contextKey string

const (
	gatewayKeyID contextKey = "gateway_key_id"
)

type Guardian struct {
	db *sql.DB
}

func NewGuardian(db *sql.DB) *Guardian {
	return &Guardian{db: db}
}

func (g *Guardian) AuthAndQuotaMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		logicalKey := strings.TrimPrefix(authHeader, "Bearer ")

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

		if dailyLimit != -1 && usedTokens >= dailyLimit {
			http.Error(w, "Daily Quota Exceeded", http.StatusTooManyRequests)
			return
		}

		// 修复 staticcheck SA1029: 使用自定义类型的常量作为 Key
		ctx := context.WithValue(r.Context(), gatewayKeyID, keyID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
