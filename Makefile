# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOTOOL=$(GOCMD) tool

BINARY_NAME=linkTorch-api
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
TEST_TIMEOUT=30s
TEST_COVERAGE_FILE=coverage.out
TEST_COVERAGE_HTML=coverage.html
# Dynamic cache control - use "make test-integration nocache=false" to enable cache
COUNT_FLAG=$(if $(filter false,$(nocache)),,-count=1)

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v main.go

# Build for Linux
build-linux:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/main.go

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(BINARY_UNIX) $(TEST_COVERAGE_FILE) $(TEST_COVERAGE_HTML)

# Run all tests (unit + integration + e2e)
test: test-unit test-integration test-e2e

# Run e2e tests only
test-e2e: db-up
	@echo "Waiting for database to be ready..."
	@while ! bash -c "echo > /dev/tcp/localhost/3309" 2>/dev/null; do \
        echo "Waiting for MySQL on port 3309..."; \
        sleep 1; \
	done
	$(GOTEST) $(COUNT_FLAG) -p=1 -v -timeout $(TEST_TIMEOUT) ./tests/e2e/...

# Run e2e tests with coverage
test-e2e-coverage: db-up
	@echo "Waiting for database to be ready..."
	@while ! bash -c "echo > /dev/tcp/localhost/3309" 2>/dev/null; do \
        echo "Waiting for MySQL on port 3309..."; \
        sleep 1; \
	done
	$(GOTEST) $(COUNT_FLAG) -p=1 -v -timeout $(TEST_TIMEOUT) -coverprofile=$(TEST_COVERAGE_FILE) ./tests/e2e/...
	$(GOTOOL) cover -html=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_HTML)
	@echo "E2E test coverage report generated: $(TEST_COVERAGE_HTML)"

# Run unit tests only (internal code + tests/unit)
test-unit:
	$(GOTEST) $(COUNT_FLAG) -v -timeout $(TEST_TIMEOUT) ./tests/unit/...

# Run integration tests only
test-integration: db-up
	@echo "Waiting for database to be ready..."
	@while ! bash -c "echo > /dev/tcp/localhost/3309" 2>/dev/null; do \
        echo "Waiting for MySQL on port 3309..."; \
        sleep 1; \
	done
	$(GOTEST) $(COUNT_FLAG) -p=1 -v -timeout $(TEST_TIMEOUT) ./tests/integration/...

# Run tests with coverage
test-coverage: db-up
	@echo "Waiting for database to be ready..."
	@while ! bash -c "echo > /dev/tcp/localhost/3309" 2>/dev/null; do \
        echo "Waiting for MySQL on port 3309..."; \
        sleep 1; \
	done
	@echo "Coverage report generated: $(TEST_COVERAGE_HTML)"

# Run unit tests with coverage
test-unit-coverage:
	$(GOTEST) $(COUNT_FLAG) -v -timeout $(TEST_TIMEOUT) -coverprofile=$(TEST_COVERAGE_FILE) ./tests/unit/...
	$(GOTOOL) cover -html=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_HTML)
	@echo "Unit test coverage report generated: $(TEST_COVERAGE_HTML)"

# Run integration tests with coverage
test-integration-coverage: db-up
	@echo "Waiting for database to be ready..."
	@while ! bash -c "echo > /dev/tcp/localhost/3309" 2>/dev/null; do \
        echo "Waiting for MySQL on port 3309..."; \
        sleep 1; \
	done
	$(GOTEST) $(COUNT_FLAG) -p=1 -v -timeout $(TEST_TIMEOUT) -coverprofile=$(TEST_COVERAGE_FILE) ./tests/integration/...
	$(GOTOOL) cover -html=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_HTML)
	@echo "Integration test coverage report generated: $(TEST_COVERAGE_HTML)"

dev-basic-auth-header:
    @EMAIL=$$(grep '^DEV_USER_EMAIL=' .env | cut -d'=' -f2); \
    PASSWORD=$$(grep '^DEV_USER_PASSWORD=' .env | cut -d'=' -f2); \
    TOKEN=$$(echo -n "$$EMAIL:$$PASSWORD" | base64); \
    echo "Authorization:"; \
    echo "Basic $$TOKEN"; \


