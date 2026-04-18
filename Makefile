# Polaris Gateway v2.0 Makefile
# 坚守 Zero-CGO 与 State-in-DB 哲学

.PHONY: all ui build clean run

APP_NAME=polaris-gateway
VERSION=2.0.0

all: ui build

# 1. 编译前端 SPA
ui:
	@echo "=> Building Vue Dashboard..."
	cd ui && npm install && npm run build

# 2. 编译 Go 二进制 (Zero-CGO)
build:
	@echo "=> Compiling Go Binary (Zero-CGO)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/$(APP_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o build/$(APP_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/$(APP_NAME)-windows-amd64.exe .
	@echo "=> Build Complete! Binaries are in the build/ directory."

clean:
	@echo "=> Cleaning up..."
	rm -rf build/
	rm -rf ui/dist/

run: ui
	@echo "=> Running locally..."
	go run main.go