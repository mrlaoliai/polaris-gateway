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

	"github.com/mrlaoliai/polaris-gateway/internal/bridge/transformer"
	"github.com/mrlaoliai/polaris-gateway/internal/dashboard"
	"github.com/mrlaoliai/polaris-gateway/internal/database"
	"github.com/mrlaoliai/polaris-gateway/internal/orchestrator"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
	"github.com/mrlaoliai/polaris-gateway/pkg/provider"
)

//go:embed ui/src/*
var staticFiles embed.FS

func main() {
	// 1. 初始化 State-in-DB (SQLite WAL 模式)
	db, err := database.InitDB("polaris.db")
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	defer db.Close()

	log.Println("🛰️ Polaris Gateway v2.0 启动中...")
	log.Println("设计哲学: Zero-CGO, State-in-DB, Zero-Poetry")

	// 2. 实例化核心组件
	router := orchestrator.NewRouter(db)
	sentinel := orchestrator.NewSentinel(db)
	guardian := middleware.NewGuardian(db)

	// 3. 启动后台拨测
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sentinel.Start(ctx)

	// 4. 组装 HTTP 路由
	mux := http.NewServeMux()

	// 挂载 Dashboard (VFS 静态资源)
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles)))

	// 定义核心的 Chat Completions 处理逻辑
	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// A. 预解析获取客户端请求的虚拟模型名称
		var peek struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(body, &peek); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// B. 智能路由决策 (Virtual -> Physical)
		target, err := router.Route(peek.Model)
		if err != nil {
			http.Error(w, fmt.Sprintf("Routing error: %v", err), http.StatusServiceUnavailable)
			return
		}
		log.Printf("[Router] 映射成功: %s -> %s (Provider: %s)", peek.Model, target.ModelName, target.Protocol)

		// C. 实例化双向协议翻译器 (Bifrost 2.0)
		// 假设客户端使用的是 Anthropic 协议 (如 Claude Code)
		trans := transformer.NewAnthropicTransformer(target.ModelName)
		stdReq, err := trans.TransformRequest(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Transform error: %v", err), http.StatusBadRequest)
			return
		}

		// D. 实例化物理厂商执行器
		var executor provider.Executor
		switch target.Protocol {
		case "anthropic":
			executor = provider.NewAnthropicExecutor(target.APIKey, target.BaseURL)
		case "google", "vertex":
			isVertex := target.Protocol == "vertex"
			executor = provider.NewGoogleExecutor(target.APIKey, target.BaseURL, isVertex)
		default:
			http.Error(w, "Unsupported provider protocol", http.StatusInternalServerError)
			return
		}

		// E. 执行代理请求
		if stdReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			physicalStream, err := executor.ExecuteStream(r.Context(), stdReq)
			if err != nil {
				// 转换为 SSE 错误格式返回
				fmt.Fprintf(w, "event: error\ndata: {\"error\":{\"message\":\"%s\"}}\n\n", err.Error())
				return
			}
			defer physicalStream.Close()

			// 通过 Bifrost 状态机执行响应的实时重写、口癖过滤和影子签名
			if err := trans.TransformStream(r.Context(), physicalStream, w); err != nil {
				log.Printf("[Stream Error] %v", err)
			}
		} else {
			// 非流式请求处理
			w.Header().Set("Content-Type", "application/json")
			respData, err := executor.Execute(r.Context(), stdReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.Write(respData)
		}
	})

	// 将核心处理逻辑包裹在 Guardian 鉴权/配额拦截器中
	mux.Handle("/v1/chat/completions", guardian.AuthAndQuotaMiddleware(coreHandler))
	mux.Handle("/v1/messages", guardian.AuthAndQuotaMiddleware(coreHandler)) // 兼容原生 Anthropic 路径

	// 5. 启动 HTTP 服务与优雅停机
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("🚀 服务已监听在 http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("监听失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭网关...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}
	log.Println("Polaris 已安全退出")
}
