.PHONY: build dashboard all

VERSION ?= v$(shell TZ=Asia/Shanghai date +%Y.%m.%d)-$(shell git rev-parse --short HEAD)

# 编译前端 Dashboard（产物 dashboard/dist/ 由 Go 通过 go:embed 打包进二进制）
dashboard:
	@echo "Building dashboard..."
	cd dashboard && pnpm install --frozen-lockfile && pnpm build

build:
	@echo "Building the project..."
	@echo "Version: $(VERSION)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags=poll_opt -gcflags "all=-N -l" -ldflags "-X main.version=$(VERSION)" -o build/release/automatic cmd/automatic/main.go

# 一次完成前端与后端的完整构建
all: dashboard build
