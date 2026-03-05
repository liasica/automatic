.PHONY: build

build:
	@echo "Building the project..."
	GO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags=sonic,poll_opt -gcflags "all=-N -l" -o build/release/automatic cmd/automatic/main.go
