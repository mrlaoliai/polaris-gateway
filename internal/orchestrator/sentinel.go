// 内部使用：internal/orchestrator/sentinel.go
// 作者：mrlaoliai
package orchestrator

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/mrlaoliai/polaris-gateway/internal/database"
)

type Sentinel struct {
	db     *sql.DB
	dbMgr  *database.DBManager // 注入协调器
	client *http.Client        // [优化] 复用 Client 提升连接效率
}

func NewSentinel(db *sql.DB, dbMgr *database.DBManager) *Sentinel {
	return &Sentinel{
		db: db,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 10,
			},
		},
		dbMgr: dbMgr,
	}
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
	// [已拦截，被动错误收集由 picker.go 的 ReportKeyError 替代]
	// 旧的 `accounts` 和 `providers` 表已经移除。
}
