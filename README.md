# CoachGPT

CoachGPT is a CLI tool that pulls workouts from multiple fitness platforms and generates detailed analyses perfect for AI coaching feedback. It supports both cardio activities (Strava) and strength training (Hevy) with an extensible provider architecture.

## ✨ Features

- 🏃‍♂️ **Multi-Platform Support**: Strava for cardio activities, Hevy for strength training
- ⚙️ **Intelligent Configuration**: Config file system with environment variable overrides
- 🧙‍♂️ **Interactive Setup Wizard**: Guided configuration for new users
- 🔌 **Extensible Architecture**: Clean provider system for adding new fitness platforms
- 📊 **Rich Analytics**: Heart rate zones, pace analysis, elevation profiles, split breakdowns
- 💾 **Smart Caching**: ETag support for efficient API usage
- 🔐 **Secure Authentication**: OAuth2 with automatic token refresh
- 📋 **AI-Ready Output**: Formatted markdown optimized for AI analysis

## 🚀 Installation

### Quick Install (Recommended)
```bash
# Install latest version
go install github.com/BrianGreenhill/coachgpt@latest

# Or use our install script (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/BrianGreenhill/coachgpt/main/scripts/install.sh | bash
```

### Manual Download
Download pre-built binaries from [Releases](https://github.com/BrianGreenhill/coachgpt/releases):

| Platform | Download |
|----------|----------|
| macOS Intel | `coachgpt-darwin-amd64` |
| macOS Apple Silicon | `coachgpt-darwin-arm64` |
| Linux x86_64 | `coachgpt-linux-amd64` |
| Linux ARM64 | `coachgpt-linux-arm64` |
| Windows | `coachgpt-windows-amd64.exe` |

```bash
# Make executable and install
chmod +x coachgpt-*
sudo mv coachgpt-* /usr/local/bin/coachgpt
```

### Build from Source
```bash
git clone https://github.com/BrianGreenhill/coachgpt.git
cd coachgpt
make build
```

## Setup

## ⚙️ Configuration

CoachGPT uses a config file system for persistent settings with environment variable overrides for flexibility.

### 🧙‍♂️ Interactive Setup (Recommended)

Run the setup wizard to configure your providers:

```bash
coachgpt config
```

The wizard will:
- Guide you through provider selection
- Walk you through API credential setup
- Create a secure config file at `~/.config/coachgpt/config.json`
- Handle both new setups and reconfiguration

### 📁 Config File Location

```bash
# macOS/Linux
~/.config/coachgpt/config.json

# Example config structure
{
  "strava": {
    "client_id": "your_client_id",
    "client_secret": "your_client_secret", 
    "hr_max": 185
  },
  "hevy": {
    "api_key": "your_api_key"
  }
}
```

### 🌍 Environment Variable Overrides

Environment variables will override config file settings:

```bash
# Strava Configuration
export STRAVA_CLIENT_ID="your_client_id"
export STRAVA_CLIENT_SECRET="your_client_secret"
export STRAVA_HRMAX="185"
export STRAVA_ACTIVITY_ID="1234567890"  # Optional: specific activity

# Hevy Configuration  
export HEVY_API_KEY="your_api_key"
```

### 🔧 Manual Provider Setup

#### Strava Setup
1. Create a Strava application at [https://www.strava.com/settings/api](https://www.strava.com/settings/api)
2. Note your Client ID and Client Secret
3. Run `coachgpt config` or set environment variables

#### Hevy Setup
1. Open the Hevy app → Settings → Developer
2. Copy your API key
3. Run `coachgpt config` or set `HEVY_API_KEY`

## 🎯 Usage

### First Time Setup
```bash
# Launch interactive setup wizard
coachgpt config
```

### Fetch Latest Workouts
```bash
# Latest Strava activity (default)
coachgpt

# Latest Hevy strength workout
coachgpt -s
coachgpt strength

# Show help
coachgpt help

# Show version
coachgpt version
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `coachgpt config` | Interactive setup wizard for providers |
| `coachgpt` | Fetch latest Strava activity (default) |
| `coachgpt -s` | Fetch latest Hevy workout |
| `coachgpt strength` | Fetch latest Hevy workout |
| `coachgpt help` | Show help information |
| `coachgpt version` | Show version information |

## 🧪 Testing & Development

This project includes comprehensive testing and development tools:

### Run Tests
```bash
# Run all tests (unit + integration)
go test ./...

# Run with verbose output
go test ./... -v

# Run specific test suites
go test ./internal/config -v          # Config system tests
go test ./internal/providers -v       # Provider tests + integration tests
go test ./pkg/strava -v              # Strava package tests
```

### Integration Tests
We include full integration tests for the setup workflow:
```bash
# Run integration tests specifically
go test ./internal/providers -v -run "Test.*Setup"
```

### Development Tools

#### Code Quality & Formatting
```bash
# Format code
go fmt ./...
goimports -w .

# Static analysis
golangci-lint run

# Go vet checks
go vet ./...
```

#### Build Commands
```bash
# Development build
go build .

# Cross-platform builds
make build-all

# Clean builds
make clean
```

### Pre-commit Hooks

The project includes automated quality checks via pre-commit hooks:

```bash
# Install pre-commit hook (optional)
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

This runs formatting, linting, and tests before each commit.

## 📊 Sample Output

CoachGPT generates detailed markdown output optimized for AI analysis:

### Strava Activity Output
```markdown
## Run Analysis
- **Type:** [Run] Morning Tempo Run
- **When:** 2024-08-21T07:00:00Z  
- **Duration:** 42:30
- **Distance:** 10.0 km (elev gain: 150m)
- **Avg Pace:** 4:15 /km
- **Avg HR:** 165 bpm (82% max)

### Heart Rate Zones
- **Z1 (50-60%):** 2.5 min (6%)
- **Z2 (60-70%):** 8.5 min (20%) 
- **Z3 (70-80%):** 18.7 min (44%)
- **Z4 (80-90%):** 12.3 min (29%)
- **Z5 (90-100%):** 0.5 min (1%)

### Splits (per km)
| Split | Time | Pace | HR | Elevation |
|-------|------|------|----|---------| 
| 1 | 4:45 | 4:45 /km | 145 | +12m |
| 2 | 4:20 | 4:20 /km | 158 | +8m |
| 3 | 4:10 | 4:10 /km | 167 | +15m |
...

### Workout Notes
- **RPE:** 7/10 
- **Fueling:** Pre-run banana, water during
- **Weather:** Cool, light headwind
- **Notes:** Felt strong, negative split execution
```

### Hevy Strength Output  
```markdown
## Strength Training
- **Type:** [Strength] Upper Body Push
- **When:** 2024-08-21T18:00:00Z
- **Duration:** 65 minutes
- **Exercises:** 5 exercises, 18 sets total
- **Volume:** 8,450 lbs total

### Exercise Breakdown
**Bench Press**
- Set 1: 135 lbs × 12 reps
- Set 2: 155 lbs × 10 reps  
- Set 3: 175 lbs × 8 reps
- Set 4: 185 lbs × 6 reps

**Overhead Press**
- Set 1: 95 lbs × 12 reps
- Set 2: 105 lbs × 10 reps
- Set 3: 115 lbs × 8 reps
...

### Session Notes
- **RPE:** 8/10
- **Rest Periods:** 2-3 minutes between sets
- **Notes:** Solid session, hit all target reps
```

## 🏗️ Architecture

CoachGPT uses a clean, extensible provider architecture:

```
internal/
├── config/          # Configuration management
├── providers/       # Provider interface & implementations
│   ├── strava.go   # Strava provider
│   ├── hevy.go     # Hevy provider  
│   └── wizard.go   # Interactive setup
pkg/
├── strava/         # Strava API client
└── hevy/           # Hevy API client
```

Adding new fitness platforms is straightforward via the Provider interface.

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with tests
4. Run the full test suite (`go test ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.
