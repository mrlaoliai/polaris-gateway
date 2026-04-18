// 内部使用：internal/database/schema.go
// 作者：mrlaoliai
// 设计哲学：State-in-DB — 所有核心状态均持久化于 SQLite，应用层无状态
package database

import "database/sql"

func migrate(db *sql.DB) error {
	queries := []string{

		// ══════════════════════════════════════════════════════════
		// 1. 系统厂商模板表 (system_providers)
		//    id 使用人类可读的 TEXT 标识符（如 google-ai-studio）
		//    区别于原 providers 表：本表为"系统预设配置层"
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS system_providers (
			id           TEXT PRIMARY KEY,          -- 人类可读标识 (如 google-ai-studio, anthropic)
			name         TEXT NOT NULL,             -- 显示名称
			protocol     TEXT NOT NULL,             -- 核心协议: google-ai | vertex | openai | anthropic
			url_template TEXT NOT NULL,             -- URL 模板，支持 {model_id} 和 {region} 占位符
			auth_type    TEXT NOT NULL,             -- 认证大类: api-key | oauth2
			auth_config  TEXT,                      -- JSON: Key 的位置和前缀配置
			conn_timeout INTEGER DEFAULT 10,        -- 连接超时(秒)
			read_timeout INTEGER DEFAULT 120,       -- 读取超时(秒)
			capabilities TEXT,                      -- JSON: 厂商扩展属性（如支持的 Region 列表）
			updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// ══════════════════════════════════════════════════════════
		// 2. 系统模型规格表 (system_models)
		//    2026 深度能力对齐：比原 model_specs 增加 max_context、
		//    supports_tools、supports_json、model_id(物理ID)、model_name(展示名) 等字段
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS system_models (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id       TEXT    NOT NULL,      -- 关联 system_providers.id
			model_id          TEXT    NOT NULL,      -- 物理 ID (如 gpt-5.4-turbo)
			model_name        TEXT    NOT NULL,      -- 展示名 (如 GPT-5.4 Omni)
			tool_format       TEXT,                  -- 工具调用格式: openai | google | anthropic
			max_context       INTEGER,               -- 最大上下文长度 (token 数)
			supports_thinking BOOLEAN DEFAULT 0,     -- 是否支持专属推理模式/块
			supports_vision   BOOLEAN DEFAULT 0,     -- 是否支持视觉输入
			supports_tools    BOOLEAN DEFAULT 0,     -- 是否支持 Function Call
			supports_json     BOOLEAN DEFAULT 0,     -- 是否支持 JSON Mode
			dsl_rules         TEXT,                  -- 默认内置的转换/清洗 DSL 逻辑
			capabilities      TEXT,                  -- JSON: 其它扩展参数（训练截止日期等）
			FOREIGN KEY (provider_id) REFERENCES system_providers(id)
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sysmod_provider_model ON system_models(provider_id, model_id);`,

		// ══════════════════════════════════════════════════════════
		// 3. 账号/密钥池：provider_id 改为 TEXT，关联 system_providers
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS accounts (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id TEXT,                        -- 关联 system_providers.id (TEXT)
			api_key     TEXT NOT NULL,
			priority    INTEGER DEFAULT 10,
			status      TEXT DEFAULT 'active',
			label       TEXT,                        -- 可选备注标签
			FOREIGN KEY(provider_id) REFERENCES system_providers(id)
		);`,

		// ══════════════════════════════════════════════════════════
		// 4. 虚拟模型路由映射：target/fallback 关联 system_models.id
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS routing_rules (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			in_model         TEXT NOT NULL,          -- 客户端请求时使用的虚拟模型名
			target_spec_id   INTEGER,                -- 关联 system_models.id
			fallback_spec_id INTEGER,                -- 关联 system_models.id (可空)
			FOREIGN KEY(target_spec_id) REFERENCES system_models(id)
		);`,

		// ══════════════════════════════════════════════════════════
		// 5. L2 对话溢出存储 (State-in-DB 核心)
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS session_chunks (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id    TEXT NOT NULL,
			chunk_index INTEGER,
			file_path   TEXT NOT NULL,               -- VFS 物理路径
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_session_trace ON session_chunks(trace_id);`,

		// ══════════════════════════════════════════════════════════
		// 6. 鉴权密钥与配额管理
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS gateway_keys (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			key_value   TEXT UNIQUE NOT NULL,
			daily_limit INTEGER,
			used_tokens INTEGER DEFAULT 0
		);`,

		// ══════════════════════════════════════════════════════════
		// 7. 逻辑并发锁 (自治协作控制)
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS logic_locks (
			resource_id TEXT PRIMARY KEY,
			session_id  TEXT,
			locked_at   DATETIME,
			expires_at  DATETIME                     -- 确保僵尸锁能被自动回收
		);`,

		// ══════════════════════════════════════════════════════════
		// 8. 请求审计追踪表
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS usage_stats (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id       TEXT NOT NULL,
			gateway_key_id INTEGER,
			in_model       TEXT,
			target_model   TEXT,
			protocol       TEXT,
			latency_ms     INTEGER DEFAULT 0,
			tokens_used    INTEGER DEFAULT 0,
			status         TEXT DEFAULT 'success',
			created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(gateway_key_id) REFERENCES gateway_keys(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_trace   ON usage_stats(trace_id);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_created ON usage_stats(created_at);`,

		// ══════════════════════════════════════════════════════════
		// 9. 用户厂商配置表 (user_providers)
		//    一个 system_provider 只能被实例化一次 (UNIQUE 约束)
		//    超时字段 0 = 继承 system_providers 的系统默认值
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS user_providers (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			system_provider_id  TEXT    NOT NULL UNIQUE,
			name                TEXT,
			custom_base_url     TEXT,
			conn_timeout        INTEGER DEFAULT 0,
			read_timeout        INTEGER DEFAULT 0,
			stream_idle_timeout INTEGER DEFAULT 30,
			max_retries         INTEGER DEFAULT 3,
			is_enabled          BOOLEAN DEFAULT 1,
			created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (system_provider_id) REFERENCES system_providers(id)
		);`,

		// ══════════════════════════════════════════════════════════
		// 10. 提供商密钥/凭证表 (provider_keys)
		//     同时支持 API Key 和 Vertex SA/OAuth2 两种认证模式
		//     selected_models: JSON 数组；NULL 或空 = 授权该厂商全部模型
		//     is_enabled: 用户手动开关
		//     status:     系统自动维护 (active|cooldown|invalid)
		// ══════════════════════════════════════════════════════════
		`CREATE TABLE IF NOT EXISTS provider_keys (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			user_provider_id     INTEGER NOT NULL,
			label                TEXT,
			credential_type      TEXT    NOT NULL DEFAULT 'api-key',
			api_key              TEXT,
			project_id           TEXT,
			region               TEXT,
			service_account_json TEXT,
			selected_models      TEXT    DEFAULT NULL,
			weight               INTEGER DEFAULT 10,
			is_enabled           BOOLEAN DEFAULT 1,
			status               TEXT    DEFAULT 'active',
			error_count          INTEGER DEFAULT 0,
			cooldown_until       DATETIME,
			total_requests       INTEGER DEFAULT 0,
			total_errors         INTEGER DEFAULT 0,
			last_used_at         DATETIME,
			created_at           DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at           DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_provider_id) REFERENCES user_providers(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pk_user_provider ON provider_keys(user_provider_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pk_status        ON provider_keys(status, is_enabled);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
