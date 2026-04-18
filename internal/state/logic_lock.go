// 内部使用：internal/state/logic_lock.go
// 作者：mrlaoliai
package state

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type LockManager struct {
	db *sql.DB
}

func NewLockManager(db *sql.DB) *LockManager {
	return &LockManager{db: db}
}

// AcquireLock 尝试获取资源锁（带 TTL 租约与防僵尸锁清理）
func (m *LockManager) AcquireLock(ctx context.Context, sessionID, resource string, ttl time.Duration) error {
	// 1. 先清理该资源可能存在的已过期僵尸锁
	_, _ = m.db.ExecContext(ctx, "DELETE FROM logic_locks WHERE resource_id = ? AND expires_at < CURRENT_TIMESTAMP", resource)

	expiresAt := time.Now().Add(ttl).Format("2006-01-02 15:04:05")

	// 2. 尝试获取锁
	// 如果同一 session 已经持有该锁，则执行 UPSERT 更新过期时间（支持续租/重入）
	query := `
		INSERT INTO logic_locks (resource_id, session_id, locked_at, expires_at) 
		VALUES (?, ?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(resource_id) DO UPDATE SET 
			expires_at = excluded.expires_at,
			locked_at = CURRENT_TIMESTAMP
		WHERE session_id = ? OR expires_at < CURRENT_TIMESTAMP
	`
	res, err := m.db.ExecContext(ctx, query, resource, sessionID, expiresAt, sessionID)
	if err != nil {
		return fmt.Errorf("系统错误，无法操作锁表: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("资源 [%s] 目前正被其他会话占用", resource)
	}

	return nil
}

func (m *LockManager) ReleaseLock(ctx context.Context, resource string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM logic_locks WHERE resource_id = ?", resource)
	return err
}
