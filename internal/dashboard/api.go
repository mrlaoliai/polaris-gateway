// internal/dashboard/api.go
// 作者：mrlaoliai
// 设计哲学：State-in-DB — 所有管理操作均通过此层读写 SQLite，前端 /api/v1/* 路由在此注册
package dashboard

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mrlaoliai/polaris-gateway/internal/database"
)

// APIHandler 聚合了全部 REST 管理接口的依赖
type APIHandler struct {
	db    *sql.DB
	dbMgr *database.DBManager
}

// NewAPIHandler 构造函数，注入主库与写入协调器
func NewAPIHandler(db *sql.DB, dbMgr *database.DBManager) *APIHandler {
	return &APIHandler{db: db, dbMgr: dbMgr}
}

// RegisterRoutes 将所有 /api/v1/* 子路由注册到传入的 ServeMux
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	// --- 网关密钥 ---
	mux.HandleFunc("/api/v1/keys", h.handleKeys)
	mux.HandleFunc("/api/v1/keys/", h.handleKeysWithID)

	// --- 物理账号 ---
	mux.HandleFunc("/api/v1/accounts", h.handleAccounts)
	mux.HandleFunc("/api/v1/accounts/", h.handleAccountsWithID)

	// --- 路由规则 ---
	mux.HandleFunc("/api/v1/routing", h.handleRouting)
	mux.HandleFunc("/api/v1/routing/", h.handleRoutingWithID)

	// --- 概览统计 ---
	mux.HandleFunc("/api/v1/stats/overview", h.handleStatsOverview)
	mux.HandleFunc("/api/v1/stats/traces", h.handleStatsTraces)

	// --- Provider 列表（供前端下拉选择）---
	mux.HandleFunc("/api/v1/providers", h.handleProviders)
}

// ─────────────────────────────────────────────────────────────
// 通用工具
// ─────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[API] 序列化响应失败: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// extractID 从形如 /api/v1/keys/42 的路径中提取末尾 ID
func extractID(r *http.Request, prefix string) (int, error) {
	raw := strings.TrimPrefix(r.URL.Path, prefix)
	raw = strings.TrimSuffix(raw, "/")
	return strconv.Atoi(raw)
}

