# CoachGPT Makefile

.PHONY: test test-unit test-integration test-coverage build clean help lint lint-fix fmt vet

# Default target
help:
	@echo "Available commands:"
	@echo "  test           - Run all tests (unit + integration)"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run static code analysis (linting)"
	@echo "  lint-fix       - Run linting with auto-fixes"
	@echo "  fmt            - Format code with gofmt"
	@echo "  vet            - Run go vet for suspicious code"
	@echo "  check          - Run all checks (test + lint + vet)"
	@echo "  build          - Build the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Run the application (requires env vars)"

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "🔬 Running unit tests..."
	go test -v

# Run integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	./test.sh

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -cover -v
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run static code analysis (linting)
lint:
	@echo "🔍 Running static code analysis..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

# Run linting with auto-fixes
lint-fix:
	@echo "🔧 Running linting with auto-fixes..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run --fix

# Format code
fmt:
	@echo "✨ Formatting code..."
	go fmt ./...
	goimports -w .

# Run go vet
vet:
	@echo "🔍 Running go vet..."
	go vet ./...

# Run all checks (comprehensive quality gate)
check: fmt vet lint test
	@echo "✅ All checks passed!"

# Build the application
build:
	@echo "🔨 Building application..."
	go build -o coachgpt .

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -f coachgpt coverage.out coverage.html
	@echo "🧹 Cleaning lint cache..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint cache clean; \
	fi

# Run the application (requires environment variables to be set)
run:
	@echo "🏃 Running CoachGPT..."
	@if [ -z "$$STRAVA_CLIENT_ID" ] || [ -z "$$STRAVA_CLIENT_SECRET" ] || [ -z "$$STRAVA_HRMAX" ]; then \
		echo "❌ Missing required environment variables:"; \
		echo "   STRAVA_CLIENT_ID, STRAVA_CLIENT_SECRET, STRAVA_HRMAX"; \
		echo ""; \
		echo "Set them and try again:"; \
		echo "   export STRAVA_CLIENT_ID=your_client_id"; \
		echo "   export STRAVA_CLIENT_SECRET=your_client_secret"; \
		echo "   export STRAVA_HRMAX=185"; \
		exit 1; \
	fi
	go run .

# Watch mode for continuous testing (requires air: go install github.com/cosmtrek/air@latest)
watch:
	@echo "👀 Starting watch mode..."
	@if ! command -v air > /dev/null; then \
		echo "Installing air for watch mode..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air -- test
