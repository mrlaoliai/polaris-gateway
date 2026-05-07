CREATE TABLE IF NOT EXISTS sys_settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	listen_addr TEXT DEFAULT '127.0.0.1:28888',
	breaker_initial_cooldown_seconds INTEGER DEFAULT 60,
	breaker_max_cooldown_seconds INTEGER DEFAULT 3600,
	breaker_failure_threshold INTEGER DEFAULT 3,
	breaker_failure_window_seconds INTEGER DEFAULT 120
);

INSERT OR IGNORE INTO sys_settings (id, listen_addr, breaker_initial_cooldown_seconds, breaker_max_cooldown_seconds, breaker_failure_threshold, breaker_failure_window_seconds) 
VALUES (1, '127.0.0.1:28888', 60, 3600, 3, 120);

CREATE TABLE IF NOT EXISTS sys_nodes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	platform TEXT NOT NULL,
	name TEXT NOT NULL UNIQUE,
	key_value TEXT NOT NULL,
	project_id TEXT DEFAULT '',
	location TEXT DEFAULT 'us-central1',
	base_url TEXT DEFAULT '',
	priority INTEGER DEFAULT 0 CHECK (priority >= 0),
	cutoff_percent REAL DEFAULT 95.0,
	budget REAL DEFAULT 0.0,
	billing_start_date TEXT DEFAULT '2000-01-01',
	is_enabled INTEGER DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
