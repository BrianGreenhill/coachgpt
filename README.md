# CoachGPT

CoachGPT is a CLI that pulls runs from Strava and generates a detailed analysis including pace, heart rate zones, elevation, and splits - perfect for pasting into ChatGPT for coaching feedback.

## Installation

### Go Install (Recommended for Go users)
```bash
go install github.com/BrianGreenhill/coachgpt@latest
```

### Quick Install (macOS/Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/BrianGreenhill/coachgpt/main/scripts/install.sh | bash
```

### Manual Download
Download the latest binary for your platform from [Releases](https://github.com/BrianGreenhill/coachgpt/releases):

- **macOS Intel**: `coachgpt-darwin-amd64`
- **macOS Apple Silicon**: `coachgpt-darwin-arm64`  
- **Linux x86_64**: `coachgpt-linux-amd64`
- **Linux ARM64**: `coachgpt-linux-arm64`
- **Windows**: `coachgpt-windows-amd64.exe`

Make it executable and move to your PATH:
```bash
chmod +x coachgpt-*
sudo mv coachgpt-* /usr/local/bin/coachgpt
```

### Build from Source
```bash
git clone https://github.com/BrianGreenhill/coachgpt.git
cd coachgpt
make build-release
```

## Features

- 🏃‍♂️ **Multiple Data Sources**: Fetches workouts from Strava (cardio) and Hevy (strength training)
- 🔌 **Provider Architecture**: Clean, extensible provider system for adding new fitness data sources
- 📊 Analyzes heart rate zones, pace, and elevation data
- 📈 Provides detailed split and lap breakdowns
- 💾 Intelligent caching with ETag support for efficient API usage
- 🔐 Secure OAuth2 authentication with token refresh
- 📋 Generates formatted markdown ready for AI analysis

## Setup

### Quick Setup (Recommended)
```bash
coachgpt config
```
This launches an interactive setup wizard that guides you through configuring your API credentials.

### Manual Setup

#### Strava Integration
1. Create a Strava application at https://www.strava.com/settings/api
2. Set your environment variables:
   ```bash
   export STRAVA_CLIENT_ID="your_client_id"
   export STRAVA_CLIENT_SECRET="your_client_secret"
   export STRAVA_HRMAX="185"  # Your maximum heart rate
   ```

#### Hevy Integration (Optional)
1. Get your Hevy API key from your Hevy account
2. Set the environment variable:
   ```bash
   export HEVY_API_KEY="your_api_key"
   ```

### Activity Selection
Optionally, set a specific activity ID:
```bash
export STRAVA_ACTIVITY_ID="1234567890"
```

## Usage

### First Time Setup
```bash
coachgpt config  # Interactive setup wizard
```

### Fetch Latest Workout
```bash
coachgpt         # Latest Strava activity (default)
coachgpt -s      # Latest Hevy workout
```

### Get Help
```bash
coachgpt help    # Show all available commands
```

## Commands

- `coachgpt config` - Interactive setup wizard for API credentials
- `coachgpt` - Fetch latest Strava activity (default)
- `coachgpt -s` - Fetch latest Hevy workout  
- `coachgpt help` - Show help information
- `coachgpt version` - Show version information

## Testing

This project includes comprehensive tests covering all core functionality:

```bash
# Run all tests
make test

# Run unit tests  
make test-unit
```

## Development

### Code Quality

This project includes comprehensive static code analysis and formatting tools:

```bash
# Run comprehensive checks (recommended before committing)
make check

# Individual tools
make fmt        # Format code with go fmt and goimports
make vet        # Run go vet for suspicious code
make lint       # Run golangci-lint for static analysis
make lint-fix   # Auto-fix linting issues where possible
```

### Pre-commit Hook

Install the pre-commit hook to automatically run quality checks:

```bash
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

# Run with coverage report
make test-coverage

```

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
