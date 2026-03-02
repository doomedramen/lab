.PHONY: run build test test-unit test-integration test-coverage clean proto proto-lint release

proto:
	cd ../../packages/proto && buf generate --template buf.gen.api.yaml

proto-lint:
	cd ../../packages/proto && buf lint

run:
	go run ./cmd/server

build: proto
	go build -o bin/server ./cmd/server

# Release builds with version info and stripped symbols
# Usage: make release VERSION=1.0.0
VERSION ?= dev
release: proto
	CGO_ENABLED=0 go build \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o bin/lab-server \
		./cmd/server

# Cross-compile for multiple platforms
# Usage: make release-all VERSION=1.0.0
release-all: release-linux-amd64 release-linux-arm64

release-linux-amd64: proto
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o bin/lab-server-linux-amd64 \
		./cmd/server

release-linux-arm64: proto
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o bin/lab-server-linux-arm64 \
		./cmd/server

test: test-unit test-integration

test-unit:
	go test -v -short ./...

test-integration:
	go test -v -tags=integration ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
