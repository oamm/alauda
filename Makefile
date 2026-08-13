# Service Registry - Build Automation

.PHONY: help setup dev test build docker clean lint fmt proto gen-api

help:
	@echo "Service Registry - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make setup              Install dependencies"
	@echo "  make dev                Start development server (Go + React hot reload)"
	@echo "  make proto              Generate code from .proto files"
	@echo ""
	@echo "Building:"
	@echo "  make build              Build registry binary"
	@echo "  make docker-build       Build Docker image"
	@echo ""
	@echo "Testing:"
	@echo "  make test               Run all tests"
	@echo "  make test-race          Run tests with race detector"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint               Run golangci-lint"
	@echo "  make fmt                Format code with gofmt"
	@echo "  make vet                Run go vet"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean              Remove build artifacts"

# Installation & Setup

setup:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	cd web && npm install
	@echo "Setup complete!"

# Development

dev:
	@echo "Starting development environment..."
	docker-compose -f docker-compose.yml up -d
	@echo ""
	@echo "Services starting:"
	@echo "  Registry API:  http://localhost:9700"
	@echo "  Database:      localhost:5432 (postgres - future)"
	@echo ""
	@echo "To view logs: docker-compose logs -f"
	@echo "To stop:      docker-compose down"

dev-server:
	@echo "Starting registry server with hot reload..."
	@go run cmd/registry/main.go --config ./config.local.yaml

dev-ui:
	@echo "Starting React UI with hot reload..."
	cd web && npm run dev

# Protobuf & Code Generation

proto:
	@echo "Generating code from .proto files..."
	buf generate

gen-api: proto
	@echo "Generated API code"

# Building

build: proto
	@echo "Building registry binary..."
	go build -o bin/registry ./cmd/registry
	@echo "Binary: bin/registry"

build-cli: proto
	@echo "Building CLI tool..."
	go build -o bin/registryctl ./cmd/registryctl
	@echo "Binary: bin/registryctl"

# Testing

test: proto
	@echo "Running tests..."
	go test -v $$(go list ./... | grep -v '/web/' | grep -v '/gen/')

test-race: proto
	@echo "Running tests with race detector..."
	go test -race -v $$(go list ./... | grep -v '/web/' | grep -v '/gen/')

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p .artifacts
	go test -v -coverprofile=.artifacts/coverage.out $$(go list ./... | grep -v '/gen/' | grep -v '/web/' | grep -v '/.artifacts/')
	go tool cover -func .artifacts/coverage.out | tee .artifacts/coverage-summary.txt
	go tool cover -html=.artifacts/coverage.out -o .artifacts/coverage.html
	@echo "Coverage report: .artifacts/coverage.html"

test-coverage-ps:
	@powershell -ExecutionPolicy Bypass -File scripts/coverage.ps1

# Code Quality

lint:
	@echo "Running linter..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	gofmt -s -w ./cmd ./internal
	cd web && npm run fmt

vet:
	@echo "Running vet..."
	go vet ./...

# Docker

docker-build: proto
	@echo "Building Docker image..."
	docker build -t service-registry:latest .
	@echo "Image: service-registry:latest"

docker-run:
	@echo "Running Docker container..."
	docker run -p 9700:9700 -v ${PWD}/data:/data service-registry:latest

# Database Migrations

migrate-up:
	@echo "Running database migrations..."
	@go run cmd/registry/main.go migrate up

migrate-down:
	@echo "Rolling back database migrations..."
	@go run cmd/registry/main.go migrate down

# Cleanup

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf gen/
	rm -f coverage.out coverage.html
	rm -f .artifacts/coverage.out .artifacts/coverage.txt .artifacts/coverage-summary.txt .artifacts/coverage.html
	rm -f *.db *.db-shm *.db-wal
	@echo "Cleanup complete"

distclean: clean
	@echo "Removing all build artifacts and dependencies..."
	go clean -modcache
	cd web && rm -rf node_modules .next dist
	@echo "Full cleanup complete"
