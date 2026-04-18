// 内部使用：internal/dashboard/vfs.go
// 作者：mrlaoliai
package dashboard

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

// WebUIHandler 接收嵌入的静态文件系统，并处理 SPA 路由回退逻辑
func WebUIHandler(staticFiles fs.FS) http.Handler {
	// 剥离 `ui/dist` 前缀
	subFS, err := fs.Sub(staticFiles, "ui/dist")
	if err != nil {
		panic("初始化虚拟文件系统失败: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 设置资源缓存策略 (1年长缓存，因为 Vite 打包文件名自带 Hash)
		if strings.Contains(r.URL.Path, ".js") || strings.Contains(r.URL.Path, ".css") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		// 2. 检查请求的文件在 VFS 中是否存在
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		_, err := fs.Stat(subFS, path)
		if err != nil && os.IsNotExist(err) {
			// 3. [核心修复] 如果文件不存在且不是静态资源请求，则重定向到 index.html
			// 这确保了 Vue Router 的 History 模式能够正常刷新页面而不报 404
			r.URL.Path = "/index.html"
		}

		fileServer.ServeHTTP(w, r)
	})
}