// ─────────────────────────────────────────────────────────────
// /api/v1/keys   (GET=列表, POST=创建)
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listKeys(w, r)
	case http.MethodPost:
		h.createKey(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) listKeys(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.db.Query("SELECT id, key_value, daily_limit, used_tokens FROM gateway_keys ORDER BY id DESC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type KeyRow struct {
		ID         int    `json:"id"`
		KeyValue   string `json:"key_value"`
		DailyLimit int    `json:"daily_limit"`
		UsedTokens int    `json:"used_tokens"`
	}
	var keys []KeyRow
	for rows.Next() {
		var k KeyRow
		if err := rows.Scan(&k.ID, &k.KeyValue, &k.DailyLimit, &k.UsedTokens); err != nil {
			continue
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []KeyRow{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *APIHandler) createKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DailyLimit int `json:"daily_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// 生成随机 Key：前缀 pk- + Unix 纳秒
	keyValue := "pk-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if body.DailyLimit == 0 {
		body.DailyLimit = -1 // 默认无限制
	}

	res, err := h.db.Exec(
		"INSERT INTO gateway_keys (key_value, daily_limit, used_tokens) VALUES (?, ?, 0)",
		keyValue, body.DailyLimit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "key_value": keyValue})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/keys/{id}   (DELETE)
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleKeysWithID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	id, err := extractID(r, "/api/v1/keys/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	h.dbMgr.AsyncWrite("DELETE FROM gateway_keys WHERE id = ?", id)
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────
// /api/v1/accounts   (GET=列表, POST=创建)
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAccounts(w, r)
	case http.MethodPost:
		h.createAccount(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) listAccounts(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.db.Query(`
		SELECT a.id, p.name AS provider_name, a.api_key, a.priority, a.status
		FROM accounts a
		JOIN providers p ON a.provider_id = p.id
		ORDER BY a.priority DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type AccountRow struct {
		ID           int    `json:"id"`
		ProviderName string `json:"provider_name"`
		APIKey       string `json:"api_key"`
		Priority     int    `json:"priority"`
		Status       string `json:"status"`
	}
	var accounts []AccountRow
	for rows.Next() {
		var a AccountRow
		if err := rows.Scan(&a.ID, &a.ProviderName, &a.APIKey, &a.Priority, &a.Status); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []AccountRow{}
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (h *APIHandler) createAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderID int    `json:"provider_id"`
		APIKey     string `json:"api_key"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.APIKey == "" {
		writeError(w, http.StatusBadRequest, "api_key is required")
		return
	}
	if body.Priority == 0 {
		body.Priority = 10
	}

	res, err := h.db.Exec(
		"INSERT INTO accounts (provider_id, api_key, priority, status) VALUES (?, ?, ?, 'active')",
		body.ProviderID, body.APIKey, body.Priority,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/accounts/{id}   (DELETE)
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleAccountsWithID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	id, err := extractID(r, "/api/v1/accounts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	h.dbMgr.AsyncWrite("DELETE FROM accounts WHERE id = ?", id)
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────
// /api/v1/routing   (GET=列表, POST=创建)
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleRouting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listRouting(w, r)
	case http.MethodPost:
		h.createRouting(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) listRouting(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.db.Query(`
		SELECT 
			rr.id,
			rr.in_model,
			tm.model_name  AS target_model,
			COALESCE(fm.model_name, '') AS fallback_model,
			CASE WHEN tm.dsl_rules IS NOT NULL AND tm.dsl_rules != '' THEN 1 ELSE 0 END AS has_dsl
		FROM routing_rules rr
		JOIN model_specs tm ON rr.target_spec_id = tm.id
		LEFT JOIN model_specs fm ON rr.fallback_spec_id = fm.id
		ORDER BY rr.id DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type RuleRow struct {
		ID            int    `json:"id"`
		InModel       string `json:"in_model"`
		TargetModel   string `json:"target_model"`
		FallbackModel string `json:"fallback_model"`
		HasDSL        bool   `json:"has_dsl"`
	}
	var rules []RuleRow
	for rows.Next() {
		var rr RuleRow
		var hasDSLInt int
		if err := rows.Scan(&rr.ID, &rr.InModel, &rr.TargetModel, &rr.FallbackModel, &hasDSLInt); err != nil {
			continue
		}
		rr.HasDSL = hasDSLInt == 1
		rules = append(rules, rr)
	}
	if rules == nil {
		rules = []RuleRow{}
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *APIHandler) createRouting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InModel        string `json:"in_model"`
		TargetSpecID   int    `json:"target_spec_id"`
		FallbackSpecID *int   `json:"fallback_spec_id"` // 可选
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.InModel == "" || body.TargetSpecID == 0 {
		writeError(w, http.StatusBadRequest, "in_model and target_spec_id are required")
		return
	}

	res, err := h.db.Exec(
		"INSERT INTO routing_rules (in_model, target_spec_id, fallback_spec_id) VALUES (?, ?, ?)",
		body.InModel, body.TargetSpecID, body.FallbackSpecID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/routing/{id}   (DELETE)
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleRoutingWithID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	id, err := extractID(r, "/api/v1/routing/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	h.dbMgr.AsyncWrite("DELETE FROM routing_rules WHERE id = ?", id)
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────
// /api/v1/stats/overview
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleStatsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var totalTokens int
	_ = h.db.QueryRow("SELECT COALESCE(SUM(used_tokens), 0) FROM gateway_keys").Scan(&totalTokens)

	var activeAgents int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE status = 'active'").Scan(&activeAgents)

	var totalKeys int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM gateway_keys").Scan(&totalKeys)

	var totalRules int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM routing_rules").Scan(&totalRules)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalTokens":  totalTokens,
		"activeAgents": activeAgents,
		"totalKeys":    totalKeys,
		"totalRules":   totalRules,
		"health":       "Healthy",
	})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/stats/traces  （从 L2 session_chunks 读取）
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleStatsTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
	}

	// 从 session_chunks 表中聚合最近的 traceID 记录
	rows, err := h.db.Query(`
		SELECT trace_id, MAX(chunk_index) AS chunks, MAX(created_at) AS last_seen
		FROM session_chunks
		GROUP BY trace_id
		ORDER BY last_seen DESC
		LIMIT ?
	`, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type TraceRow struct {
		ID       string `json:"id"`
		Chunks   int    `json:"latency"` // 用 chunk 数量近似代表负载
		LastSeen string `json:"last_seen"`
		Status   string `json:"status"`
	}
	var traces []TraceRow
	for rows.Next() {
		var t TraceRow
		if err := rows.Scan(&t.ID, &t.Chunks, &t.LastSeen); err != nil {
			continue
		}
		t.Status = "success"
		traces = append(traces, t)
	}
	if traces == nil {
		traces = []TraceRow{}
	}
	writeJSON(w, http.StatusOK, traces)
}

// ─────────────────────────────────────────────────────────────
// /api/v1/providers  (GET=列表)  供前端下拉菜单联动使用
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	rows, err := h.db.Query("SELECT id, name, protocol_type, base_url FROM providers ORDER BY id ASC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type ProviderRow struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		ProtocolType string `json:"protocol_type"`
		BaseURL      string `json:"base_url"`
	}
	var providers []ProviderRow
	for rows.Next() {
		var p ProviderRow
		if err := rows.Scan(&p.ID, &p.Name, &p.ProtocolType, &p.BaseURL); err != nil {
			continue
		}
		providers = append(providers, p)
	}
	if providers == nil {
		providers = []ProviderRow{}
	}
	writeJSON(w, http.StatusOK, providers)
}
