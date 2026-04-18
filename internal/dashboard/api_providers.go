// internal/dashboard/api_providers.go
// 作者：mrlaoliai
// Provider 与 ModelSpec 的完整 CRUD：创建/更新/删除
package dashboard

import (
	"encoding/json"
	"net/http"
)

// ─────────────────────────────────────────────────────────────
// /api/v1/providers   POST（创建新厂商）
// GET 由原 handleProviders 处理；POST 由此扩展
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		ProtocolType string `json:"protocol_type"`
		BaseURL      string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.Name == "" || body.ProtocolType == "" || body.BaseURL == "" {
		writeError(w, http.StatusBadRequest, "name、protocol_type、base_url 均为必填")
		return
	}
	res, err := h.db.Exec(
		"INSERT INTO providers (name, protocol_type, base_url) VALUES (?, ?, ?)",
		body.Name, body.ProtocolType, body.BaseURL,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/providers/{id}   PUT（更新）& DELETE
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) handleProvidersWithID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/v1/providers/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		// 级联删除该厂商下所有模型规格，再删厂商本身
		h.dbMgr.AsyncWrite("DELETE FROM model_specs WHERE provider_id = ?", id)
		h.dbMgr.AsyncWrite("DELETE FROM providers WHERE id = ?", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) updateProvider(w http.ResponseWriter, r *http.Request, id int) {
	var body struct {
		Name         string `json:"name"`
		ProtocolType string `json:"protocol_type"`
		BaseURL      string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.Name == "" || body.ProtocolType == "" || body.BaseURL == "" {
		writeError(w, http.StatusBadRequest, "name、protocol_type、base_url 均为必填")
		return
	}
	_, err := h.db.Exec(
		"UPDATE providers SET name=?, protocol_type=?, base_url=? WHERE id=?",
		body.Name, body.ProtocolType, body.BaseURL, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─────────────────────────────────────────────────────────────
// /api/v1/model-specs   POST（新增模型规格）
// GET 由原 handleModelSpecs 处理；POST 由此扩展
// ─────────────────────────────────────────────────────────────

func (h *APIHandler) createModelSpec(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderID       int    `json:"provider_id"`
		ModelName        string `json:"model_name"`
		ToolFormat       string `json:"tool_format"`
		SupportsThinking bool   `json:"supports_thinking"`
		SupportsVision   bool   `json:"supports_vision"`
		DSLRules         string `json:"dsl_rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.ProviderID == 0 || body.ModelName == "" {
		writeError(w, http.StatusBadRequest, "provider_id 和 model_name 为必填")
		return
	}
	thinking, vision := 0, 0
	if body.SupportsThinking {
		thinking = 1
	}
	if body.SupportsVision {
		vision = 1
	}
	res, err := h.db.Exec(
		`INSERT INTO model_specs (provider_id, model_name, tool_format, supports_thinking, supports_vision, dsl_rules)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		body.ProviderID, body.ModelName, body.ToolFormat, thinking, vision, body.DSLRules,
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
		h.dbMgr.AsyncWrite("DELETE FROM model_specs WHERE id = ?", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (h *APIHandler) updateModelSpec(w http.ResponseWriter, r *http.Request, id int) {
	var body struct {
		ModelName        string `json:"model_name"`
		ToolFormat       string `json:"tool_format"`
		SupportsThinking bool   `json:"supports_thinking"`
		SupportsVision   bool   `json:"supports_vision"`
		DSLRules         string `json:"dsl_rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if body.ModelName == "" {
		writeError(w, http.StatusBadRequest, "model_name 为必填")
		return
	}
	thinking, vision := 0, 0
	if body.SupportsThinking {
		thinking = 1
	}
	if body.SupportsVision {
		vision = 1
	}
	_, err := h.db.Exec(
		`UPDATE model_specs SET model_name=?, tool_format=?, supports_thinking=?, supports_vision=?, dsl_rules=? WHERE id=?`,
		body.ModelName, body.ToolFormat, thinking, vision, body.DSLRules, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
