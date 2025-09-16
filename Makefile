.PHONY: all test cover lint clean build build-all-platforms release-check release-tag docker-build install dev-setup

GO := /usr/local/go/bin/go
PKGS_CORE := $(shell go list ./internal/... ./pkg/... ./cmd/helios-cli/internal/cli)
BINARY_NAME := helios
BUILD_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

all: test build

# Development
dev-setup:
	@echo "--- Setting up development environment ---"
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Testing
test:
	@echo "--- Running Go tests with race detector ---"
	$(GO) test -race ./...

test-go: test

cover:
	@echo "--- Checking test coverage (threshold: 85%) ---"
	$(GO) test -coverprofile=coverage.out $(PKGS_CORE)
	@$(GO) tool cover -func=coverage.out | tail -n1
	@$(GO) tool cover -func=coverage.out | tail -n1 | awk '{pct=$$NF; gsub("%","",pct); if(pct+0 < 85.0){print "Coverage check FAILED: " pct "% < 85%"; exit 1}else{print "Coverage check PASSED ✅: " pct "% >= 85%"}}'

cover-check-go: cover

# Linting
lint:
	@echo "--- Running golangci-lint ---"
	golangci-lint run

# Building
build:
	@echo "--- Building $(BINARY_NAME) ---"
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/helios-cli

build-all-platforms:
	@echo "--- Building for all platforms ---"
	@mkdir -p $(BUILD_DIR)

	# Linux
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/helios-cli
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/helios-cli

	# macOS
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/helios-cli
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/helios-cli

	# Windows
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/helios-cli

# Installation
install: build
	@echo "--- Installing $(BINARY_NAME) ---"
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Release management
release-check:
	@echo "--- Checking if ready for release ---"
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION not set"; exit 1; fi
	@if [ -z "$(shell git status --porcelain)" ]; then echo "✅ Working tree is clean"; else echo "❌ Working tree has uncommitted changes"; exit 1; fi
	@echo "✅ Ready for release $(VERSION)"

release-tag:
	@echo "--- Creating release tag ---"
	@if [ -z "$(TAG)" ]; then echo "Error: TAG not set (use make release-tag TAG=v1.0.0)"; exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)

release-notes:
	@echo "--- Generating release notes ---"
	@if [ -z "$(TAG)" ]; then echo "Error: TAG not set"; exit 1; fi
	@echo "# Release $(TAG)"
	@echo ""
	@echo "## Changes"
	@git log --oneline --no-merges $(shell git describe --tags --abbrev=0 $(TAG)^)..$(TAG) | sed 's/^/- /'

# Docker
docker-build:
	@echo "--- Building Docker image ---"
	docker build -t ghcr.io/good-night-oppie/helios:$(VERSION) .
	docker build -t ghcr.io/good-night-oppie/helios:latest .

docker-push:
	@echo "--- Pushing Docker image ---"
	docker push ghcr.io/good-night-oppie/helios:$(VERSION)
	docker push ghcr.io/good-night-oppie/helios:latest

# Cleanup
clean:
	@echo "--- Cleaning up ---"
	rm -f coverage.out
	rm -rf $(BUILD_DIR)
	docker rmi ghcr.io/good-night-oppie/helios:$(VERSION) 2>/dev/null || true
	docker rmi ghcr.io/good-night-oppie/helios:latest 2>/dev/null || true

# Help
help:
	@echo "Available targets:"
	@echo "  dev-setup           - Set up development environment"
	@echo "  test                - Run tests with race detector"
	@echo "  cover               - Run tests with coverage check (85% threshold)"
	@echo "  lint                - Run linting"
	@echo "  build               - Build binary for current platform"
	@echo "  build-all-platforms - Build binaries for all platforms"
	@echo "  install             - Install binary to /usr/local/bin"
	@echo "  release-check       - Check if ready for release"
	@echo "  release-tag         - Create and push release tag (set TAG=v1.0.0)"
	@echo "  release-notes       - Generate release notes (set TAG=v1.0.0)"
	@echo "  docker-build        - Build Docker image"
	@echo "  docker-push         - Push Docker image"
	@echo "  clean               - Clean up build artifacts"
	@echo "  help                - Show this help message"
