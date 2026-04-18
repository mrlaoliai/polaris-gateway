// 内部使用：internal/orchestrator/sentinel.go
// 作者：mrlaoliai
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
	db     *sql.DB
	client *http.Client // [优化] 复用 Client 提升连接效率
}

func NewSentinel(db *sql.DB) *Sentinel {
	return &Sentinel{
		db: db,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
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

	// [优化] 限制拨测并发数为 20，防止 DB 连接池爆满或被厂商封禁
	semaphore := make(chan struct{}, 20)

	for rows.Next() {
		var id int
		var apiKey, protocol, baseURL string
		if err := rows.Scan(&id, &apiKey, &protocol, &baseURL); err != nil {
			continue
		}

		semaphore <- struct{}{} // 获取令牌
		go func(accountID int, key, proto, url string) {
			defer func() { <-semaphore }() // 释放令牌

			if !s.realPing(key, proto, url) {
				log.Printf("[Sentinel] 账号 [%d] 拨测失败，执行状态下线", accountID)
				_, err := s.db.Exec("UPDATE accounts SET status = 'error' WHERE id = ?", accountID)
				if err != nil {
					log.Printf("[Sentinel] 账号 [%d] 状态库更新失败: %v", accountID, err)
				}
			}
		}(id, apiKey, protocol, baseURL)
	}
}

func (s *Sentinel) realPing(apiKey, protocol, baseURL string) bool {
	var req *http.Request
	var err error

	switch protocol {
	case "openai", "deepseek":
		// 路由转换逻辑保持原有的优雅策略
		modelsURL := strings.Replace(baseURL, "/chat/completions", "/models", 1)
		req, err = http.NewRequest("GET", modelsURL, nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
	case "anthropic":
		req, err = http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		return true
	}

	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 只要不是 401/403，通常代表账号 Key 本身有效
	return resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden
}
