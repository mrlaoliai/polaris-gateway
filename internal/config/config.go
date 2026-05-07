package config

import (
	"fmt"
	"log/slog"
	"sort"

	"polaris-gateway/internal/db"
)

type AccountDetail struct {
	Name             string  `json:"name"`
	Enabled          bool    `json:"enabled"`
	Key              string  `json:"-"`
	BaseURL          string  `json:"base_url"`
	ProjectID        string  `json:"project_id"`
	Location         string  `json:"location"`
	Priority         int     `json:"priority"`
	Budget           float64 `json:"budget"`
	CutoffPercent    float64 `json:"cutoff_percent"`
	BillingStartDate string  `json:"billing_start_date"`
}

type Config struct {
	ListenAddr string `json:"listen_addr"`
	DebugMode  bool   `json:"debug_mode"`
	Breaker    struct {
		InitialCooldownSeconds int `json:"initial_cooldown_seconds"`
		MaxCooldownSeconds     int `json:"max_cooldown_seconds"`
		FailureThreshold       int `json:"failure_threshold"`
		FailureWindowSeconds   int `json:"failure_window_seconds"`
	} `json:"breaker"`
	Providers map[string][]AccountDetail `json:"providers"`
}

var AppConfig Config

func LoadConfig(yamlFile string, envFile string) error {
	// 迁移逻辑已移除。由于用户希望通过前端自行管理节点，
	// 此处直接从数据库加载配置。如果数据库为空，则初始启动时无节点。
	return ReloadFromDB()
}

func ReloadFromDB() error {
	AppConfig = Config{
		Providers: make(map[string][]AccountDetail),
	}

	// Load Settings
	err := db.DB().QueryRow("SELECT listen_addr, breaker_initial_cooldown_seconds, breaker_max_cooldown_seconds, breaker_failure_threshold, breaker_failure_window_seconds FROM sys_settings WHERE id = 1").Scan(
		&AppConfig.ListenAddr,
		&AppConfig.Breaker.InitialCooldownSeconds,
		&AppConfig.Breaker.MaxCooldownSeconds,
		&AppConfig.Breaker.FailureThreshold,
		&AppConfig.Breaker.FailureWindowSeconds,
	)
	if err != nil {
		slog.Error("读取系统配置失败，将使用默认值", "error", err)
		AppConfig.ListenAddr = "127.0.0.1:28888"
		AppConfig.Breaker.InitialCooldownSeconds = 60
		AppConfig.Breaker.MaxCooldownSeconds = 3600
		AppConfig.Breaker.FailureThreshold = 3
		AppConfig.Breaker.FailureWindowSeconds = 120
	}

	if AppConfig.ListenAddr == "" {
		AppConfig.ListenAddr = "127.0.0.1:28888"
	}

	// Load Nodes
	rows, err := db.DB().Query("SELECT platform, name, key_value, project_id, location, base_url, priority, cutoff_percent, budget, billing_start_date, is_enabled FROM sys_nodes")
	if err != nil {
		return fmt.Errorf("读取节点列表失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var platform string
		var acc AccountDetail
		var isEnabled int
		if err := rows.Scan(&platform, &acc.Name, &acc.Key, &acc.ProjectID, &acc.Location, &acc.BaseURL, &acc.Priority, &acc.CutoffPercent, &acc.Budget, &acc.BillingStartDate, &isEnabled); err != nil {
			slog.Error("扫描节点数据失败", "error", err)
			continue
		}
		acc.Enabled = (isEnabled == 1)

		if !acc.Enabled {
			continue // Skip disabled nodes in memory
		}
		AppConfig.Providers[platform] = append(AppConfig.Providers[platform], acc)
	}

	// Sort & Check
	for platform, accounts := range AppConfig.Providers {
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].Priority < accounts[j].Priority
		})
		AppConfig.Providers[platform] = accounts
		if len(accounts) > 0 {
			slog.Info("🚦 平台装载完成", "platform", platform, "active_nodes", len(accounts))
		}
	}

	if len(AppConfig.Providers["vertex"]) == 0 && len(AppConfig.Providers["openai"]) == 0 {
		slog.Error("无可用物理节点，网关将返回 503 直到添加新节点")
	}

	return nil
}
