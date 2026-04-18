// 内部使用：internal/state/session_manager.go
// 作者：mrlaoliai
package state

import (
	"database/sql"
	"fmt"
	"time"
)

type SessionManager struct {
	db *sql.DB
}

func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{db: db}
}

// SpillToDisk 将长对话片段持久化到磁盘 L2 缓存
func (m *SessionManager) SpillToDisk(traceID string, chunkIndex int, data []byte) error {
	// 防御：单次写入超过 5MB 予以阻断，防止数据库膨胀异常
	if len(data) > 5*1024*1024 {
		return fmt.Errorf("数据片段过大 (%d bytes)，已触发熔断保护", len(data))
	}

	query := `INSERT INTO session_chunks (trace_id, chunk_index, payload, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := m.db.Exec(query, traceID, chunkIndex, data)
	if err != nil {
		return fmt.Errorf("L2 溢出写入失败: %w", err)
	}
	return nil
}

// GetFullContext 顺序还原对话上下文
func (m *SessionManager) GetFullContext(traceID string) ([][]byte, error) {
	rows, err := m.db.Query("SELECT payload FROM session_chunks WHERE trace_id = ? ORDER BY chunk_index ASC", traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks [][]byte
	for rows.Next() {
		var chunk []byte
		if err := rows.Scan(&chunk); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// CleanupSessions 清理陈旧的 L2 缓存
func (m *SessionManager) CleanupSessions(maxAge time.Duration) (int64, error) {
	deadline := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")
	res, err := m.db.Exec("DELETE FROM session_chunks WHERE created_at < ?", deadline)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
