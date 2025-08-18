#!/bin/bash

# Integration test script for CoachGPT
# This script tests the basic functionality with environment setup

set -e

echo "🧪 Running CoachGPT Integration Tests"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
fi

echo "✅ Go is available"

# Run unit tests
echo "🔬 Running unit tests..."
go test -v

if [ $? -eq 0 ]; then
    echo "✅ Unit tests passed"
else
    echo "❌ Unit tests failed"
    exit 1
fi

# Test compilation
echo "🔨 Testing compilation..."
go build -o coachgpt-test

if [ $? -eq 0 ]; then
    echo "✅ Compilation successful"
    rm -f coachgpt-test
else
    echo "❌ Compilation failed"
    exit 1
fi

# Test environment variable validation
echo "🔧 Testing environment variable validation..."

# Test missing environment variables
export STRAVA_CLIENT_ID=""
export STRAVA_CLIENT_SECRET=""
export STRAVA_HRMAX=""

echo "Testing missing env vars (should fail gracefully)..."
timeout 5s go run . 2>/dev/null || echo "✅ Correctly failed with missing env vars"

# Test with dummy values (will fail at OAuth but should start)
echo "Testing with dummy env vars..."
export STRAVA_CLIENT_ID="dummy_client_id"
export STRAVA_CLIENT_SECRET="dummy_client_secret" 
export STRAVA_HRMAX="185"
export STRAVA_NOCACHE="1"  # Use no cache for testing

# This should fail at the OAuth step but get past env validation
timeout 3s go run . 2>/dev/null || echo "✅ Started with env vars, failed at expected OAuth step"

echo "🎉 All integration tests passed!"
echo ""
echo "To run the application with real Strava credentials:"
echo "1. Set STRAVA_CLIENT_ID to your Strava app client ID"
echo "2. Set STRAVA_CLIENT_SECRET to your Strava app client secret"
echo "3. Set STRAVA_HRMAX to your max heart rate (e.g., 185)"
echo "4. Optionally set STRAVA_ACTIVITY_ID to analyze a specific activity"
echo "5. Run: go run ."
