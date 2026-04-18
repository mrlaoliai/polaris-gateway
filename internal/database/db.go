// 内部使用：internal/database/db.go
// 作者：mrlaoliai
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // 坚持 Zero-CGO，使用纯 Go 实现驱动
)

func InitDB(dataSourceName string) (*sql.DB, error) {
	// [优化] 增加 busy_timeout 参数。在并发写冲突时，驱动会自动等待并重试
	// 这解决了 SQLite 在高并发下常见的 "database is locked" 错误
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)", dataSourceName)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// [优化] 针对本地数据库合理配置连接池
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 开启 WAL 模式：支持多读者与单写者并发，提升 Agent 状态同步效率
	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`)
	if err != nil {
		return nil, fmt.Errorf("配置 WAL 模式失败: %v", err)
	}

	// 执行自动迁移逻辑
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %v", err)
	}

	// 写入厂商与模型规格种子数据（INSERT OR IGNORE，幂等安全）
	if err := Seed(db); err != nil {
		return nil, fmt.Errorf("种子数据写入失败: %v", err)
	}

	return db, nil
}
