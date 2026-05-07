package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// LogFile holds the reference to the log file so it can be read by the API
var LogFile *os.File

func getLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./polaris-gateway.log"
	}
	
	dir := filepath.Join(home, ".polaris-gateway")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "./polaris-gateway.log"
	}
	
	return filepath.Join(dir, "polaris-gateway.log")
}

// InitLogger initializes the global slog instance.
func InitLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		// Optional: uncomment to log source file path and line number
		// AddSource: true,
	}

	logPath := getLogPath()
	// 尝试打开或创建 polaris-gateway.log
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err == nil {
		LogFile = f
		// 双写：既输出到控制台，又写入日志文件
		multiWriter := io.MultiWriter(os.Stdout, f)
		handler := slog.NewTextHandler(multiWriter, opts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	} else {
		// 如果文件打开失败，降级为仅控制台输出
		handler := slog.NewTextHandler(os.Stdout, opts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	}
}
