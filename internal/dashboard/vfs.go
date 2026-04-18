// 内部使用：internal/dashboard/vfs.go
// 作者：mrlaoliai
package dashboard

import (
	"io/fs"
	"net/http"
)

// WebUIHandler 接收从外部（如 main.go）传入的嵌入静态文件系统
func WebUIHandler(staticFiles fs.FS) http.Handler {
	// 剥离 `ui/dist` 前缀，直接将内部文件暴露给 HTTP
	subFS, err := fs.Sub(staticFiles, "ui/dist")
	if err != nil {
		panic("初始化 VFS 静态资源失败: " + err.Error())
	}
	return http.FileServer(http.FS(subFS))
}
