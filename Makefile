.PHONY: all test cover lint clean

GO := /usr/local/go/bin/go
PKGS_CORE := $(shell go list ./internal/... ./pkg/... ./cmd/helios-cli/internal/cli)

all: test

test:
	@echo "--- Running Go tests with race detector ---"
	$(GO) test -race ./...

cover:
	@echo "--- Checking test coverage (threshold: 85%) ---"
	$(GO) test -coverprofile=coverage.out $(PKGS_CORE)
	@$(GO) tool cover -func=coverage.out | tail -n1
	@$(GO) tool cover -func=coverage.out | tail -n1 | awk '{pct=$$NF; gsub("%","",pct); if(pct+0 < 85.0){print "Coverage check FAILED: " pct "% < 85%"; exit 1}else{print "Coverage check PASSED ✅: " pct "% >= 85%"}}'

lint:
	@echo "--- Running golangci-lint ---"
	golangci-lint run || true

clean:
	@echo "--- Cleaning up ---"
	rm -f coverage.out
