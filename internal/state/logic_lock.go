package state

import (
	"context"
	"database/sql"
	"fmt"
)

// LockManager 处理工具调用期间的资源锁定
type LockManager struct {
	db *sql.DB
}

func NewLockManager(db *sql.DB) *LockManager {
	return &LockManager{db: db}
}

// AcquireLock 尝试获取资源锁。如果资源已被占用，则根据策略等待或返回错误。
// 这里的 resource 可能是文件路径或逻辑对象 ID
func (m *LockManager) AcquireLock(ctx context.Context, sessionID, resource string) error {
	// 使用 State-in-DB 哲学，锁状态持久化在 SQLite 中
	// 尝试插入锁记录，利用 UNIQUE 约束实现原子锁
	_, err := m.db.ExecContext(ctx,
		"INSERT INTO logic_locks (resource_id, session_id, locked_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		resource, sessionID)

	if err != nil {
		// 如果插入失败，说明资源已被锁定
		return fmt.Errorf("资源 [%s] 正在被其他工具调用占用，请稍后重试", resource)
	}
	return nil
}

// ReleaseLock 释放资源
func (m *LockManager) ReleaseLock(ctx context.Context, resource string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM logic_locks WHERE resource_id = ?", resource)
	return err
}
