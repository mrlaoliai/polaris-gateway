// 内部使用：internal/database/schema.go
// 作者：mrlaoliai
package database

import "database/sql"

func migrate(db *sql.DB) error {
	queries := []string{
		// 1. 物理厂商表：定义 API 的根地址与协议
		`CREATE TABLE IF NOT EXISTS providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			protocol_type TEXT NOT NULL, -- openai, anthropic, vertex, google
			base_url TEXT NOT NULL
		);`,

		// 2. 账号/密钥池：实现物理层密钥管理
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER,
			api_key TEXT NOT NULL,
			priority INTEGER DEFAULT 10,
			status TEXT DEFAULT 'active',
			FOREIGN KEY(provider_id) REFERENCES providers(id)
		);`,

		// 3. 模型技术规格：包含核心的 DSL 规则
		`CREATE TABLE IF NOT EXISTS model_specs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER,
			model_name TEXT NOT NULL,
			tool_format TEXT,
			supports_thinking BOOLEAN DEFAULT 0,
			supports_vision BOOLEAN DEFAULT 0, -- [补全] 配合 Transcoder 执行多模态降级
			dsl_rules TEXT, -- 存储动态改写 Payload 的 CEL 表达式
			FOREIGN KEY(provider_id) REFERENCES providers(id)
		);`,

		// 4. 虚拟模型路由映射
		`CREATE TABLE IF NOT EXISTS routing_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			in_model TEXT NOT NULL,
			target_spec_id INTEGER,
			fallback_spec_id INTEGER,
			FOREIGN KEY(target_spec_id) REFERENCES model_specs(id)
		);`,

		// 5. L2 对话溢出存储 (State-in-DB 的核心)
		`CREATE TABLE IF NOT EXISTS session_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT NOT NULL,
			chunk_index INTEGER,
			file_path TEXT NOT NULL, -- 存储 VFS 中的物理路径
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_session_trace ON session_chunks(trace_id);`,

		// 6. 鉴权密钥与配额管理
		`CREATE TABLE IF NOT EXISTS gateway_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_value TEXT UNIQUE NOT NULL,
			daily_limit INTEGER,
			used_tokens INTEGER DEFAULT 0
		);`,

		// 7. 逻辑并发锁 (自治协作控制)
		`CREATE TABLE IF NOT EXISTS logic_locks (
			resource_id TEXT PRIMARY KEY,
			session_id TEXT,
			locked_at DATETIME,
			expires_at DATETIME -- [补全] 确保僵尸锁能被自动回收
		);`,

		// 8. [补全] 请求审计追踪表 (对应 README 中的 usage_stats)
		`CREATE TABLE IF NOT EXISTS usage_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT NOT NULL,
			gateway_key_id INTEGER,
			in_model TEXT,
			target_model TEXT,
			protocol TEXT,
			latency_ms INTEGER DEFAULT 0,
			tokens_used INTEGER DEFAULT 0,
			status TEXT DEFAULT 'success',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(gateway_key_id) REFERENCES gateway_keys(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_trace ON usage_stats(trace_id);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_created ON usage_stats(created_at);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
