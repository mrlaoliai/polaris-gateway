// 内部使用：internal/bridge/dsl/engine.go
// 作者：mrlaoliai
package dsl

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Engine 封装了 CEL-go 评估逻辑，并支持 AST 编译缓存以提升高并发性能
type Engine struct {
	env   *cel.Env
	cache sync.Map // 用于缓存编译后的 cel.Program，避免重复编译
}

// NewEngine 初始化 DSL 引擎环境
func NewEngine() (*Engine, error) {
	// 定义 DSL 环境中可用的变量和函数
	e, err := cel.NewEnv(
		cel.Variable("msg", cel.DynType), // 允许传入任意结构的 JSON Map
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 CEL 环境失败: %w", err)
	}

	return &Engine{
		env: e,
	}, nil
}

// ExecuteTransform 执行动态表达式评估
// 返回 interface{} 以保留原始类型（String, Bool, Map 等），由调用方决定如何使用
func (e *Engine) ExecuteTransform(expression string, msg map[string]interface{}) (interface{}, error) {
	if expression == "" {
		return nil, nil
	}

	// 1. 尝试从缓存获取预编译的程序
	var prg cel.Program
	if val, ok := e.cache.Load(expression); ok {
		prg = val.(cel.Program)
	} else {
		// 2. 缓存未命中，执行编译（昂贵操作）
		ast, iss := e.env.Compile(expression)
		if iss.Err() != nil {
			return nil, fmt.Errorf("DSL 语法错误: %v", iss.Err())
		}

		newPrg, err := e.env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("生成 DSL 执行计划失败: %w", err)
		}

		// 存入缓存
		e.cache.Store(expression, newPrg)
		prg = newPrg
	}

	// 3. 准备执行上下文
	input := map[string]interface{}{
		"msg": msg,
	}

	// 4. 执行评估
	out, _, err := prg.Eval(input)
	if err != nil {
		return nil, fmt.Errorf("DSL 执行运行时错误: %w", err)
	}

	// 5. 智能处理返回值
	return convertToNative(out), nil
}

// convertToNative 将 CEL 内部类型转换回 Go 原生类型
func convertToNative(val ref.Val) interface{} {
	switch val.Type() {
	case types.BoolType:
		return val.Value().(bool)
	case types.IntType:
		return val.Value().(int64)
	case types.DoubleType:
		return val.Value().(float64)
	case types.StringType:
		return val.Value().(string)
	case types.ListType:
		return val.Value()
	case types.MapType:
		return val.Value()
	default:
		// 回退到字符串表示
		return fmt.Sprintf("%v", val.Value())
	}
}
