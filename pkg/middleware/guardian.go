// 内部使用：pkg/middleware/guardian.go
// 作者：mrlaoliai
package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/mrlaoliai/polaris-gateway/internal/database"
)

type contextKey string

const (
	GatewayKeyID contextKey = "gateway_key_id"
)

type Guardian struct {
	db    *sql.DB
	dbMgr *database.DBManager // 注入协调器
}

func NewGuardian(db *sql.DB, dbMgr *database.DBManager) *Guardian {
	return &Guardian{db: db, dbMgr: dbMgr}
}

// AuthAndQuotaMiddleware 执行入口鉴权与静态配额预检
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

		// 配额预检
		if dailyLimit != -1 && usedTokens >= dailyLimit {
			http.Error(w, "Daily Quota Exceeded", http.StatusTooManyRequests)
			return
		}

		// 将 KeyID 注入 Context，方便后续链路记录使用量
		ctx := context.WithValue(r.Context(), GatewayKeyID, keyID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RecordUsage 异步记录 Token 使用量 (由 main.go 或 Transformer 在响应结束后调用)
func (g *Guardian) RecordUsage(mgr *database.DBManager, keyID int, tokens int) {
	mgr.AsyncWrite("UPDATE gateway_keys SET used_tokens = used_tokens + ? WHERE id = ?", tokens, keyID)
}
