package state

import (
	"database/sql"
	"fmt"
)

type SessionManager struct {
	db *sql.DB
}

func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{db: db}
}

// SpillToDisk 将内存中的 L1 缓冲溢出到 SQLite L2 表中
func (m *SessionManager) SpillToDisk(traceID string, chunkIndex int, data []byte) error {
	query := `INSERT INTO session_chunks (trace_id, chunk_index, payload) VALUES (?, ?, ?)`
	_, err := m.db.Exec(query, traceID, chunkIndex, data)
	if err != nil {
		return fmt.Errorf("L2 溢出写入失败: %w", err)
	}
	return nil
}

// GetFullContext 还原完整的上下文链路
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
