package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reg_go/internal/storage"
	"syscall"
	"time"
)

// getEnv 获取环境变量，若为空返回默认值
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// StartHTTPServer 启动 HTTP API 服务
func StartHTTPServer() {
	port := getEnv("PORT", "8080")
	addr := ":" + port

	// 启动 token 清理
	go CleanupTokens()

	// 初始化存储目录（确保 /app/data 可用）
	initDataDir()

	// 创建路由
	mux := http.NewServeMux()

	// ---- 公开接口（无需登录） ----
	mux.HandleFunc("/api/login", handleLogin)

	// ---- 需要认证的接口 ----
	mux.HandleFunc("/api/overview", AuthMiddleware(handleOverview))
	mux.HandleFunc("/api/status", AuthMiddleware(handleStatus))
	mux.HandleFunc("/api/logs", AuthMiddleware(handleLogs))
	mux.HandleFunc("/api/task/start", AuthMiddleware(handleStartTask))
	mux.HandleFunc("/api/task/stop", AuthMiddleware(handleStopTask))
	mux.HandleFunc("GET /api/outlook", AuthMiddleware(handleOutlookGet))
	mux.HandleFunc("POST /api/outlook", AuthMiddleware(handleOutlookPost))
	mux.HandleFunc("DELETE /api/outlook", AuthMiddleware(handleOutlookDelete))
	mux.HandleFunc("POST /api/outlook/clear", AuthMiddleware(handleOutlookClear))
	mux.HandleFunc("POST /api/outlook/clear-registered", AuthMiddleware(handleOutlookClearRegistered))
	mux.HandleFunc("POST /api/outlook/import", AuthMiddleware(handleOutlookImport))
	mux.HandleFunc("/api/moemail", AuthMiddleware(handleMoeMail))
	mux.HandleFunc("/api/moemail/test", AuthMiddleware(handleMoeMailTest))
	mux.HandleFunc("/api/proxy", AuthMiddleware(handleProxy))
	mux.HandleFunc("/api/proxy/reset", AuthMiddleware(handleProxyReset))
	mux.HandleFunc("/api/datadir", AuthMiddleware(handleDataDir))
	mux.HandleFunc("/api/datadir/reset", AuthMiddleware(handleDataDirReset))
	mux.HandleFunc("/api/outputdir", AuthMiddleware(handleOutputDir))
	mux.HandleFunc("/api/outputdir/reset", AuthMiddleware(handleOutputDirReset))
	mux.HandleFunc("/api/accounts", AuthMiddleware(handleAccounts))
	mux.HandleFunc("/api/subscription/plans", AuthMiddleware(handleSubscriptionPlans))
	mux.HandleFunc("/api/subscription/link", AuthMiddleware(handleSubscriptionLink))
	mux.HandleFunc("/api/check-update", AuthMiddleware(handleCheckUpdate))

	// 健康检查（公开）
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	// 前端静态文件服务（放在最后，避免覆盖 API 路由）
	serveStaticFiles(mux)

	// CORS 中间件
	handler := corsMiddleware(mux)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[HTTP] 收到关闭信号，正在退出...")
		storage.FlushAccountsSync()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("[HTTP] 服务启动: http://0.0.0.0%s", addr)
	log.Printf("[HTTP] 默认密码: %s", getPassword())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[HTTP] 服务启动失败: %v", err)
	}
	fmt.Println("[HTTP] 服务已停止")
}

// corsMiddleware 添加 CORS 支持
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// initDataDir 初始化数据目录
func initDataDir() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/app/data"
	}
	os.MkdirAll(dataDir, 0755)
}

// serveStaticFiles 提供前端静态文件服务
func serveStaticFiles(mux *http.ServeMux) {
	// Try to serve embedded frontend/dist
	distDir := "frontend/dist"
	if _, err := os.Stat(distDir); err == nil {
		log.Printf("[HTTP] 静态文件服务: %s", distDir)
		// 使用自定义 handler，跳过 /api/ 路径
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// API 路径不处理
			if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
				http.NotFound(w, r)
				return
			}
			// SPA 路由: 如果文件不存在，返回 index.html
			filePath := distDir + r.URL.Path
			if info, err := os.Stat(filePath); err != nil || info.IsDir() {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
			http.FileServer(http.Dir(distDir)).ServeHTTP(w, r)
		})
	} else {
		log.Printf("[HTTP] 警告: 前端目录不存在: %s", distDir)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte("<h1>KiroX</h1><p>前端文件未找到。请检查 frontend/dist 目录。</p>"))
		})
	}
}
