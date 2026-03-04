.PHONY: all build build-all docker-build test lint vuln secrets security-check clean

# Make all will just build
all: build

build:
	go build -ldflags="-s -w" -trimpath -o bin/gollm ./cmd/gollm

build-all:
	@echo "Building Linux (amd64) with CGO for eBPF support..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/gollm-linux-amd64 ./cmd/gollm
	@echo "Building Linux (arm64) without CGO (cross-compiler missing)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/gollm-linux-arm64 ./cmd/gollm
	@echo "Building Darwin (arm64) without CGO (eBPF unsupported on Mac)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/gollm-darwin-arm64 ./cmd/gollm

docker-build:
	docker build -t nexus-llm:latest .

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

secrets:
	gitleaks detect -v

# The comprehensive security pre-flight check
security-check:
	./scripts/ci_preflight.sh

clean:
	rm -rf bin/
