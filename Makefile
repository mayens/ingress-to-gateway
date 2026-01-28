# Makefile for ingress-to-gateway

# Variables
BINARY_NAME=ingress-to-gateway
VERSION?=0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/mayens/ingress-to-gateway/internal/version.Version=$(VERSION) -X github.com/mayens/ingress-to-gateway/internal/version.GitCommit=$(GIT_COMMIT) -X github.com/mayens/ingress-to-gateway/internal/version.BuildDate=$(BUILD_DATE)"

# Build targets
.PHONY: all build install clean test fmt vet lint help

all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## coverage: Generate coverage report
coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## lint: Run golangci-lint (requires golangci-lint to be installed)
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

## verify: Run verification checks (fmt, vet, test)
verify: fmt vet test

## run-audit: Run audit command (example)
run-audit: build
	./bin/$(BINARY_NAME) audit --all-namespaces

## run-convert: Run convert command (example)
run-convert: build
	./bin/$(BINARY_NAME) convert my-ingress -n default

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
