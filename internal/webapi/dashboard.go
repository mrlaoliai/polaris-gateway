package webapi

import (
	"io/fs"
	"log/slog"
	"net/http"

	"polaris-gateway/web"
)

// DashboardHandler 返回用于服务前端 UI 的静态文件处理器
func DashboardHandler() http.Handler {
	uiSub, err := fs.Sub(web.FS, "ui")
	if err != nil {
		slog.Error("Failed to mount UI filesystem", "error", err)
		return http.NotFoundHandler()
	}
	
	fileServer := http.FileServer(http.FS(uiSub))
	return http.StripPrefix("/dashboard/", fileServer)
}
