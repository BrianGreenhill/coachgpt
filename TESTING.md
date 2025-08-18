# CoachGPT Testing Guide

This document describes the testing setup for the CoachGPT Strava activity analyzer.

## Test Files

### `main_test.go`
Comprehensive unit tests covering:

- **Time and Pace Functions**:
  - `TestSecToHHMM`: Tests time formatting from seconds to HH:MM
  - `TestPaceFromMoving`: Tests pace calculation from distance and time

- **Data Processing Functions**:
  - `TestComputeZones`: Tests heart rate zone calculations
  - `TestComputeSplitHR`: Tests split-by-split heart rate analysis
  - `TestLapElevationFromStreams`: Tests elevation gain/loss calculations

- **Cache System**:
  - `TestCacheOperations`: Tests cache read/write operations
  - `TestAPIGETCachedMockResponse`: Tests API response caching

- **Data Structures**:
  - `TestActivityDataStructures`: Tests JSON marshaling/unmarshaling
  - `TestLapElevationFromStreams_EdgeCases`: Tests boundary conditions

### `test.sh`
Integration test script that:
- Verifies Go installation
- Runs all unit tests
- Tests compilation
- Validates environment variable handling
- Provides setup instructions

## Running Tests

### Unit Tests Only
```bash
go test -v
```

### Full Integration Tests
```bash
./test.sh
```

### Running Tests with Coverage
```bash
go test -cover -v
```

### Continuous Testing (watch mode)
```bash
# Install the air tool first if you haven't:
# go install github.com/cosmtrek/air@latest
air -- test
```

## Test Coverage

Current test coverage includes:
- ✅ Time and pace formatting functions
- ✅ Heart rate zone calculations  
- ✅ Split analysis with heart rate data
- ✅ Elevation calculations from streams
- ✅ Cache operations and ETag handling
- ✅ Data structure marshaling
- ✅ Environment variable validation
- ✅ Compilation verification

## Manual Testing

For manual testing with real Strava data:

1. Set up environment variables:
   ```bash
   export STRAVA_CLIENT_ID="your_client_id"
   export STRAVA_CLIENT_SECRET="your_client_secret"
   export STRAVA_HRMAX="185"  # Your max heart rate
   ```

2. Optional - test with specific activity:
   ```bash
   export STRAVA_ACTIVITY_ID="1234567890"
   ```

3. Run the application:
   ```bash
   go run .
   ```

## Test Data

The tests use realistic but synthetic data:
- Heart rate zones based on common training zones (70%, 80%, 88%, 95%)
- Typical running paces (4:00-6:00 min/km)
- Realistic elevation profiles
- Standard activity distances (5km, 8km runs)

## Adding New Tests

When adding new functionality:

1. Add unit tests for pure functions in `main_test.go`
2. Test both happy path and edge cases
3. Use table-driven tests for multiple inputs
4. Mock external dependencies (HTTP calls, file system)
5. Update this documentation

## Troubleshooting

Common test issues:

- **Cache tests failing**: Check file permissions in temp directory
- **Time zone issues**: Tests use UTC time consistently
- **Floating point precision**: Use appropriate delta comparisons
- **Environment isolation**: Tests clean up environment variables

## CI/CD Integration

This test suite is designed to work with common CI systems:

```yaml
# Example GitHub Actions workflow
- name: Run tests
  run: |
    go test -v -cover ./...
    ./test.sh
```
