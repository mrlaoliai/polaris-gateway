// internal/dashboard/api_providers.go
// 作者：mrlaoliai
// system_providers 与 system_models 的完整 CRUD（创建/更新/删除）
package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ─────────────────────────────────────────────────────────────
// /api/v1/providers  POST（创建新厂商）
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Protocol     string `json:"protocol"`
		URLTemplate  string `json:"url_template"`
		AuthType     string `json:"auth_type"`
		AuthConfig   string `json:"auth_config"`
		ConnTimeout  int    `json:"conn_timeout"`
		ReadTimeout  int    `json:"read_timeout"`
		Capabilities string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.ID == "" || body.Name == "" || body.Protocol == "" || body.URLTemplate == "" || body.AuthType == "" {
		writeError(w, http.StatusBadRequest, "id、name、protocol、url_template、auth_type 均为必填")
		return
	}
	if body.ConnTimeout == 0 {
		body.ConnTimeout = 10
	}
	if body.ReadTimeout == 0 {
		body.ReadTimeout = 120
	}
	_, err := h.db.Exec(`
		INSERT INTO system_providers
			(id, name, protocol, url_template, auth_type, auth_config, conn_timeout, read_timeout, capabilities)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		body.ID, body.Name, body.Protocol, body.URLTemplate,
		body.AuthType, body.AuthConfig, body.ConnTimeout, body.ReadTimeout, body.Capabilities,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": body.ID})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/providers/{id}   PUT（更新）& DELETE
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleProvidersWithID(w http.ResponseWriter, r *http.Request) {
	id := extractIDStr(r, "/api/v1/providers/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		// 级联删除该厂商下所有模型，再删厂商本身
		h.dbMgr.AsyncWrite("DELETE FROM system_models WHERE provider_id = ?", id)
		h.dbMgr.AsyncWrite("DELETE FROM system_providers WHERE id = ?", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) updateProvider(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Name         string `json:"name"`
		Protocol     string `json:"protocol"`
		URLTemplate  string `json:"url_template"`
		AuthType     string `json:"auth_type"`
		AuthConfig   string `json:"auth_config"`
		ConnTimeout  int    `json:"conn_timeout"`
		ReadTimeout  int    `json:"read_timeout"`
		Capabilities string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.Name == "" || body.Protocol == "" || body.URLTemplate == "" || body.AuthType == "" {
		writeError(w, http.StatusBadRequest, "name、protocol、url_template、auth_type 均为必填")
		return
	}
	_, err := h.db.Exec(`
		UPDATE system_providers
		SET name=?, protocol=?, url_template=?, auth_type=?, auth_config=?,
		    conn_timeout=?, read_timeout=?, capabilities=?,
		    updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		body.Name, body.Protocol, body.URLTemplate, body.AuthType, body.AuthConfig,
		body.ConnTimeout, body.ReadTimeout, body.Capabilities, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/model-specs  POST（新增模型）
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) createModelSpec(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderID       string `json:"provider_id"`
		ModelID          string `json:"model_id"`
		ModelName        string `json:"model_name"`
		ToolFormat       string `json:"tool_format"`
		MaxContext       int    `json:"max_context"`
		SupportsThinking bool   `json:"supports_thinking"`
		SupportsVision   bool   `json:"supports_vision"`
		SupportsTools    bool   `json:"supports_tools"`
		SupportsJSON     bool   `json:"supports_json"`
		DSLRules         string `json:"dsl_rules"`
		Capabilities     string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.ProviderID == "" || body.ModelID == "" || body.ModelName == "" {
		writeError(w, http.StatusBadRequest, "provider_id、model_id、model_name 为必填")
		return
	}
	t, v, tools, j := btoi(body.SupportsThinking), btoi(body.SupportsVision), btoi(body.SupportsTools), btoi(body.SupportsJSON)
	res, err := h.db.Exec(`
		INSERT INTO system_models
			(provider_id, model_id, model_name, tool_format, max_context,
			 supports_thinking, supports_vision, supports_tools, supports_json, dsl_rules, capabilities)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		body.ProviderID, body.ModelID, body.ModelName, body.ToolFormat, body.MaxContext,
		t, v, tools, j, body.DSLRules, body.Capabilities,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/model-specs/{id}   PUT（更新）& DELETE
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleModelSpecsWithID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/v1/model-specs/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateModelSpec(w, r, id)
	case http.MethodDelete:
		h.dbMgr.AsyncWrite("DELETE FROM system_models WHERE id = ?", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) updateModelSpec(w http.ResponseWriter, r *http.Request, id int) {
	var body struct {
		ModelID          string `json:"model_id"`
		ModelName        string `json:"model_name"`
		ToolFormat       string `json:"tool_format"`
		MaxContext       int    `json:"max_context"`
		SupportsThinking bool   `json:"supports_thinking"`
		SupportsVision   bool   `json:"supports_vision"`
		SupportsTools    bool   `json:"supports_tools"`
		SupportsJSON     bool   `json:"supports_json"`
		DSLRules         string `json:"dsl_rules"`
		Capabilities     string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.ModelID == "" || body.ModelName == "" {
		writeError(w, http.StatusBadRequest, "model_id 和 model_name 为必填")
		return
	}
	t, v, tools, j := btoi(body.SupportsThinking), btoi(body.SupportsVision), btoi(body.SupportsTools), btoi(body.SupportsJSON)
	_, err := h.db.Exec(`
		UPDATE system_models
		SET model_id=?, model_name=?, tool_format=?, max_context=?,
		    supports_thinking=?, supports_vision=?, supports_tools=?, supports_json=?,
		    dsl_rules=?, capabilities=?
		WHERE id=?`,
		body.ModelID, body.ModelName, body.ToolFormat, body.MaxContext,
		t, v, tools, j, body.DSLRules, body.Capabilities, id,
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

// extractIDStr 从如 /api/v1/providers/google-ai-studio 中提取 TEXT ID
func extractIDStr(r *http.Request, prefix string) string {
	raw := strings.TrimPrefix(r.URL.Path, prefix)
	return strings.TrimSuffix(raw, "/")
}

// btoi 将 bool 转换为 SQLite BOOLEAN 整数 (0/1)
func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}
