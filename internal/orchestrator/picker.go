// internal/orchestrator/picker.go
// 作者：mrlaoliai
// 核心职责：从 provider_keys 池中按"空即全选 + 加权随机"策略挑选最佳凭证
package orchestrator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// ─────────────────────────────────────────────────────────────
// 数据结构
// ─────────────────────────────────────────────────────────────

// PickResult 包含挑选出的凭证信息及已解析的路由参数
type PickResult struct {
	KeyID          int    // provider_keys.id
	Label          string
	CredentialType string // 'api-key' | 'vertex-sa' | 'vertex-adc'

	// API Key 认证
	APIKey string

	// Vertex AI 专属
	ProjectID          string
	Region             string
	ServiceAccountJSON string

	// 解析后的路由参数（三级超时层级已合并）
	Protocol    string // system_providers.protocol
	BaseURL     string // 最终生效的 base_url（用户自定义 > 系统默认）
	ReadTimeout int    // 最终生效的读取超时(秒)
	ConnTimeout int    // 最终生效的连接超时(秒)
	StreamIdle  int    // 流式空闲超时(秒)
	MaxRetries  int    // 最大重试次数
}

// ─────────────────────────────────────────────────────────────
// PickCredential — 主入口
// modelID: system_models.model_id（物理模型 ID）
// ─────────────────────────────────────────────────────────────

// PickCredential 按照以下链条挑选最优凭证：
//
//	Step1: 锁定 user_providers.is_enabled=1 的实例
//	Step2: 筛选 provider_keys.status='active' 且 is_enabled=1
//	Step3: 过滤 selected_models（空=全选）
//	Step4: 加权随机算法选出最终 Key
//	Step5: 合并三级超时（用户配置 > 系统预设）
func PickCredential(db *sql.DB, modelID string) (*PickResult, error) {
	// ── Step1+2：一次 SQL 查询获取所有候选 Key ─────────────────
	rows, err := db.Query(`
		SELECT
			pk.id,
			COALESCE(pk.label, '')               AS label,
			pk.credential_type,
			COALESCE(pk.api_key,              '') AS api_key,
			COALESCE(pk.project_id,           '') AS project_id,
			COALESCE(pk.region,               '') AS region,
			COALESCE(pk.service_account_json, '') AS sa_json,
			COALESCE(pk.selected_models,      '') AS selected_models,
			pk.weight,
			-- 三级超时合并：用户配置 > 系统预设
			COALESCE(NULLIF(up.read_timeout,   0), sp.read_timeout,   120) AS read_timeout,
			COALESCE(NULLIF(up.conn_timeout,   0), sp.conn_timeout,    10) AS conn_timeout,
			COALESCE(NULLIF(up.stream_idle_timeout, 0), 30)               AS stream_idle,
			up.max_retries,
			COALESCE(NULLIF(up.custom_base_url,''), sp.url_template)       AS base_url,
			sp.protocol
		FROM provider_keys pk
		JOIN user_providers   up ON pk.user_provider_id = up.id
		JOIN system_providers sp ON up.system_provider_id = sp.id
		JOIN system_models    sm ON sm.provider_id = sp.id
		WHERE up.is_enabled = 1
		  AND pk.is_enabled = 1
		  AND pk.status     = 'active'
		  AND (pk.cooldown_until IS NULL OR pk.cooldown_until < datetime('now'))
		  AND sm.model_id   = ?
		GROUP BY pk.id
	`, modelID)
	if err != nil {
		return nil, fmt.Errorf("picker query: %w", err)
	}
	defer rows.Close()

	// ── Step3：过滤 selected_models（空=全选） ─────────────────
	type candidate struct {
		PickResult
		weight int
	}
	var pool []candidate

	for rows.Next() {
		var c candidate
		var selectedModelsJSON string
		if err := rows.Scan(
			&c.KeyID, &c.Label, &c.CredentialType,
			&c.APIKey, &c.ProjectID, &c.Region, &c.ServiceAccountJSON,
			&selectedModelsJSON,
			&c.weight,
			&c.ReadTimeout, &c.ConnTimeout, &c.StreamIdle, &c.MaxRetries,
			&c.BaseURL, &c.Protocol,
		); err != nil {
			log.Printf("[Picker] scan error: %v", err)
			continue
		}

		// 空即全选：selectedModelsJSON 为空字符串或 "[]" 直接通过
		if selectedModelsJSON != "" && selectedModelsJSON != "[]" && selectedModelsJSON != "null" {
			var allowed []string
			if err := json.Unmarshal([]byte(selectedModelsJSON), &allowed); err == nil {
				if !containsStr(allowed, modelID) {
					continue // 此 Key 不授权该模型，跳过
				}
			}
		}

		if c.weight <= 0 {
			c.weight = 1
		}
		pool = append(pool, c)
	}

	if len(pool) == 0 {
		return nil, fmt.Errorf("no available credential for model %q", modelID)
	}

	// ── Step4：加权随机算法 ────────────────────────────────────
	chosen := weightedRandomPick(pool, func(c candidate) int { return c.weight })
	result := chosen.PickResult
	return &result, nil
}

