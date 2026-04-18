// 核心入口：main.go
// 项目：Polaris Gateway
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
	// 1. 优先下发数据给客户端
	n, err = v.target.Write(p)
	if err != nil {
		return n, err
	}

	// 2. 将数据片段存入 VFS，显式忽略备份错误以防阻塞主业务
	_ = v.sessionMgr.SpillToVFS(v.traceID, v.startIndex, p)
	v.startIndex++

	return n, nil
}

func main() {
	// 0. 加载配置文件
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Printf("⚠️ 未找到配置文件，使用默认配置: %v", err)
		// 这里可以设置一套默认值，或者直接 fatal
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 使用配置初始化数据库
	primaryDB, _ := database.InitDB(cfg.Database.Primary)
	if err != nil {
		log.Fatalf("❌ 核心数据库初始化失败: %v", err)
	}
	defer primaryDB.Close()
	l2DB, _ := database.InitDB(cfg.Database.L2)
	if err != nil {
		log.Fatalf("❌ L2 索引库初始化失败: %v", err)
	}
	defer l2DB.Close()

	// 2. 启动异步写入中台
	dbMgr := database.NewDBManager(primaryDB, l2DB)
	dbMgr.StartWriterWorker(ctx)

	// 3. 实例化核心组件
	dslEngine, _ := dsl.NewEngine()
	router := orchestrator.NewRouter(primaryDB)
	sentinel := orchestrator.NewSentinel(primaryDB, dbMgr)
	guardian := middleware.NewGuardian(primaryDB, dbMgr)
	// 初始化 Session 管理器
	sessionMgr := state.NewSessionManager(l2DB, dbMgr, cfg.Storage.VFSPath)

	log.Println("🛰️ Polaris Gateway v2.0 运行中...")
	go sentinel.Start(ctx)

	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.WebUIHandler(staticFiles)))

	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
		chunkIndex := 0

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Read body failed", http.StatusBadRequest)
			return
		}

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

			// 执行转换流
			_ = trans.TransformStream(r.Context(), physicalStream, vfsInterceptor)
		} else {
			resp, _ := executor.Execute(r.Context(), stdReq)
			_ = sessionMgr.SpillToVFS(traceID, chunkIndex, resp)

			w.Header().Set("Content-Type", "application/json")
			// 修正赋值不匹配与 errcheck：显式忽略最终响应写入错误
			_, _ = w.Write(resp)
		}

		if keyID, ok := r.Context().Value(middleware.GatewayKeyID).(int); ok {
			dbMgr.AsyncWrite("UPDATE gateway_keys SET used_tokens = used_tokens + ? WHERE id = ?", 1, keyID)
		}
	})

	protectedHandler := guardian.AuthAndQuotaMiddleware(coreHandler)
	mux.Handle("/v1/chat/completions", protectedHandler)
	mux.Handle("/v1/messages", protectedHandler)

	// 3. 动态组装监听地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Minute,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Minute,
	}

	go func() {
		log.Printf("🚀 Polaris Gateway 运行于 http://%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⚠️ 停机信号接收，正在执行清理...")
	_ = server.Shutdown(context.Background())
	log.Println("✅ Polaris 已安全退出")
}
