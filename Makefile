# CoachGPT Makefile

.PHONY: test test-unit test-integration test-coverage build clean help

# Default target
help:
	@echo "Available commands:"
	@echo "  test           - Run all tests (unit + integration)"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-coverage  - Run tests with coverage report"
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

# Build the application
build:
	@echo "🔨 Building application..."
	go build -o coachgpt .

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -f coachgpt coverage.out coverage.html

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
