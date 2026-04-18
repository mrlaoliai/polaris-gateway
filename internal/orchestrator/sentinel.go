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

// Start 启动后台拨测循环
func (s *Sentinel) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // 每 5 分钟检查一次密钥池
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
	// 1. 从 DB 获取所有活跃账号
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

		// 2. 执行物理拨测 (这里模拟调用厂商的 /models 接口或最小量请求)
		go func(accountID int, key string) {
			if !s.pingProvider(key) {
				log.Printf("[Sentinel] 账号 [%d] 拨测失败，执行下线处理", accountID)
				s.db.Exec("UPDATE accounts SET status = 'error' WHERE id = ?", accountID)
			}
		}(id, apiKey)
	}
}

func (s *Sentinel) pingProvider(apiKey string) bool {
	// 实际代码中，这里会调用 pkg/provider 下的对应厂商执行器
	// 为了演示，这里假设拨测逻辑已通
	return true
}
