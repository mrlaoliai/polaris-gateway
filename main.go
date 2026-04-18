// 核心入口：main.go
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
	"github.com/mrlaoliai/polaris-gateway/internal/config"
	"github.com/mrlaoliai/polaris-gateway/internal/dashboard"
	"github.com/mrlaoliai/polaris-gateway/internal/database"
	"github.com/mrlaoliai/polaris-gateway/internal/orchestrator"
	"github.com/mrlaoliai/polaris-gateway/internal/state"
	"github.com/mrlaoliai/polaris-gateway/pkg/middleware"
	"github.com/mrlaoliai/polaris-gateway/pkg/provider"
)

//go:embed ui/dist/*
var staticFiles embed.FS

type vfsWriterInterceptor struct {
	traceID    string
	startIndex int
	sessionMgr *state.SessionManager
	target     io.Writer
}

func (v *vfsWriterInterceptor) Write(p []byte) (n int, err error) {
	n, err = v.target.Write(p) // Write 返回 2 个值
	if err != nil {
		return n, err
	}
	// SpillToVFS 返回 1 个值 (error)
	_ = v.sessionMgr.SpillToVFS(v.traceID, v.startIndex, p)
	v.startIndex++
	return n, nil
}

func main() {
	// 0. 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ 配置文件加载失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化库 (Close 返回 1 个值)
	primaryDB, err := database.InitDB(cfg.Database.Primary)
	if err != nil {
		log.Fatalf("❌ 核心数据库初始化失败: %v", err)
	}
	defer func() { _ = primaryDB.Close() }()

	l2DB, err := database.InitDB(cfg.Database.L2)
	if err != nil {
		log.Fatalf("❌ L2 索引库初始化失败: %v", err)
	}
	defer func() { _ = l2DB.Close() }()

	dbMgr := database.NewDBManager(primaryDB, l2DB)
	dbMgr.StartWriterWorker(ctx)

	// 2. 组件实例化
	dslEngine, _ := dsl.NewEngine()
	router := orchestrator.NewRouter(primaryDB)
	sentinel := orchestrator.NewSentinel(primaryDB, dbMgr)
	guardian := middleware.NewGuardian(primaryDB, dbMgr)
	sessionMgr := state.NewSessionManager(l2DB, dbMgr, cfg.Storage.VFSPath)

	limiter := middleware.NewConcurrentLimiter(cfg.Server.MaxConcurrency)

	log.Printf("🛰️ Polaris Gateway v2.0 运行中... (Port: %d, Max: %d)\n", cfg.Server.Port, cfg.Server.MaxConcurrency)
	go sentinel.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/dashboard/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("--> [UI]  访问页面: %s", r.URL.Path)
		// 原始逻辑
		handler := http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles))
		handler.ServeHTTP(w, r)
	})

	// 注册 REST 管理 API，让前端可以真正读写 SQLite
	apiHandler := dashboard.NewAPIHandler(primaryDB, dbMgr)
	apiHandler.RegisterRoutes(mux)

	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 强制打印所有进入网关接口的请求
		log.Printf("--> [API] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		traceID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
		chunkIndex := 0

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Read body failed", http.StatusBadRequest)
			return
		}

		// SpillToVFS 返回 1 个值
		_ = sessionMgr.SpillToVFS(traceID, chunkIndex, body)
		chunkIndex++

		var peek struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &peek)

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
		case "google":
			executor = provider.NewGoogleExecutor(target.APIKey, target.BaseURL)
		case "vertex":
			// [新增] 独立实例化的 Vertex 执行器
			executor = provider.NewVertexExecutor(target.APIKey, target.BaseURL)
		default:
			http.Error(w, "Unsupported Protocol", 500)
			return
		}

		if stdReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")

			physicalStream, err := executor.ExecuteStream(r.Context(), stdReq)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%v\"}\n\n", err)
				return
			}
			defer physicalStream.Close()

			vfsInterceptor := &vfsWriterInterceptor{
				traceID:    traceID,
				startIndex: chunkIndex,
				sessionMgr: sessionMgr,
				target:     w,
			}
			_ = trans.TransformStream(r.Context(), physicalStream, vfsInterceptor)
		} else {
			resp, _ := executor.Execute(r.Context(), stdReq)
			// SpillToVFS 返回 1 个值
			_ = sessionMgr.SpillToVFS(traceID, chunkIndex, resp)

			w.Header().Set("Content-Type", "application/json")
			// w.Write 返回 2 个值
			_, _ = w.Write(resp)
		}

		if keyID, ok := r.Context().Value(middleware.GatewayKeyID).(int); ok {
			dbMgr.AsyncWrite("UPDATE gateway_keys SET used_tokens = used_tokens + ? WHERE id = ?", 1, keyID)
		}
	})

	apiChain := guardian.AuthAndQuotaMiddleware(limiter.LimitMiddleware(coreHandler))
	mux.Handle("/v1/chat/completions", apiChain)
	mux.Handle("/v1/messages", apiChain)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Minute,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Minute,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	_ = server.Shutdown(context.Background())
}
