// 内部使用：internal/dashboard/vfs.go
// 作者：mrlaoliai
package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/dist/*
var staticFiles embed.FS

// WebUIHandler 提供编译入二进制的 Dashboard 前端文件
func WebUIHandler() http.Handler {
	// 剥离 `ui/dist` 前缀，直接将内部文件暴露给 HTTP
	subFS, err := fs.Sub(staticFiles, "ui/dist")
	if err != nil {
		panic("初始化 VFS 静态资源失败: " + err.Error())
	}
	return http.FileServer(http.FS(subFS))
}
