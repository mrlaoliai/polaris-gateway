package orchestrator

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"
)

type Sentinel struct {
	db *sql.DB
}

func NewSentinel(db *sql.DB) *Sentinel {
	return &Sentinel{db: db}
}

func (s *Sentinel) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAccounts(ctx)
		}
	}
}

func (s *Sentinel) checkAccounts(ctx context.Context) {
	// 联表查询，获取协议类型和基础地址，以便进行真实的物理拨测
	query := `
		SELECT a.id, a.api_key, p.protocol_type, p.base_url 
		FROM accounts a 
		JOIN providers p ON a.provider_id = p.id 
		WHERE a.status = 'active'
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("[Sentinel] 获取账号列表失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var apiKey, protocol, baseURL string
		if err := rows.Scan(&id, &apiKey, &protocol, &baseURL); err != nil {
			continue
		}

		go func(accountID int, key, proto, url string) {
			if !s.realPing(key, proto, url) {
				log.Printf("[Sentinel] 账号 [%d] 拨测失败，执行下线处理", accountID)
				_, err := s.db.Exec("UPDATE accounts SET status = 'error' WHERE id = ?", accountID)
				if err != nil {
					log.Printf("[Sentinel] 账号 [%d] 状态更新失败: %v", accountID, err)
				}
			}
		}(id, apiKey, protocol, baseURL)
	}
}

// realPing 执行真实的物理层探活
func (s *Sentinel) realPing(apiKey, protocol, baseURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	var req *http.Request
	var err error

	switch protocol {
	case "openai", "deepseek":
		// OpenAI 兼容协议通常有 /models 接口可以免费低频测试
		modelsURL := strings.Replace(baseURL, "/chat/completions", "/models", 1)
		req, err = http.NewRequest("GET", modelsURL, nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
	case "anthropic":
		// Anthropic 可以发送一个极其简单的 1 token 错误请求或验证 headers
		req, err = http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		// 其他协议暂默认放行或走特殊逻辑
		return true
	}

	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 只要不是 401(未授权) 或 403(被封禁/无配额) 或 429(耗尽)，都认为是存活的
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false
	}
	return true
}
