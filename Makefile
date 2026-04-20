# Makefile for Fuse Video Streamer

.PHONY: help build test test-verbose test-coverage clean lint fmt deps dev-setup

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the application
	cd app && go build -o ../bin/fuse_video_streamer .

build-release: ## Build release version
	cd app && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ../bin/fuse_video_streamer .

# Test targets
test: ## Run tests
	cd app && go test ./...

test-verbose: ## Run tests with verbose output
	cd app && go test -v ./...

test-coverage: ## Run tests with coverage
	cd app && go test -coverprofile=coverage.out ./...
	cd app && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: app/coverage.html"

test-coverage-text: ## Show test coverage in terminal
	cd app && go test -coverprofile=coverage.out ./...
	cd app && go tool cover -func=coverage.out

# Code quality targets
lint: ## Run linter
	cd app && golangci-lint run

fmt: ## Format code
	cd app && go fmt ./...
	cd app && goimports -w .

vet: ## Run go vet
	cd app && go vet ./...

# Dependency management
deps: ## Download dependencies
	cd app && go mod download

deps-update: ## Update dependencies
	cd app && go get -u ./...
	cd app && go mod tidy

deps-verify: ## Verify dependencies
	cd app && go mod verify

# Development setup
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Development tools installed!"

# Docker targets
docker-build: ## Build Docker image
	docker build -t fuse_video_streamer:dev .

docker-run: ## Run Docker container (requires config.yml)
	docker run --rm \
		--cap-add SYS_ADMIN \
		--device /dev/fuse:/dev/fuse:rwm \
		--pid host \
		-v $(PWD)/config.yml:/app/config.yml \
		-v $(PWD)/mnt:/mnt/fvs:rshared \
		fuse_video_streamer:dev

# Clean targets
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf app/coverage.out app/coverage.html

clean-all: clean ## Clean everything including dependencies
	cd app && go clean -modcache

# Development targets
run: build ## Build and run the application
	./bin/fuse_video_streamer

debug: ## Run with debug output
	cd app && go run -race main.go

benchmark: ## Run benchmarks
	cd app && go test -bench=. -benchmem ./...

# Ensure bin directory exists
bin:
	mkdir -p bin

# Make build depend on bin directory
build: bin