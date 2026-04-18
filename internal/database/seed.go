// internal/database/seed.go
// 作者：mrlaoliai
// 设计哲学：State-in-DB — 系统启动时自动写入所有已知厂商与模型规格种子数据
// 使用 INSERT OR IGNORE 保证幂等性，不会在重启时重复插入
package database

import "database/sql"

// Seed 写入所有预置的 Provider 与 ModelSpec 种子数据
// 必须在 migrate() 之后调用，以确保表已存在
func Seed(db *sql.DB) error {
	// ──────────────────────────────────────────────
	// 1. 厂商 (providers)
	// ──────────────────────────────────────────────
	providers := []struct {
		name         string
		protocolType string
		baseURL      string
	}{
		// Anthropic — Claude 系列
		{
			"Anthropic",
			"anthropic",
			"https://api.anthropic.com/v1/messages",
		},
		// Google AI Studio — Gemini API (API Key 认证)
		{
			"Google AI Studio",
			"google",
			"https://generativelanguage.googleapis.com/v1beta/models",
		},
		// Google Vertex AI — Gemini on GCP (OAuth2 / Service Account)
		{
			"Google Vertex AI",
			"vertex",
			"https://aiplatform.googleapis.com/v1",
		},
		// DeepSeek — OpenAI 兼容协议
		{
			"DeepSeek",
			"openai",
			"https://api.deepseek.com/v1/chat/completions",
		},
		// OpenAI — 官方接口
		{
			"OpenAI",
			"openai",
			"https://api.openai.com/v1/chat/completions",
		},
	}

	for _, p := range providers {
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO providers (name, protocol_type, base_url) VALUES (?, ?, ?)`,
			p.name, p.protocolType, p.baseURL,
		); err != nil {
			return err
		}
	}

	// ──────────────────────────────────────────────
	// 2. 读取各厂商 ID（用于关联 model_specs）
	// ──────────────────────────────────────────────
	providerID := func(name string) int {
		var id int
		_ = db.QueryRow(`SELECT id FROM providers WHERE name = ?`, name).Scan(&id)
		return id
	}

	anthropicID := providerID("Anthropic")
	googleStudioID := providerID("Google AI Studio")
	vertexID := providerID("Google Vertex AI")
	deepseekID := providerID("DeepSeek")
	openaiID := providerID("OpenAI")

	// ──────────────────────────────────────────────
	// 3. 模型规格 (model_specs)
	//    base_url 以"模型路径前缀"结尾，Provider 执行器会自动追加动作后缀
	// ──────────────────────────────────────────────
	type modelSpec struct {
		providerID        int
		modelName         string
		toolFormat        string
		supportsThinking  bool
		supportsVision    bool
		dslRules          string
	}

	specs := []modelSpec{
		// ── Anthropic Claude ──────────────────────────────────────
		{anthropicID, "claude-3-7-sonnet-20250219", "anthropic", true, true, ""},
		{anthropicID, "claude-3-5-sonnet-20241022", "anthropic", true, true, ""},
		{anthropicID, "claude-3-5-haiku-20241022", "anthropic", true, true, ""},
		{anthropicID, "claude-3-opus-20240229", "anthropic", false, true, ""},
		{anthropicID, "claude-3-sonnet-20240229", "anthropic", false, true, ""},
		{anthropicID, "claude-3-haiku-20240307", "anthropic", false, true, ""},

		// ── Google AI Studio (Gemini) ─────────────────────────────
		// base_url 格式: 固定前缀/模型名，执行器追加 :generateContent 或 :streamGenerateContent
		{googleStudioID, "gemini-2.5-pro-preview-05-06", "google", true, true, ""},
		{googleStudioID, "gemini-2.5-flash-preview-04-17", "google", true, true, ""},
		{googleStudioID, "gemini-2.0-flash", "google", true, true, ""},
		{googleStudioID, "gemini-2.0-flash-lite", "google", false, true, ""},
		{googleStudioID, "gemini-1.5-pro", "google", false, true, ""},
		{googleStudioID, "gemini-1.5-flash", "google", false, true, ""},
		{googleStudioID, "gemini-1.5-flash-8b", "google", false, true, ""},

		// ── Google Vertex AI (Gemini on GCP) ──────────────────────
		{vertexID, "gemini-2.5-pro-preview-05-06", "google", true, true, ""},
		{vertexID, "gemini-2.0-flash", "google", true, true, ""},
		{vertexID, "gemini-2.0-flash-lite", "google", false, true, ""},
		{vertexID, "gemini-1.5-pro", "google", false, true, ""},
		{vertexID, "gemini-1.5-flash", "google", false, true, ""},

		// ── DeepSeek ──────────────────────────────────────────────
		{deepseekID, "deepseek-chat", "openai", false, false, ""},
		{deepseekID, "deepseek-reasoner", "openai", true, false, ""},

		// ── OpenAI ────────────────────────────────────────────────
		{openaiID, "gpt-4o", "openai", false, true, ""},
		{openaiID, "gpt-4o-mini", "openai", false, true, ""},
		{openaiID, "gpt-4-turbo", "openai", false, true, ""},
		{openaiID, "o1", "openai", true, true, ""},
		{openaiID, "o1-mini", "openai", true, false, ""},
		{openaiID, "o3-mini", "openai", true, false, ""},
		{openaiID, "o4-mini", "openai", true, false, ""},
	}

	for _, s := range specs {
		if s.providerID == 0 {
			continue // provider 未找到，跳过
		}
		thinking := 0
		if s.supportsThinking {
			thinking = 1
		}
		vision := 0
		if s.supportsVision {
			vision = 1
		}
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO model_specs
				(provider_id, model_name, tool_format, supports_thinking, supports_vision, dsl_rules)
			VALUES (?, ?, ?, ?, ?, ?)`,
			s.providerID, s.modelName, s.toolFormat, thinking, vision, s.dslRules,
		); err != nil {
			return err
		}
	}

	return nil
}
