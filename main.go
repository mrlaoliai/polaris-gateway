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
	"time"

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
	// 1. 初始化 State-in-DB (SQLite WAL 模式)
	db, err := database.InitDB("polaris.db")
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	defer db.Close()

	// 实例化核心组件
	dslEngine, err := dsl.NewEngine()
	if err != nil {
		log.Fatalf("无法加载 DSL 引擎: %v", err)
	}

	router := orchestrator.NewRouter(db)
	sentinel := orchestrator.NewSentinel(db)
	guardian := middleware.NewGuardian(db)

	log.Println("🛰️ Polaris Gateway v2.0 启动中...")
	log.Println("设计哲学: Zero-CGO, State-in-DB, Zero-Poetry")

	// 2. 启动后台拨测
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sentinel.Start(ctx)

	// 3. 组装 HTTP 路由
	mux := http.NewServeMux()

	// 挂载 Dashboard
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles)))

	// 定义核心处理逻辑
	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Read body failed", 400)
			return
		}

		var peek struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &peek)

		// A. 智能路由
		target, err := router.Route(peek.Model)
		if err != nil {
			http.Error(w, err.Error(), 503)
			return
		}

		// B. [集成] 执行动态 DSL 规则改写
		if target.DSLRules != "" {
			var inputMap map[string]interface{}
			_ = json.Unmarshal(body, &inputMap)
			newModel, dslErr := dslEngine.ExecuteTransform(target.DSLRules, inputMap)
			if dslErr == nil && newModel != "" {
				log.Printf("[DSL] 物理模型重写: %s -> %s", target.ModelName, newModel)
				target.ModelName = newModel
			}
		}

		// C. 实例化翻译器与执行器
		trans := transformer.NewAnthropicTransformer(target.ModelName)
		stdReq, _ := trans.TransformRequest(body)

		var executor provider.Executor
		switch target.Protocol {
		case "anthropic":
			executor = provider.NewAnthropicExecutor(target.APIKey, target.BaseURL)
		case "google", "vertex":
			executor = provider.NewGoogleExecutor(target.APIKey, target.BaseURL, target.Protocol == "vertex")
		default:
			http.Error(w, "Protocol mismatch", 500)
			return
		}

		// D. 代理转发
		if stdReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			stream, err := executor.ExecuteStream(r.Context(), stdReq)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%v\"}\n\n", err)
				return
			}
			defer stream.Close()
			_ = trans.TransformStream(r.Context(), stream, w)
		} else {
			resp, _ := executor.Execute(r.Context(), stdReq)
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}
	})

	// 挂载鉴权中间件并注册路由
	protectedHandler := guardian.AuthAndQuotaMiddleware(coreHandler)
	mux.Handle("/v1/chat/completions", protectedHandler)
	mux.Handle("/v1/messages", protectedHandler)

	// 4. 启动服务
	server := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		log.Println("🚀 服务监听在 :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在优雅停机...")
	sCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer sCancel()
	_ = server.Shutdown(sCtx)
}
