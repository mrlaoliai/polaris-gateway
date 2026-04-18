package orchestrator

import (
	"database/sql"
	"fmt"
)

type Router struct {
	db *sql.DB
}

func NewRouter(db *sql.DB) *Router {
	return &Router{db: db}
}

type TargetAccount struct {
	AccountID int
	APIKey    string
	BaseURL   string
	ModelName string
	Protocol  string
}

// Route 根据虚拟模型名（如 claude-3-opus）寻找最佳物理账号
func (r *Router) Route(inModel string) (*TargetAccount, error) {
	query := `
		SELECT 
			a.id, a.api_key, p.base_url, p.protocol_type, m.model_name
		FROM routing_rules rr
		JOIN model_specs m ON rr.target_spec_id = m.id
		JOIN providers p ON m.provider_id = p.id
		JOIN accounts a ON p.id = a.provider_id
		WHERE rr.in_model = ? AND a.status = 'active'
		ORDER BY a.priority DESC
		LIMIT 1
	`

	var target TargetAccount
	err := r.db.QueryRow(query, inModel).Scan(
		&target.AccountID,
		&target.APIKey,
		&target.BaseURL,
		&target.Protocol,
		&target.ModelName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("找不到模型 [%s] 的可用路由或账号已耗尽", inModel)
	}
	if err != nil {
		return nil, err
	}

	return &target, nil
}
