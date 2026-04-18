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

// vfsWriterInterceptor 实现了 io.Writer 接口
// 它在数据流向客户端的同时，按顺序同步将数据分片存入 VFS 物理文件
type vfsWriterInterceptor struct {
	traceID    string
	startIndex int
	sessionMgr *state.SessionManager
	target     io.Writer
}

func (v *vfsWriterInterceptor) Write(p []byte) (n int, err error) {
	// 1. 优先下发数据给客户端，保证交互实时性
	n, err = v.target.Write(p)
	if err != nil {
		return n, err
	}

	// 2. 将数据片段存入 VFS (L2 存储)
	_ = v.sessionMgr.SpillToVFS(v.traceID, v.startIndex, p)
	v.startIndex++

	return n, nil
}

func main() {
	// 全局生命周期控制
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化双库地基
	// primaryDB: 存放核心配置、账号、路由、配额
	primaryDB, err := database.InitDB("polaris.db")
	if err != nil {
		log.Fatalf("❌ 核心数据库初始化失败: %v", err)
	}
	defer primaryDB.Close()

	// l2DB: 存放长对话片段的物理路径索引
	l2DB, err := database.InitDB("polaris_l2.db")
	if err != nil {
		log.Fatalf("❌ L2 索引库初始化失败: %v", err)
	}
	defer l2DB.Close()

	// 2. 启动全局异步写入中台 (消除 SQLite 锁竞争)
	dbMgr := database.NewDBManager(primaryDB, l2DB)
	dbMgr.StartWriterWorker(ctx)

	// 3. 实例化核心组件 (依赖注入)
	dslEngine, _ := dsl.NewEngine()
	router := orchestrator.NewRouter(primaryDB)
	sentinel := orchestrator.NewSentinel(primaryDB, dbMgr) // 内部应改用 dbMgr 更新状态
	guardian := middleware.NewGuardian(primaryDB, dbMgr)   // 内部应改用 dbMgr 更新配额

	// Session 管理器：使用 L2 库记录索引，内容存入物理目录
	sessionMgr := state.NewSessionManager(l2DB, dbMgr, "./data/vfs")

	log.Println("🛰️ Polaris Gateway v2.0 运行中...")

	// 后台自愈拨测
	go sentinel.Start(ctx)

	// 4. 组装 HTTP 路由
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles)))

	// 核心逻辑处理器
	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A. 生成对话追踪 ID 与初始索引
		traceID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
		chunkIndex := 0

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Read body failed", http.StatusBadRequest)
			return
		}

		// B. [备份] 将用户提问原样存入 VFS
		_ = sessionMgr.SpillToVFS(traceID, chunkIndex, body)
		chunkIndex++

		var peek struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &peek)

		// C. 路由决策与 DSL 动态重写
		target, err := router.Route(peek.Model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		if target.DSLRules != "" {
			var inputMap map[string]interface{}
			_ = json.Unmarshal(body, &inputMap)
			if result, err := dslEngine.ExecuteTransform(target.DSLRules, inputMap); err == nil {
				if newModel, ok := result.(string); ok && newModel != "" {
					target.ModelName = newModel
				}
			}
		}

		// D. 协议适配与物理执行器实例化
		trans := transformer.NewAnthropicTransformer(target.ModelName)
		stdReq, _ := trans.TransformRequest(body)

		var executor provider.Executor
		switch target.Protocol {
		case "anthropic":
			executor = provider.NewAnthropicExecutor(target.APIKey, target.BaseURL)
		case "google", "vertex":
			executor = provider.NewGoogleExecutor(target.APIKey, target.BaseURL, target.Protocol == "vertex")
		default:
			http.Error(w, "Unsupported Protocol", 500)
			return
		}

		// E. 执行代理并拦截响应流
		if stdReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")

			physicalStream, err := executor.ExecuteStream(r.Context(), stdReq)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%v\"}\n\n", err)
				return
			}
			defer physicalStream.Close()

			// 拦截器集成：数据在发送给客户端的同时，自动分片沉降到 VFS
			vfsInterceptor := &vfsWriterInterceptor{
				traceID:    traceID,
				startIndex: chunkIndex,
				sessionMgr: sessionMgr,
				target:     w,
			}

			_ = trans.TransformStream(r.Context(), physicalStream, vfsInterceptor)
		} else {
			// 非流式转发
			resp, _ := executor.Execute(r.Context(), stdReq)

			// 备份完整响应
			_ = sessionMgr.SpillToVFS(traceID, chunkIndex, resp)

			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}

		// F. 异步记录配额消耗
		if keyID, ok := r.Context().Value(middleware.GatewayKeyID).(int); ok {
			dbMgr.AsyncWrite("UPDATE gateway_keys SET used_tokens = used_tokens + ? WHERE id = ?", 1, keyID)
		}
	})

	// 注册 API 接口
	protectedHandler := guardian.AuthAndQuotaMiddleware(coreHandler)
	mux.Handle("/v1/chat/completions", protectedHandler)
	mux.Handle("/v1/messages", protectedHandler)

	// 5. 启动 HTTP 服务
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
	}

	go func() {
		log.Println("🚀 服务监听中: http://0.0.0.0:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⚠️ 停机信号接收，正在执行清理...")
	_ = server.Shutdown(context.Background())
	log.Println("✅ Polaris 已安全退出")
}
