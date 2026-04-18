package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrlaoliai/polaris-gateway/internal/dashboard"
	"github.com/mrlaoliai/polaris-gateway/internal/database"
	"github.com/mrlaoliai/polaris-gateway/internal/orchestrator"
)

func main() {
	// 1. 初始化 State-in-DB (SQLite WAL 模式)
	db, err := database.InitDB("polaris.db")
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	defer db.Close()

	log.Println("🛰️ Polaris Gateway v2.0 启动中...")
	log.Println("设计哲学: Zero-CGO, State-in-DB, Zero-Poetry")

	// 2. 组装路由 (后续填充 Bifrost 逻辑)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		// 这里将是 Bifrost 2.0 引擎的入口
		w.Write([]byte("Polaris Gateway v2.0 Node Ready"))
	})

	// 1. 实例化核心组件
	router := orchestrator.NewRouter(db)
	sentinel := orchestrator.NewSentinel(db)

	// 2. 启动后台拨测
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sentinel.Start(ctx)

	// 3. 更新网关处理函数
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		// A. 身份校验 (使用 internal/database 中的 gateway_keys)
		// B. 路由决策
		target, err := router.Route("claude-3-5-sonnet") // 假设客户端请求的模型
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// C. 调用 Bifrost 2.0 执行协议转换与流式转发
		log.Printf("路由请求至: %s (%s)", target.ModelName, target.BaseURL)
		// transformer.TransformStream(...)
	})

	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler()))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// 3. 优雅停机处理
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("监听失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭网关...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}
	log.Println("Polaris 已安全退出")
}
