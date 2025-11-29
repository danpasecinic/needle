.PHONY: all build test lint fmt vet clean cover help

GO := go
GOFLAGS := -v
MODULE := github.com/danpasecinic/needle
COVERAGE_FILE := coverage.out

all: fmt lint test

build:
	$(GO) build $(GOFLAGS) ./...

test:
	$(GO) test $(GOFLAGS) -race ./...

test-short:
	$(GO) test $(GOFLAGS) -short ./...

cover:
	$(GO) test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

cover-func:
	$(GO) test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -func=$(COVERAGE_FILE)

lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	$(GO) fmt ./...
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

vet:
	$(GO) vet ./...

clean:
	$(GO) clean
	rm -f $(COVERAGE_FILE) coverage.html

tidy:
	$(GO) mod tidy

deps:
	$(GO) mod download

check: fmt vet lint test

bench:
	$(GO) test -bench=. -benchmem ./...

doc:
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Starting godoc server at http://localhost:6060/pkg/$(MODULE)"
	godoc -http=:6060

help:
	@echo "Available targets:"
	@echo "  all        - Format, lint, and test"
	@echo "  build      - Build all packages"
	@echo "  test       - Run tests with race detector"
	@echo "  test-short - Run short tests only"
	@echo "  cover      - Generate coverage report"
	@echo "  cover-func - Show coverage by function"
	@echo "  lint       - Run golangci-lint"
	@echo "  lint-fix   - Run golangci-lint with auto-fix"
	@echo "  fmt        - Format code with gofmt and goimports"
	@echo "  vet        - Run go vet"
	@echo "  clean      - Clean build artifacts"
	@echo "  tidy       - Tidy go.mod"
	@echo "  deps       - Download dependencies"
	@echo "  check      - Run all checks (fmt, vet, lint, test)"
	@echo "  bench      - Run benchmarks"
	@echo "  doc        - Start godoc server"
	@echo "  help       - Show this help"
