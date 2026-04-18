// 核心入口：main.go
// 项目：Polaris Gateway (北极星自治 AI Agent 操作系统基建)
// 作者：mrlaoliai
// 设计哲学：Zero-CGO, State-in-DB (WAL), VFS-Storage, Zero-Poetry
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
	"github.com/mrlaoliai/polaris-gateway/internal/state"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
	"github.com/mrlaoliai/polaris-gateway/pkg/provider"
)

//go:embed ui/dist/*
var staticFiles embed.FS

func main() {
	// 启动全局生命周期 Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化地基：双库分离
	// primaryDB: 存放核心配置、账号、路由、配额（高频读取）
	primaryDB, err := database.InitDB("polaris.db")
	if err != nil {
		log.Fatalf("❌ 核心数据库初始化失败: %v", err)
	}
	defer primaryDB.Close()

	// l2DB: 仅存放长对话片段的索引，具体重型 Payload 沉降到 VFS 文件系统
	l2DB, err := database.InitDB("polaris_l2.db")
	if err != nil {
		log.Fatalf("❌ L2 索引库初始化失败: %v", err)
	}
	defer l2DB.Close()

	// 2. 启动全局异步写入中台 (DB Coordinator)
	// 彻底解决多协程并发写导致的 SQLite "database is locked" 问题
	dbMgr := database.NewDBManager(primaryDB, l2DB)
	dbMgr.StartWriterWorker(ctx)

	// 3. 实例化核心组件
	dslEngine, _ := dsl.NewEngine()
	router := orchestrator.NewRouter(primaryDB)
	sentinel := orchestrator.NewSentinel(primaryDB, dbMgr)
	guardian := middleware.NewGuardian(primaryDB, dbMgr)

	// 实例化 Session 管理器，将大内容负载重定向到本地 VFS 目录
	sessionMgr := state.NewSessionManager(primaryDB, "./data/vfs")

	log.Println("🛰️ Polaris Gateway v2.0 启动成功")
	log.Println("运行模式: [Dual-DB] [Async-Writer] [VFS-Storage]")

	// 4. 启动后台自愈拨测协程 (定期检查 Key 存活性)
	go sentinel.Start(ctx)

	// 5. 组装 HTTP 路由
	mux := http.NewServeMux()

	// 挂载控制台 (处理 SPA 路由回退，支持 History 模式刷新)
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles)))

	// 核心逻辑处理器：处理 Chat Completions 请求
	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A. 读取并预解析 Payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Read body failed", http.StatusBadRequest)
			return
		}

		var peek struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &peek)

		// B. 智能路由决策 (基于主库)
		target, err := router.Route(peek.Model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// C. [动态评估] DSL 规则改写物理映射
		if target.DSLRules != "" {
			var inputMap map[string]interface{}
			_ = json.Unmarshal(body, &inputMap)
			if result, err := dslEngine.ExecuteTransform(target.DSLRules, inputMap); err == nil {
				// 如果 DSL 命中并返回新模型名称，则执行强制覆盖
				if newModel, ok := result.(string); ok && newModel != "" {
					log.Printf("[DSL] 物理路由动态改写: %s -> %s", target.ModelName, newModel)
					target.ModelName = newModel
				}
			}
		}

		// D. 协议翻译器初始化 (默认适配 Claude Code 等 Anthropic 协议客户端)
		trans := transformer.NewAnthropicTransformer(target.ModelName)
		stdReq, err := trans.TransformRequest(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Protocol Transform Error: %v", err), http.StatusBadRequest)
			return
		}

		// E. 物理厂商执行器路由
		var executor provider.Executor
		switch target.Protocol {
		case "anthropic":
			executor = provider.NewAnthropicExecutor(target.APIKey, target.BaseURL)
		case "google", "vertex":
			executor = provider.NewGoogleExecutor(target.APIKey, target.BaseURL, target.Protocol == "vertex")
		default:
			http.Error(w, "Unsupported physical protocol", http.StatusInternalServerError)
			return
		}

		// F. 执行代理请求并实时重写响应
		if stdReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			physicalStream, err := executor.ExecuteStream(r.Context(), stdReq)
			if err != nil {
				// 通过 SSE 格式向客户端下发错误信息
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%v\"}\n\n", err)
				return
			}
			defer physicalStream.Close()

			// 执行流式转换 (内部自动处理 Heartbeat 注入与 Zero-Poetry 清洗)
			_ = trans.TransformStream(r.Context(), physicalStream, w)
		} else {
			// 非流式转发处理
			resp, err := executor.Execute(r.Context(), stdReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}

		// G. [核心改进] 异步更新配额消耗
		// 通过 dbMgr 将写操作推入队列，确保请求主路径以毫秒级速度返回
		if keyID, ok := r.Context().Value(middleware.GatewayKeyID).(int); ok {
			dbMgr.AsyncWrite("UPDATE gateway_keys SET used_tokens = used_tokens + ? WHERE id = ?", 1, keyID)
		}

		// H. 审计与 L2 溢出存储 (可选调用)
		_ = sessionMgr
	})

	// 挂载 Guardian 鉴权中间件，并注册 API 路径
	// 兼容 OpenAI 标准路径与 Anthropic 原生路径
	protectedHandler := guardian.AuthAndQuotaMiddleware(coreHandler)
	mux.Handle("/v1/chat/completions", protectedHandler)
	mux.Handle("/v1/messages", protectedHandler)

	// 6. 配置并启动 HTTP 服务
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Minute, // 适配 2026 年长时思维链推理
		WriteTimeout: 15 * time.Minute,
	}

	go func() {
		log.Println("🚀 网关服务监听在: http://0.0.0.0:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ 监听失败: %v", err)
		}
	}()

	// 7. 监听系统信号，实现优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⚠️ 接收到停机指令，正在清理资源...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ 服务强制关闭: %v", err)
	}
	log.Println("✅ Polaris 已安全退出")
}
