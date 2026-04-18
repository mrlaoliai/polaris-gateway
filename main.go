// 核心入口：main.go
// 项目：Polaris Gateway v2.0
// 作者：mrlaoliai
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/dsl"
	"github.com/mrlaoliai/polaris-gateway/internal/bridge/transformer"
	"github.com/mrlaoliai/polaris-gateway/internal/dashboard"
	"github.com/mrlaoliai/polaris-gateway/internal/database"
	"github.com/mrlaoliai/polaris-gateway/internal/orchestrator"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
	"github.com/mrlaoliai/polaris-gateway/pkg/provider"
)

//go:embed ui/dist/*
var staticFiles embed.FS

func main() {
	// 1. 初始化地基：State-in-DB
	db, err := database.InitDB("polaris.db")
	if err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer db.Close()

	// 2. 实例化所有核心“器官”
	dslEngine, _ := dsl.NewEngine()
	router := orchestrator.NewRouter(db)
	sentinel := orchestrator.NewSentinel(db)
	guardian := middleware.NewGuardian(db)

	log.Println("🛰️ Polaris Gateway v2.0 已就绪")

	// 3. 启动后台自愈拨测
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sentinel.Start(ctx)

	// 4. 路由逻辑
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles)))

	// 核心逻辑处理器
	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var peek struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &peek)

		// A. 智能调度决策
		target, err := router.Route(peek.Model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// B. [深度集成] DSL 动态改写
		if target.DSLRules != "" {
			var inputMap map[string]interface{}
			_ = json.Unmarshal(body, &inputMap)
			if result, err := dslEngine.ExecuteTransform(target.DSLRules, inputMap); err == nil {
				// 如果 DSL 评估结果是字符串，则视为改写物理模型名称
				if newModel, ok := result.(string); ok && newModel != "" {
					target.ModelName = newModel
				}
			}
		}

		// C. 实例化翻译器 (Bifrost 2.0)
		trans := transformer.NewAnthropicTransformer(target.ModelName)
		stdReq, _ := trans.TransformRequest(body)

		// D. 实例化执行器
		var executor provider.Executor
		switch target.Protocol {
		case "anthropic":
			executor = provider.NewAnthropicExecutor(target.APIKey, target.BaseURL)
		case "google", "vertex":
			executor = provider.NewGoogleExecutor(target.APIKey, target.BaseURL, target.Protocol == "vertex")
		default:
			http.Error(w, "未知的物理协议", 500)
			return
		}

		// E. 执行代理
		if stdReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			stream, err := executor.ExecuteStream(r.Context(), stdReq)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%v\"}\n\n", err)
				return
			}
			defer stream.Close()

			// 执行流式转换 (内部已集成 Heartbeat 和 Zero-Poetry)
			_ = trans.TransformStream(r.Context(), stream, w)
		} else {
			resp, _ := executor.Execute(r.Context(), stdReq)
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}

		// F. [深度集成] 记录配额消耗 (异步)
		if keyID, ok := r.Context().Value(middleware.GatewayKeyID).(int); ok {
			guardian.RecordUsage(keyID, 1) // 简化计费：每请求 1 次，未来可改为 Token 统计
		}
	})

	// 挂载防御中间件
	protectedHandler := guardian.AuthAndQuotaMiddleware(coreHandler)
	mux.Handle("/v1/chat/completions", protectedHandler)
	mux.Handle("/v1/messages", protectedHandler)

	// 5. 启动服务与优雅停机
	server := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		log.Println("🚀 监听地址 http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Polaris 关闭中...")
	_ = server.Shutdown(context.Background())
}
