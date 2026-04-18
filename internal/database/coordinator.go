package database

import (
	"context"
	"database/sql"
	"log"
)

// WriteTask 定义一个通用的写入任务
type WriteTask struct {
	Query string
	Args  []interface{}
}

type DBManager struct {
	PrimaryDB *sql.DB // 核心库
	L2DB      *sql.DB // 大内容/临时库
	writeChan chan WriteTask
}

func NewDBManager(primary, l2 *sql.DB) *DBManager {
	mgr := &DBManager{
		PrimaryDB: primary,
		L2DB:      l2,
		writeChan: make(chan WriteTask, 1000), // 缓冲通道
	}
	return mgr
}

// StartWriterWorker 启动唯一的全局写入协程
func (m *DBManager) StartWriterWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-m.writeChan:
				// 这里统一在核心库执行写入。如果是 L2 的写入，可以单独再开一个 Channel
				_, err := m.PrimaryDB.Exec(task.Query, task.Args...)
				if err != nil {
					log.Printf("[DB-Writer] 写入失败: %v, Query: %s", err, task.Query)
				}
			}
		}
	}()
}

// AsyncWrite 供其他模块调用，实现非阻塞写入
func (m *DBManager) AsyncWrite(query string, args ...interface{}) {
	m.writeChan <- WriteTask{Query: query, Args: args}
}
