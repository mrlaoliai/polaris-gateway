// 内部使用：internal/state/session_manager.go
// 作者：mrlaoliai
package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SessionManager struct {
	db      *sql.DB
	baseDir string // VFS 根目录，例如 "./data/l2_storage"
}

func NewSessionManager(db *sql.DB, baseDir string) *SessionManager {
	// 确保 VFS 根目录存在
	_ = os.MkdirAll(baseDir, 0755)
	return &SessionManager{
		db:      db,
		baseDir: baseDir,
	}
}

// SpillToVFS 将大内容写入本地文件，并在 DB 中记录索引
func (m *SessionManager) SpillToVFS(traceID string, chunkIndex int, data []byte) error {
	// 1. 计算物理存储路径：按日期分层防止单个目录下文件过多
	datePrefix := time.Now().Format("20060102")
	dirPath := filepath.Join(m.baseDir, datePrefix, traceID)
	_ = os.MkdirAll(dirPath, 0755)

	fileName := fmt.Sprintf("chunk_%04d.bin", chunkIndex)
	fullPath := filepath.Join(dirPath, fileName)

	// 2. 写入文件 (Zero-CGO 的标准库操作)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("文件写入失败: %w", err)
	}

	// 3. 在 DB 中记录文件引用，而不是原始数据
	query := `INSERT INTO session_chunks (trace_id, chunk_index, file_path, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := m.db.Exec(query, traceID, chunkIndex, fullPath)
	return err
}

// GetFullContext 从 VFS 还原对话上下文
func (m *SessionManager) GetFullContext(traceID string) ([][]byte, error) {
	rows, err := m.db.Query("SELECT file_path FROM session_chunks WHERE trace_id = ? ORDER BY chunk_index ASC", traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks [][]byte
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}

		// 从文件读取数据
		data, err := os.ReadFile(path)
		if err != nil {
			continue // 允许个别文件缺失，提高容错
		}
		chunks = append(chunks, data)
	}
	return chunks, nil
}

// Cleanup 物理删除文件及其 DB 记录
func (m *SessionManager) Cleanup(maxAge time.Duration) {
	deadline := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	// 查出要删除的文件路径
	rows, _ := m.db.Query("SELECT file_path FROM session_chunks WHERE created_at < ?", deadline)
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil {
			_ = os.Remove(path) // 物理删除
		}
	}

	// 清理 DB 记录
	_, _ = m.db.Exec("DELETE FROM session_chunks WHERE created_at < ?", deadline)
}
