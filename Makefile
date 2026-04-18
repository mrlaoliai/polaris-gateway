# Polaris Gateway v2.0 Makefile
# 坚守 Zero-CGO 与 State-in-DB 哲学

.PHONY: all ui-deps ui build clean run dev-backend

APP_NAME=polaris-gateway
VERSION=2.0.0
BUILD_DIR=build
UI_DIR=ui
UI_DIST=$(UI_DIR)/dist

all: ui build

# 1. 仅在必要时安装前端依赖
ui-deps:
	@echo "=> Checking UI dependencies..."
	cd $(UI_DIR) && [ -d node_modules ] || npm install

# 2. 编译前端 SPA
# 增加 mkdir 确保即使构建失败，Go 的 embed 也不会报目录不存在的错
ui: ui-deps
	@echo "=> Building Vue Dashboard..."
	mkdir -p $(UI_DIST)
	touch $(UI_DIST)/.gitkeep
	cd $(UI_DIR) && npm run build

# 3. 编译 Go 二进制 (Zero-CGO)
# 显式依赖 ui，确保 dist 目录存在
build: ui
	@echo "=> Compiling Go Binary (Zero-CGO)..."
	mkdir -p $(BUILD_DIR)
	# Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	# macOS (M1/M2/M3)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "=> Build Complete! Binaries are in the $(BUILD_DIR)/ directory."

# 4. 清理
clean:
	@echo "=> Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -rf $(UI_DIST)

# 5. 完整运行 (含前端构建)
run: ui
	@echo "=> Running locally with UI..."
	go run main.go

# 6. 快速运行 (跳过前端构建，仅用于后端逻辑调试)
# 只要 ui/dist 目录存在即可编译通过
dev-backend:
    @mkdir -p $(UI_DIST)
    @touch $(UI_DIST)/.gitkeep
    @echo "=> Quick running backend..."
    go run main.go