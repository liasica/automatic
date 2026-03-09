.PHONY: build

VERSION ?= v$(shell TZ=Asia/Shanghai date +%Y.%m.%d)-$(shell git rev-parse --short HEAD)

build:
	@echo "Building the project..."
	@echo "Version: $(VERSION)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags=poll_opt -gcflags "all=-N -l" -ldflags "-X main.version=$(VERSION)" -o build/release/automatic cmd/automatic/main.go
