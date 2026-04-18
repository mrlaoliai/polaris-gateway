// internal/database/seed.go
// 作者：mrlaoliai
// 设计哲学：State-in-DB — 系统启动时自动写入 system_providers 和 system_models 种子数据
// 策略：使用 SQLite UPSERT (ON CONFLICT DO UPDATE)，既能首次写入也能同步更新，完全幂等
package database

import (
	"database/sql"
	"fmt"
	"log"
)

// Seed 写入所有预置的系统厂商与模型规格种子数据
// 必须在 migrate() 之后调用，以确保表已存在
func Seed(db *sql.DB) error {
	if err := seedProviders(db); err != nil {
		return fmt.Errorf("seedProviders: %w", err)
	}
	if err := seedModels(db); err != nil {
		return fmt.Errorf("seedModels: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────
// system_providers 种子数据
// ──────────────────────────────────────────────────────────────────
func seedProviders(db *sql.DB) error {
	type Provider struct {
		id           string
		name         string
		protocol     string
		urlTemplate  string
		authType     string
		authConfig   string
		readTimeout  int
		capabilities string
	}

	providers := []Provider{
		// Google AI Studio (开发者快速通道)
		{
			"google-ai-studio",
			"Google AI Studio (Gemini API)",
			"google-ai",
			"https://generativelanguage.googleapis.com/v1beta/models/{model_id}",
			"api-key",
			`{"location":"header","key_name":"x-goog-api-key","prefix":""}`,
			300,
			`{"note":"Fast prototyping, easy auth"}`,
		},
		// Google Cloud Vertex AI (企业级通道)
		{
			"google-vertex",
			"Google Cloud Vertex AI",
			"vertex",
			"https://{region}-aiplatform.googleapis.com/v1/publishers/google/models/{model_id}",
			"oauth2",
			`{"location":"header","key_name":"Authorization","prefix":"Bearer "}`,
			300,
			`{"regions":["us-central1","europe-west4","asia-northeast1"]}`,
		},
		// OpenAI
		{
			"openai",
			"OpenAI",
			"openai",
			"https://api.openai.com/v1/chat/completions",
			"api-key",
			`{"location":"header","key_name":"Authorization","prefix":"Bearer "}`,
			120,
			`{"batch":true}`,
		},
		// Anthropic
		{
			"anthropic",
			"Anthropic",
			"anthropic",
			"https://api.anthropic.com/v1/messages",
			"api-key",
			`{"location":"header","key_name":"x-api-key","prefix":""}`,
			180,
			`{"caching":true}`,
		},
		// Groq (Meta Llama)
		{
			"meta-groq",
			"Groq (Meta Llama)",
			"openai",
			"https://api.groq.com/openai/v1/chat/completions",
			"api-key",
			`{"location":"header","key_name":"Authorization","prefix":"Bearer "}`,
			60,
			`{"note":"Ultra high speed inference"}`,
		},
		// DeepSeek
		{
			"deepseek",
			"DeepSeek",
			"openai",
			"https://api.deepseek.com/v1/chat/completions",
			"api-key",
			`{"location":"header","key_name":"Authorization","prefix":"Bearer "}`,
			120,
			`{"note":"Cost-effective reasoning models"}`,
		},
	}

	// UPSERT：已存在则更新，不存在则插入
	for _, p := range providers {
		res, err := db.Exec(`
			INSERT INTO system_providers
				(id, name, protocol, url_template, auth_type, auth_config, read_timeout, capabilities)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				name         = excluded.name,
				protocol     = excluded.protocol,
				url_template = excluded.url_template,
				auth_type    = excluded.auth_type,
				auth_config  = excluded.auth_config,
				read_timeout = excluded.read_timeout,
				capabilities = excluded.capabilities,
				updated_at   = CURRENT_TIMESTAMP`,
			p.id, p.name, p.protocol, p.urlTemplate, p.authType, p.authConfig, p.readTimeout, p.capabilities,
		)
		if err != nil {
			return fmt.Errorf("upsert provider %s: %w", p.id, err)
		}
		rows, _ := res.RowsAffected()
		if rows > 0 {
			log.Printf("[Seed] system_providers upserted: %s", p.id)
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────
// system_models 种子数据 (2026 旗舰级)
// ──────────────────────────────────────────────────────────────────
func seedModels(db *sql.DB) error {
	type Model struct {
		providerID       string
		modelID          string
		modelName        string
		toolFormat       string
		maxContext       int
		supportsThinking bool
		supportsVision   bool
		supportsTools    bool
		supportsJSON     bool
		dslRules         string
	}

	models := []Model{
		// ── OpenAI ──────────────────────────────────────────────────
		{
			"openai", "gpt-5.4-omni", "GPT-5.4 Omni",
			"openai", 1050000, false, true, true, true,
			`res.content = res.content.replace(/^Certainly! /i, "").replace(/^How can I help you today\?/i, "").trim()`,
		},
		{
			"openai", "gpt-5.4-thinking", "GPT-5.4 Pro (Deep Reasoning)",
			"openai", 1050000, true, true, true, true,
			`res.thinking = res.thinking.strip(); res.content = res.content.replace(/^(Based on my analysis|After deep thought), /i, "")`,
		},

		// ── Anthropic ────────────────────────────────────────────────
		{
			"anthropic", "claude-4.7-opus-202604", "Claude 4.7 Opus",
			"anthropic", 1000000, true, true, true, true,
			`res.content = res.content.replace(/<thinking>[\s\S]*?<\/thinking>/g, ""); res.content = res.content.replace(/^I understand. /i, "")`,
		},
		{
			"anthropic", "claude-4.6-sonnet-202602", "Claude 4.6 Sonnet",
			"anthropic", 1000000, false, true, true, true,
			`res.content = res.content.replace(/^Certainly, I can help with that. /i, "")`,
		},

		// ── Google AI Studio ─────────────────────────────────────────
		{
			"google-ai-studio", "gemini-3.1-pro", "Gemini 3.1 Pro",
			"google", 2000000, true, true, true, true,
			`res.content = res.content.replace(/\*\*Thought:\*\*[\s\S]*?\n\n/i, "")`,
		},
		{
			"google-ai-studio", "gemini-3.1-flash", "Gemini 3.1 Flash",
			"google", 1000000, false, true, true, true,
			`res.content = res.content.trim()`,
		},

		// ── Google Vertex AI ─────────────────────────────────────────
		{
			"google-vertex", "gemini-3.1-pro", "Gemini 3.1 Pro (Enterprise)",
			"google", 2000000, true, true, true, true,
			`res.content = res.content.replace(/^(Alright|Sure), /i, "")`,
		},

		// ── DeepSeek ─────────────────────────────────────────────────
		{
			"deepseek", "deepseek-v4", "DeepSeek V4 (Universal)",
			"openai", 256000, true, true, true, true,
			`res.content = res.content.replace(/^Okay, /i, "")`,
		},

		// ── Meta / Groq ──────────────────────────────────────────────
		{
			"meta-groq", "llama-4-maverick-70b", "Llama 4 Maverick (Groq)",
			"openai", 128000, false, true, true, true,
			`res.content = res.content.replace(/^Assistant: /i, "").trim()`,
		},
	}

	upserted := 0
	for _, m := range models {
		t := btob(m.supportsThinking)
		v := btob(m.supportsVision)
		tools := btob(m.supportsTools)
		j := btob(m.supportsJSON)

		// ON CONFLICT DO UPDATE：唯一索引 (provider_id, model_id) 冲突时更新
		res, err := db.Exec(`
			INSERT INTO system_models
				(provider_id, model_id, model_name, tool_format, max_context,
				 supports_thinking, supports_vision, supports_tools, supports_json, dsl_rules)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(provider_id, model_id) DO UPDATE SET
				model_name        = excluded.model_name,
				tool_format       = excluded.tool_format,
				max_context       = excluded.max_context,
				supports_thinking = excluded.supports_thinking,
				supports_vision   = excluded.supports_vision,
				supports_tools    = excluded.supports_tools,
				supports_json     = excluded.supports_json,
				dsl_rules         = excluded.dsl_rules`,
			m.providerID, m.modelID, m.modelName, m.toolFormat, m.maxContext,
			t, v, tools, j, m.dslRules,
		)
		if err != nil {
			return fmt.Errorf("upsert model %s/%s: %w", m.providerID, m.modelID, err)
		}
		rows, _ := res.RowsAffected()
		if rows > 0 {
			upserted++
			log.Printf("[Seed] system_models upserted: %s / %s", m.providerID, m.modelID)
		}
	}
	log.Printf("[Seed] system_models done: %d/%d rows affected", upserted, len(models))
	return nil
}

// btob 将 bool 转为 SQLite BOOLEAN 整数 (0/1)
// 注意：api_providers.go 中的 btoi 功能相同，此处独立定义避免跨文件依赖
func btob(v bool) int {
	if v {
		return 1
	}
	return 0
}
