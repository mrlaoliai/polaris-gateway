// 内部使用：internal/database/coordinator.go
// 作者：mrlaoliai
// 设计哲学：通过异步 Channel 序列化所有写操作，确保 State-in-DB 在高并发下的绝对稳定性
package database

import (
	"context"
	"database/sql"
	"log"
)

// WriteTask 封装了一个待执行的数据库写入任务
type WriteTask struct {
	DB    *sql.DB       // 目标数据库实例 (Primary 或 L2)
	Query string        // SQL 语句
	Args  []interface{} // 参数化查询参数
}

// DBManager 负责协调多个 SQLite 实例的并发访问与异步写入
type DBManager struct {
	PrimaryDB *sql.DB        // 核心库：存放路由、密钥、配额
	L2DB      *sql.DB        // L2 索引库：存放 VFS 片段路径
	writeChan chan WriteTask // 全局写入任务通道
}

// NewDBManager 初始化数据库协调器
func NewDBManager(primary, l2 *sql.DB) *DBManager {
	return &DBManager{
		PrimaryDB: primary,
		L2DB:      l2,
		writeChan: make(chan WriteTask, 2048), // 设定较大的缓冲区以应对突发流量
	}
}

// StartWriterWorker 启动全局唯一的后台写入协程
// 它按顺序从通道读取任务并执行，确保 SQLite 永远不会遇到并发写冲突
func (m *DBManager) StartWriterWorker(ctx context.Context) {
	go func() {
		log.Println("🛠️ DB-Writer 协程已启动，接管全局异步写入任务")
		for {
			select {
			case <-ctx.Done():
				log.Println("⚠️ DB-Writer 协程接收到关闭指令，正在退出...")
				return
			case task := <-m.writeChan:
				// 执行物理写入
				_, err := task.DB.Exec(task.Query, task.Args...)
				if err != nil {
					log.Printf("[DB-Writer] 任务执行失败: %v | SQL: %s | Args: %v", err, task.Query, task.Args)
				}
			}
		}
	}()
}

// AsyncWrite 异步写入核心库 (Primary DB)
// 适用于：扣除配额、记录审计日志、更新账号状态等
func (m *DBManager) AsyncWrite(query string, args ...interface{}) {
	m.writeChan <- WriteTask{
		DB:    m.PrimaryDB,
		Query: query,
		Args:  args,
	}
}

// AsyncWriteL2 异步写入 L2 索引库 (L2 DB)
// 适用于：记录 VFS 物理文件路径索引
func (m *DBManager) AsyncWriteL2(query string, args ...interface{}) {
	m.writeChan <- WriteTask{
		DB:    m.L2DB,
		Query: query,
		Args:  args,
	}
}

// GetPrimary 返回主库句柄，用于各模块直接执行“读”操作 (Query)
func (m *DBManager) GetPrimary() *sql.DB {
	return m.PrimaryDB
}

// GetL2 返回 L2 库句柄，用于“读”操作
func (m *DBManager) GetL2() *sql.DB {
	return m.L2DB
}
