// internal/orchestrator/sentinel.go
package orchestrator

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type Sentinel struct {
	db *sql.DB
}

func NewSentinel(db *sql.DB) *Sentinel {
	return &Sentinel{db: db}
}

func (s *Sentinel) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAccounts(ctx)
		}
	}
}

func (s *Sentinel) checkAccounts(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, api_key, provider_id FROM accounts WHERE status = 'active'")
	if err != nil {
		log.Printf("[Sentinel] 获取账号列表失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var apiKey, providerID string
		if err := rows.Scan(&id, &apiKey, &providerID); err != nil {
			continue
		}

		go func(accountID int, key string) {
			if !s.pingProvider(key) {
				log.Printf("[Sentinel] 账号 [%d] 拨测失败，执行下线处理", accountID)

				// 修复 errcheck: 捕获并处理 Exec 的返回值
				_, err := s.db.Exec("UPDATE accounts SET status = 'error' WHERE id = ?", accountID)
				if err != nil {
					log.Printf("[Sentinel] 账号 [%d] 状态更新为 error 失败: %v", accountID, err)
				}
			}
		}(id, apiKey)
	}
}

func (s *Sentinel) pingProvider(apiKey string) bool {
	return true
}
