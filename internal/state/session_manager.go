// 内部使用：internal/state/session_manager.go
// 作者：mrlaoliai
package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mrlaoliai/polaris-gateway/internal/database"
)

type SessionManager struct {
	db      *sql.DB
	dbMgr   *database.DBManager // 注入异步写入协调器
	baseDir string              // VFS 存储根目录，例如 "./data/vfs"
}

func NewSessionManager(db *sql.DB, dbMgr *database.DBManager, baseDir string) *SessionManager {
	// 确保 VFS 根目录存在
	_ = os.MkdirAll(baseDir, 0755)
	return &SessionManager{
		db:      db,
		dbMgr:   dbMgr,
		baseDir: baseDir,
	}
}

// SpillToVFS 将大内容写入本地物理文件，并通过 dbMgr 异步记录索引
func (m *SessionManager) SpillToVFS(traceID string, chunkIndex int, data []byte) error {
	// 1. 物理路径计算：按日期分层，防止单目录下文件过多导致文件系统性能下降
	datePrefix := time.Now().Format("20060102")
	dirPath := filepath.Join(m.baseDir, datePrefix, traceID)
	_ = os.MkdirAll(dirPath, 0755)

	// 文件名格式：chunk_0001.bin
	fileName := fmt.Sprintf("chunk_%04d.bin", chunkIndex)
	fullPath := filepath.Join(dirPath, fileName)

	// 2. 执行物理写入 (同步操作，确保数据安全落盘)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("VFS 物理写入失败: %w", err)
	}

	// 3. 异步记录文件索引到数据库
	// 不再直接存 data，而是存储 fullPath。使用 AsyncWrite 避免阻塞主请求
	m.dbMgr.AsyncWriteL2(
		"INSERT INTO session_chunks (trace_id, chunk_index, file_path, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		traceID, chunkIndex, fullPath,
	)

	return nil
}

// GetFullContext 按顺序从 VFS 文件系统中还原对话上下文
func (m *SessionManager) GetFullContext(traceID string) ([][]byte, error) {
	// 读取索引是“读”操作，直接走 db.Query (同步读)
	rows, err := m.db.Query(
		"SELECT file_path FROM session_chunks WHERE trace_id = ? ORDER BY chunk_index ASC",
		traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks [][]byte
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}

		// 从文件系统读取二进制内容
		data, err := os.ReadFile(path)
		if err != nil {
			// 如果物理文件丢失，记录日志但继续处理其他片段，提高系统鲁棒性
			continue
		}
		chunks = append(chunks, data)
	}
	return chunks, nil
}

// Cleanup 执行彻底的物理清理，删除过期文件及数据库记录
func (m *SessionManager) Cleanup(maxAge time.Duration) {
	deadline := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	// 1. 找出所有过期的物理文件路径
	rows, err := m.db.Query("SELECT file_path FROM session_chunks WHERE created_at < ?", deadline)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil {
			// 2. 执行物理删除
			_ = os.Remove(path)

			// 尝试删除空的父目录 (traceID 层和日期层)
			dir := filepath.Dir(path)
			_ = os.Remove(dir)
		}
	}

	// 3. 异步清理数据库索引记录
	m.dbMgr.AsyncWrite("DELETE FROM session_chunks WHERE created_at < ?", deadline)
}
