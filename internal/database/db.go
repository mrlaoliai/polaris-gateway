package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // 坚持 Zero-CGO
)

func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	// 开启 WAL 模式以支持高并发读写，满足 2026 年 Agent 的高频访问
	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`)
	if err != nil {
		return nil, fmt.Errorf("配置 WAL 模式失败: %v", err)
	}

	// 执行自动迁移
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %v", err)
	}

	return db, nil
}