# Run linting
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Format code with gofumpt (stricter formatting)
fmt-strict:
	gofumpt -w .
	goimports -w .

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Download dependencies
deps:
	$(GOMOD) download

# Verify dependencies
verify:
	$(GOMOD) verify

# Install pre-commit hooks
install-hooks:
	pre-commit install

# Run pre-commit on all files
pre-commit-all:
	pre-commit run --all-files

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./main.go
	./$(BINARY_NAME)


# Run in development mode
dev:
	air

docker-compose-up:
	docker compose up -d

docker-compose-down:
	docker compose down

# Database commands
db-up:
	docker compose up -d mysql

db-down:
	docker compose down mysql

# Test database setup (for integration tests)
test-db-setup:
	@echo "Setting up test database..."
	@if docker ps | grep -q linkTorch-mysql; then \
        echo "Database container is running"; \
	else \
        echo "Starting database container..."; \
        docker-compose up -d mysql; \
        echo "Waiting for database to be ready..."; \
        sleep 10; \
	fi

# Benchmark tests
benchmark: db-up
	@echo "Waiting for database to be ready..."
	@while ! bash -c "echo > /dev/tcp/localhost/3309" 2>/dev/null; do \
        echo "Waiting for MySQL on port 3309..."; \
        sleep 1; \
	done
	$(GOTEST) $(COUNT_FLAG) -v -bench=. -benchmem ./tests/...

# Benchmark unit tests
benchmark-unit:
	$(GOTEST) $(COUNT_FLAG) -v -bench=. -benchmem ./tests/unit/...

# Install development tools
install-dev-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/kisielk/errcheck@latest
	go install mvdan.cc/gofumpt@latest

# Generate Swagger documentation
swagger:
	swag init -g main.go

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build               - Build the application"
	@echo "  build-linux         - Build for Linux"
	@echo "  clean               - Clean build artifacts"
	@echo ""
	@echo "Test targets:"
	@echo "  test                - Run all tests (unit + integration + e2e)"
	@echo "  test-unit           - Run unit tests only"
	@echo "  test-integration    - Run integration tests only"
	@echo "  test-e2e            - Run end-to-end tests only"
	@echo "  test-coverage       - Run tests with coverage"
	@echo "  test-unit-coverage  - Run unit tests with coverage"
	@echo "  test-integration-coverage - Run integration tests with coverage"
	@echo "  test-e2e-coverage   - Run e2e tests with coverage"
	@echo ""
	@echo "  Cache control:"
	@echo "    make test-integration              - Run without cache (default)"
	@echo "    make test-integration nocache=false  - Run with cache enabled"
	@echo ""
	@echo "Development targets:"
	@echo "  lint                - Run linting"
	@echo "  fmt                 - Format code"
	@echo "  fmt-strict          - Format code with strict formatting"
	@echo "  tidy                - Tidy dependencies"
	@echo "  deps                - Download dependencies"
	@echo "  verify              - Verify dependencies"
	@echo ""
	@echo "Git hooks:"
	@echo "  install-hooks       - Install pre-commit hooks"
	@echo "  pre-commit-all      - Run pre-commit on all files"
	@echo ""
	@echo "Run targets:"
	@echo "  run                 - Build and run the application"
	@echo "  dev                 - Run in development mode"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-compose-up   - Start with docker-compose"
	@echo "  docker-compose-down - Stop docker-compose"
	@echo ""
	@echo "Database targets:"
	@echo "  db-up               - Start database container"
	@echo "  db-down             - Stop database container"
	@echo "  test-db-setup       - Setup test database"
	@echo ""
	@echo "Performance targets:"
	@echo "  benchmark           - Run benchmark tests"
	@echo "  benchmark-unit      - Run unit benchmark tests"
	@echo ""
	@echo "Utility targets:"
	@echo "  install-dev-tools   - Install development tools"
	@echo "  swagger             - Generate Swagger documentation"
	@echo "  help                - Show this help"

.PHONY: build build-linux clean test test-unit test-integration test-e2e test-coverage \
    test-unit-coverage test-integration-coverage test-e2e-coverage lint fmt fmt-strict \
    tidy deps verify install-hooks pre-commit-all run dev docker-compose-up \
    docker-compose-down db-up db-down test-db-setup benchmark benchmark-unit \
    install-dev-tools full swagger help