// ─────────────────────────────────────────────────────────────
// 故障上报：更新 error_count / status / cooldown
// ─────────────────────────────────────────────────────────────

// ReportKeyError 在请求失败后调用，实现 Sentinel 故障自愈
//
//	httpStatus: 实际 HTTP 返回码（429, 401, 5xx, etc.）
//	threshold:  error_count 超过此值自动 invalid 化（建议 3）
func ReportKeyError(db *sql.DB, keyID int, httpStatus int, threshold int) {
	switch httpStatus {
	case 429:
		// 限流：冷却 60 秒
		db.Exec(`
			UPDATE provider_keys SET
				status         = 'cooldown',
				cooldown_until = datetime('now', '+60 seconds'),
				error_count    = error_count + 1,
				total_errors   = total_errors + 1,
				updated_at     = CURRENT_TIMESTAMP
			WHERE id = ?`, keyID)
		log.Printf("[Sentinel] key#%d → cooldown (429 rate limit)", keyID)

	case 401, 403:
		// 认证失败：直接 invalid
		db.Exec(`
			UPDATE provider_keys SET
				status       = 'invalid',
				is_enabled   = 0,
				error_count  = error_count + 1,
				total_errors = total_errors + 1,
				updated_at   = CURRENT_TIMESTAMP
			WHERE id = ?`, keyID)
		log.Printf("[Sentinel] key#%d → invalid (auth failure %d)", keyID, httpStatus)

	default:
		// 其他错误：累计 error_count，超阈值则自动下线
		db.Exec(`
			UPDATE provider_keys SET
				error_count  = error_count + 1,
				total_errors = total_errors + 1,
				status       = CASE WHEN error_count + 1 >= ? THEN 'invalid' ELSE status END,
				updated_at   = CURRENT_TIMESTAMP
			WHERE id = ?`, threshold, keyID)
	}
}

// ReportKeySuccess 请求成功时调用，重置 error_count 并更新统计
func ReportKeySuccess(db *sql.DB, keyID int) {
	db.Exec(`
		UPDATE provider_keys SET
			error_count    = 0,
			status         = 'active',
			cooldown_until = NULL,
			total_requests = total_requests + 1,
			last_used_at   = CURRENT_TIMESTAMP,
			updated_at     = CURRENT_TIMESTAMP
		WHERE id = ?`, keyID)
}

// ─────────────────────────────────────────────────────────────
// 内部工具
// ─────────────────────────────────────────────────────────────

// weightedRandomPick 实现加权随机：总权重累加后随机落点
func weightedRandomPick[T any](items []T, weightFn func(T) int) T {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	total := 0
	for _, item := range items {
		total += weightFn(item)
	}
	if total == 0 {
		return items[rng.Intn(len(items))]
	}
	r := rng.Intn(total)
	for _, item := range items {
		r -= weightFn(item)
		if r < 0 {
			return item
		}
	}
	return items[len(items)-1]
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
