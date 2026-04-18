package dsl

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

type Engine struct {
	env *cel.Env
}

func NewEngine() (*Engine, error) {
	// 初始化 CEL 环境，定义可用于转换的变量
	env, err := cel.NewEnv(
		cel.Variable("msg", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		return nil, err
	}
	return &Engine{env: env}, nil
}

// ExecuteTransform 执行热更新的动态转换逻辑
func (e *Engine) ExecuteTransform(expression string, input map[string]interface{}) (string, error) {
	ast, iss := e.env.Compile(expression)
	if iss.Err() != nil {
		return "", iss.Err()
	}
	program, err := e.env.Program(ast)
	if err != nil {
		return "", err
	}

	out, _, err := program.Eval(map[string]interface{}{"msg": input})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", out.Value()), nil
}
