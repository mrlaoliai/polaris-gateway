package database

import "database/sql"

func migrate(db *sql.DB) error {
	queries := []string{
		// 1. 物理厂商表
		`CREATE TABLE IF NOT EXISTS providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			protocol_type TEXT NOT NULL, -- openai, anthropic, vertex
			base_url TEXT NOT NULL
		);`,

		// 2. 账号/密钥池
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER,
			api_key TEXT NOT NULL,
			priority INTEGER DEFAULT 10,
			status TEXT DEFAULT 'active',
			FOREIGN KEY(provider_id) REFERENCES providers(id)
		);`,

		// 3. 模型技术细节 (包含 DSL 转换逻辑)
		`CREATE TABLE IF NOT EXISTS model_specs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER,
			model_name TEXT NOT NULL,
			tool_format TEXT,
			supports_thinking BOOLEAN DEFAULT 0,
			dsl_rules TEXT, -- 存储 CEL-go 转换表达式 (JSON)
			FOREIGN KEY(provider_id) REFERENCES providers(id)
		);`,

		// 4. 路由规则映射
		`CREATE TABLE IF NOT EXISTS routing_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			in_model TEXT NOT NULL,
			target_spec_id INTEGER,
			fallback_spec_id INTEGER,
			FOREIGN KEY(target_spec_id) REFERENCES model_specs(id)
		);`,

		// 5. L2 溢出缓冲区 (Session Buffer)
		`CREATE TABLE IF NOT EXISTS session_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT NOT NULL,
			chunk_index INTEGER,
			payload BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_session_trace ON session_chunks(trace_id);`,

		// 6. 逻辑凭证与审计
		`CREATE TABLE IF NOT EXISTS gateway_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_value TEXT UNIQUE NOT NULL,
			daily_limit INTEGER,
			used_tokens INTEGER DEFAULT 0
		);`,
		// 在 internal/database/schema.go 的 queries 数组中添加：
		`CREATE TABLE IF NOT EXISTS logic_locks (
			resource_id TEXT PRIMARY KEY,
			session_id TEXT,
			locked_at DATETIME
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
