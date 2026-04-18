// internal/dashboard/api_user_providers.go
// 作者：mrlaoliai
// user_providers 与 provider_keys 的完整 CRUD 接口
package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ═══════════════════════════════════════════════════════════════
// /api/v1/user-providers
// ═══════════════════════════════════════════════════════════════

func (h *APIHandler) handleUserProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listUserProviders(w)
	case http.MethodPost:
		h.createUserProvider(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

// GET /api/v1/user-providers — 列出所有已配置的用户厂商实例
func (h *APIHandler) listUserProviders(w http.ResponseWriter) {
	rows, err := h.db.Query(`
		SELECT
			up.id,
			up.system_provider_id,
			COALESCE(up.name, sp.name)    AS name,
			COALESCE(up.custom_base_url, '') AS custom_base_url,
			COALESCE(up.conn_timeout,   0)  AS conn_timeout,
			COALESCE(up.read_timeout,   0)  AS read_timeout,
			COALESCE(up.stream_idle_timeout, 30) AS stream_idle_timeout,
			COALESCE(up.max_retries,    3)  AS max_retries,
			up.is_enabled,
			sp.name      AS system_name,
			sp.protocol,
			sp.url_template,
			sp.auth_type,
			COUNT(pk.id) AS key_count
		FROM user_providers up
		JOIN system_providers sp ON up.system_provider_id = sp.id
		LEFT JOIN provider_keys pk ON pk.user_provider_id = up.id
		GROUP BY up.id
		ORDER BY up.created_at ASC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type Row struct {
		ID                 int    `json:"id"`
		SystemProviderID   string `json:"system_provider_id"`
		Name               string `json:"name"`
		CustomBaseURL      string `json:"custom_base_url"`
		ConnTimeout        int    `json:"conn_timeout"`
		ReadTimeout        int    `json:"read_timeout"`
		StreamIdleTimeout  int    `json:"stream_idle_timeout"`
		MaxRetries         int    `json:"max_retries"`
		IsEnabled          bool   `json:"is_enabled"`
		SystemName         string `json:"system_name"`
		Protocol           string `json:"protocol"`
		URLTemplate        string `json:"url_template"`
		AuthType           string `json:"auth_type"`
		KeyCount           int    `json:"key_count"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		var isEnabledInt int
		if err := rows.Scan(
			&r.ID, &r.SystemProviderID, &r.Name, &r.CustomBaseURL,
			&r.ConnTimeout, &r.ReadTimeout, &r.StreamIdleTimeout, &r.MaxRetries,
			&isEnabledInt,
			&r.SystemName, &r.Protocol, &r.URLTemplate, &r.AuthType,
			&r.KeyCount,
		); err != nil {
			continue
		}
		r.IsEnabled = isEnabledInt == 1
		list = append(list, r)
	}
	if list == nil {
		list = []Row{}
	}
	writeJSON(w, http.StatusOK, list)
}

// POST /api/v1/user-providers — 新增用户厂商配置
func (h *APIHandler) createUserProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SystemProviderID   string `json:"system_provider_id"`
		Name               string `json:"name"`
		CustomBaseURL      string `json:"custom_base_url"`
		ConnTimeout        int    `json:"conn_timeout"`
		ReadTimeout        int    `json:"read_timeout"`
		StreamIdleTimeout  int    `json:"stream_idle_timeout"`
		MaxRetries         int    `json:"max_retries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if body.SystemProviderID == "" {
		writeError(w, http.StatusBadRequest, "system_provider_id 为必填")
		return
	}
	if body.StreamIdleTimeout == 0 {
		body.StreamIdleTimeout = 30
	}
	if body.MaxRetries == 0 {
		body.MaxRetries = 3
	}
	res, err := h.db.Exec(`
		INSERT INTO user_providers
			(system_provider_id, name, custom_base_url, conn_timeout, read_timeout, stream_idle_timeout, max_retries)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		body.SystemProviderID, body.Name, body.CustomBaseURL,
		body.ConnTimeout, body.ReadTimeout, body.StreamIdleTimeout, body.MaxRetries,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "该厂商已经存在配置，请勿重复添加")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/user-providers/{id}   PUT / DELETE
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleUserProvidersWithID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/v1/user-providers/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateUserProvider(w, r, id)
	case http.MethodDelete:
		// 级联删除该 Provider 下所有 Key
		h.dbMgr.AsyncWrite("DELETE FROM provider_keys WHERE user_provider_id = ?", id)
		h.dbMgr.AsyncWrite("DELETE FROM user_providers WHERE id = ?", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) updateUserProvider(w http.ResponseWriter, r *http.Request, id int) {
	var body struct {
		Name              string `json:"name"`
		CustomBaseURL     string `json:"custom_base_url"`
		ConnTimeout       int    `json:"conn_timeout"`
		ReadTimeout       int    `json:"read_timeout"`
		StreamIdleTimeout int    `json:"stream_idle_timeout"`
		MaxRetries        int    `json:"max_retries"`
		IsEnabled         bool   `json:"is_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	_, err := h.db.Exec(`
		UPDATE user_providers SET
			name = ?, custom_base_url = ?, conn_timeout = ?, read_timeout = ?,
			stream_idle_timeout = ?, max_retries = ?, is_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		body.Name, body.CustomBaseURL, body.ConnTimeout, body.ReadTimeout,
		body.StreamIdleTimeout, body.MaxRetries, btoi(body.IsEnabled), id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/user-providers/available
// 返回尚未被用户配置过的 system_providers 列表（用于下拉选单）
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleAvailableProviders(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.db.Query(`
		SELECT sp.id, sp.name, sp.protocol, sp.auth_type, sp.url_template
		FROM system_providers sp
		WHERE sp.id NOT IN (SELECT system_provider_id FROM user_providers)
		ORDER BY sp.name ASC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type Row struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Protocol    string `json:"protocol"`
		AuthType    string `json:"auth_type"`
		URLTemplate string `json:"url_template"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Name, &r.Protocol, &r.AuthType, &r.URLTemplate); err != nil {
			continue
		}
		list = append(list, r)
	}
	if list == nil {
		list = []Row{}
	}
	writeJSON(w, http.StatusOK, list)
}

// ═══════════════════════════════════════════════════════════════
// /api/v1/provider-keys
// ═══════════════════════════════════════════════════════════════

func (h *APIHandler) handleProviderKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProviderKeys(w, r)
	case http.MethodPost:
		h.createProviderKey(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

// GET /api/v1/provider-keys?user_provider_id=N
func (h *APIHandler) listProviderKeys(w http.ResponseWriter, r *http.Request) {
	upid := r.URL.Query().Get("user_provider_id")
	if upid == "" {
		writeError(w, http.StatusBadRequest, "user_provider_id 为必填参数")
		return
	}
	rows, err := h.db.Query(`
		SELECT
			pk.id,
			COALESCE(pk.label, '')               AS label,
			pk.credential_type,
			COALESCE(pk.api_key, '')             AS api_key,
			COALESCE(pk.project_id, '')          AS project_id,
			COALESCE(pk.region, '')              AS region,
			COALESCE(pk.selected_models, '')     AS selected_models,
			pk.weight,
			pk.is_enabled,
			pk.status,
			pk.error_count,
			pk.total_requests,
			pk.total_errors,
			COALESCE(pk.last_used_at, '')        AS last_used_at
		FROM provider_keys pk
		WHERE pk.user_provider_id = ?
		ORDER BY pk.weight DESC, pk.id ASC
	`, upid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type KeyRow struct {
		ID             int    `json:"id"`
		Label          string `json:"label"`
		CredentialType string `json:"credential_type"`
		APIKeyMasked   string `json:"api_key_masked"` // 仅返回脱敏值
		ProjectID      string `json:"project_id"`
		Region         string `json:"region"`
		SelectedModels string `json:"selected_models"`
		Weight         int    `json:"weight"`
		IsEnabled      bool   `json:"is_enabled"`
		Status         string `json:"status"`
		ErrorCount     int    `json:"error_count"`
		TotalRequests  int    `json:"total_requests"`
		TotalErrors    int    `json:"total_errors"`
		LastUsedAt     string `json:"last_used_at"`
	}
	var list []KeyRow
	for rows.Next() {
		var k KeyRow
		var rawKey string
		var isEnabledInt int
		if err := rows.Scan(
			&k.ID, &k.Label, &k.CredentialType, &rawKey,
			&k.ProjectID, &k.Region, &k.SelectedModels,
			&k.Weight, &isEnabledInt, &k.Status,
			&k.ErrorCount, &k.TotalRequests, &k.TotalErrors, &k.LastUsedAt,
		); err != nil {
			continue
		}
		k.IsEnabled = isEnabledInt == 1
		k.APIKeyMasked = maskAPIKey(rawKey)
		list = append(list, k)
	}
	if list == nil {
		list = []KeyRow{}
	}
	writeJSON(w, http.StatusOK, list)
}

// POST /api/v1/provider-keys — 添加新密钥
func (h *APIHandler) createProviderKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserProviderID     int    `json:"user_provider_id"`
		Label              string `json:"label"`
		CredentialType     string `json:"credential_type"`
		APIKey             string `json:"api_key"`
		ProjectID          string `json:"project_id"`
		Region             string `json:"region"`
		ServiceAccountJSON string `json:"service_account_json"`
		SelectedModels     string `json:"selected_models"` // JSON 数组字符串
		Weight             int    `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if body.UserProviderID == 0 {
		writeError(w, http.StatusBadRequest, "user_provider_id 为必填")
		return
	}
	if body.CredentialType == "" {
		body.CredentialType = "api-key"
	}
	if body.Weight <= 0 {
		body.Weight = 10
	}
	var selectedModels interface{} = nil
	if body.SelectedModels != "" && body.SelectedModels != "[]" {
		selectedModels = body.SelectedModels
	}
	res, err := h.db.Exec(`
		INSERT INTO provider_keys
			(user_provider_id, label, credential_type, api_key,
			 project_id, region, service_account_json, selected_models, weight)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		body.UserProviderID, body.Label, body.CredentialType, body.APIKey,
		body.ProjectID, body.Region, body.ServiceAccountJSON, selectedModels, body.Weight,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/provider-keys/{id}   PUT / DELETE / PATCH
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleProviderKeysWithID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// PATCH /api/v1/provider-keys/{id}/toggle — 切换 is_enabled
	if strings.HasSuffix(path, "/toggle") {
		rawPath := strings.TrimSuffix(path, "/toggle")
		id, err := extractID(r, "/api/v1/provider-keys/")
		_ = rawPath
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid ID")
			return
		}
		var body struct {
			IsEnabled bool `json:"is_enabled"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		h.dbMgr.AsyncWrite(`UPDATE provider_keys SET is_enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			btoi(body.IsEnabled), id)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	id, err := extractID(r, "/api/v1/provider-keys/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateProviderKey(w, r, id)
	case http.MethodDelete:
		h.dbMgr.AsyncWrite("DELETE FROM provider_keys WHERE id = ?", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) updateProviderKey(w http.ResponseWriter, r *http.Request, id int) {
	var body struct {
		Label              string `json:"label"`
		CredentialType     string `json:"credential_type"`
		APIKey             string `json:"api_key"`
		ProjectID          string `json:"project_id"`
		Region             string `json:"region"`
		ServiceAccountJSON string `json:"service_account_json"`
		SelectedModels     string `json:"selected_models"`
		Weight             int    `json:"weight"`
		IsEnabled          bool   `json:"is_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if body.Weight <= 0 {
		body.Weight = 10
	}
	var selectedModels interface{} = nil
	if body.SelectedModels != "" && body.SelectedModels != "[]" {
		selectedModels = body.SelectedModels
	}
	_, err := h.db.Exec(`
		UPDATE provider_keys SET
			label=?, credential_type=?, api_key=?,
			project_id=?, region=?, service_account_json=?,
			selected_models=?, weight=?, is_enabled=?,
			status = CASE WHEN is_enabled=0 AND ? = 1 THEN 'active' ELSE status END,
			error_count = CASE WHEN is_enabled=0 AND ? = 1 THEN 0 ELSE error_count END,
			updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		body.Label, body.CredentialType, body.APIKey,
		body.ProjectID, body.Region, body.ServiceAccountJSON,
		selectedModels, body.Weight, btoi(body.IsEnabled),
		btoi(body.IsEnabled), btoi(body.IsEnabled),
		id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─────────────────────────────────────────────────────────────
// 工具函数
// ─────────────────────────────────────────────────────────────

// maskAPIKey 对 API Key 进行脱敏展示，只保留前后 4 位
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
