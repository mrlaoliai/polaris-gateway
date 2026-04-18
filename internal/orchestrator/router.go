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
	DSLRules  string // [新增] 存储从 model_specs 表中读取的动态转换规则
}

func (r *Router) Route(inModel string) (*TargetAccount, error) {
	// 联表查询：补充读取 m.dsl_rules 字段
	query := `
		SELECT 
			a.id, a.api_key, p.base_url, p.protocol_type, m.model_name, m.dsl_rules
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
		&target.DSLRules,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("找不到模型 [%s] 的可用路由", inModel)
	}
	return &target, err
}
