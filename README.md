# CoachGPT

CoachGPT is a CLI that pulls runs from Strava and generates a detailed analysis including pace, heart rate zones, elevation, and splits - perfect for pasting into ChatGPT for coaching feedback.

## Features

- 🏃‍♂️ Fetches your latest run or a specific activity from Strava
- 📊 Analyzes heart rate zones, pace, and elevation data
- 📈 Provides detailed split and lap breakdowns
- 💾 Intelligent caching with ETag support for efficient API usage
- 🔐 Secure OAuth2 authentication with token refresh
- 📋 Generates formatted markdown ready for AI analysis

## Setup

1. Create a Strava application at https://www.strava.com/settings/api
2. Set your environment variables:
   ```bash
   export STRAVA_CLIENT_ID="your_client_id"
   export STRAVA_CLIENT_SECRET="your_client_secret"
   export STRAVA_HRMAX="185"  # Your maximum heart rate
   ```
3. Optionally, set a specific activity ID:
   ```bash
   export STRAVA_ACTIVITY_ID="1234567890"
   ```

## Usage

```bash
# Run with latest activity
go run .

# Run with specific activity
STRAVA_ACTIVITY_ID=1234567890 go run .

# Disable caching for fresh data
STRAVA_NOCACHE=1 go run .
```

## Testing

This project includes comprehensive tests covering all core functionality:

```bash
# Run all tests
make test

# Run unit tests only  
make test-unit

# Run with coverage report
make test-coverage

# Run integration tests
make test-integration
```

See [TESTING.md](TESTING.md) for detailed testing information.

## Development

```bash
# Build the application
make build

# Clean build artifacts
make clean

# Run with environment validation
make run
```

## Output

The tool generates markdown output like this:

```markdown
## Run Log
- **Type:** [Run] Morning Run
- **When:** 2024-08-19T07:00:00Z
- **Duration:** 40:00
- **Distance:** 8.0 km (elev 150 m)
- **Avg Pace:** 5:00 / km
- **Avg HR:** 145 bpm
- **HR Zones:** Z1: 20%, Z2: 40%, Z3: 30%, Z4: 10%, Z5: 0%
- **Splits:** [detailed split table]
- **Laps:** [detailed lap table]
- **RPE:** 0-10 (0=rest, 10=max effort)
- **Fueling:** [pre + during]
- **Terrain/Weather:** []
- **Notes:** []
```

## License

MIT License
