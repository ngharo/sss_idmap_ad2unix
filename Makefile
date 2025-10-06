.PHONY: all build test clean fmt lint install help

# Build variables
BINARY_NAME=sss-idmap
PKG=github.com/ngharo/sss_idmap_ad2unix
CMD_DIR=./cmd/sss-idmap
PKG_DIR=./pkg/...

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

all: fmt build

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_DIR)

install: ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(CMD_DIR)

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out $(PKG_DIR)
	go tool cover -func=coverage.out

fmt: ## Format code with goimports
	@echo "Formatting code..."
	@command -v goimports >/dev/null 2>&1 || { echo "goimports not found, installing..."; go install golang.org/x/tools/cmd/goimports@latest; }
	goimports -w -local $(PKG) .

lint: ## Run linters
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install from https://golangci-lint.run/"; exit 1; }
	golangci-lint run

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) coverage.out

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
