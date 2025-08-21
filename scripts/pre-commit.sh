#!/bin/bash

# Pre-commit hook for CoachGPT
# This script runs before each commit to ensure code quality

set -e

echo "🔍 Running pre-commit checks..."

# Format code
echo "✨ Formatting code..."
make fmt

# Run linting
echo "🔍 Running linter..."
if ! make lint; then
    echo "❌ Linting failed. Please fix the issues and try again."
    echo "💡 Tip: Run 'make lint-fix' to auto-fix some issues."
    exit 1
fi

# Run go vet
echo "🔍 Running go vet..."
if ! make vet; then
    echo "❌ go vet failed. Please fix the issues and try again."
    exit 1
fi

# Run tests
echo "🔬 Running tests..."
if ! make test-unit; then
    echo "❌ Tests failed. Please fix the issues and try again."
    exit 1
fi

echo "✅ All pre-commit checks passed!"
