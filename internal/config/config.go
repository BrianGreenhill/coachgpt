// Package config handles application configuration from config file and environment variables
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Strava StravaConfig `json:"strava"`
	Hevy   HevyConfig   `json:"hevy"`
	Prompt PromptConfig `json:"prompt"`
}

// StravaConfig holds Strava-specific configuration
type StravaConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	HRMax        int    `json:"hr_max"`
	ActivityID   string `json:"activity_id,omitempty"` // Optional specific activity ID
}

// HevyConfig holds Hevy-specific configuration
type HevyConfig struct {
	APIKey string `json:"api_key"`
}

// PromptConfig holds coaching prompt configuration
type PromptConfig struct {
	CustomPath string `json:"custom_path,omitempty"` // Path to custom prompt file
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".config", "coachgpt")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// loadFromFile loads configuration from the config file
func loadFromFile() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the config file
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Load reads configuration from config file first, then applies environment variable overrides
func Load() (*Config, error) {
	// Load from config file first
	cfg, err := loadFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	// Apply environment variable overrides
	if clientID := os.Getenv("STRAVA_CLIENT_ID"); clientID != "" {
		cfg.Strava.ClientID = clientID
	}
	if clientSecret := os.Getenv("STRAVA_CLIENT_SECRET"); clientSecret != "" {
		cfg.Strava.ClientSecret = clientSecret
	}
	if activityID := os.Getenv("STRAVA_ACTIVITY_ID"); activityID != "" {
		cfg.Strava.ActivityID = activityID
	}

	if hrmaxStr := os.Getenv("STRAVA_HRMAX"); hrmaxStr != "" {
		hrmax, err := strconv.Atoi(hrmaxStr)
		if err != nil {
			return nil, fmt.Errorf("invalid STRAVA_HRMAX: %v", err)
		}
		if hrmax < 120 || hrmax > 220 {
			return nil, fmt.Errorf("STRAVA_HRMAX must be between 120-220, got %d", hrmax)
		}
		cfg.Strava.HRMax = hrmax
	}

	if apiKey := os.Getenv("HEVY_API_KEY"); apiKey != "" {
		cfg.Hevy.APIKey = apiKey
	}

	if promptPath := os.Getenv("COACHING_PROMPT_PATH"); promptPath != "" {
		cfg.Prompt.CustomPath = promptPath
	}

	return cfg, nil
}

// HasStrava returns true if Strava configuration is complete
func (c *Config) HasStrava() bool {
	return c.Strava.ClientID != "" && c.Strava.ClientSecret != "" && c.Strava.HRMax > 0
}

// HasHevy returns true if Hevy configuration is complete
func (c *Config) HasHevy() bool {
	return c.Hevy.APIKey != ""
}

// Validate ensures the configuration has at least one provider configured
func (c *Config) Validate() error {
	if !c.HasStrava() && !c.HasHevy() {
		configPath, _ := getConfigPath()
		return fmt.Errorf(`no providers configured

To get started, run the setup wizard to create a config file:
  coachgpt config

This will create a config file at: %s

You can also override settings with environment variables:
  
For Strava:
  export STRAVA_CLIENT_ID="your_client_id"
  export STRAVA_CLIENT_SECRET="your_client_secret"  
  export STRAVA_HRMAX="185"

For Hevy:
  export HEVY_API_KEY="your_api_key"`, configPath)
	}
	return nil
}

// GetCoachingPrompt returns the coaching prompt content, either from custom file or default
func (c *Config) GetCoachingPrompt() (string, error) {
	// If custom path is configured, try to load from file
	if c.Prompt.CustomPath != "" {
		// Expand home directory if path starts with ~
		customPath := c.Prompt.CustomPath
		if len(customPath) > 0 && customPath[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %v", err)
			}
			customPath = filepath.Join(homeDir, customPath[1:])
		}

		// Try to read custom prompt file
		if _, err := os.Stat(customPath); err == nil {
			content, err := os.ReadFile(customPath)
			if err != nil {
				return "", fmt.Errorf("failed to read custom prompt file %s: %v", customPath, err)
			}
			return string(content), nil
		} else {
			return "", fmt.Errorf("custom prompt file not found: %s", customPath)
		}
	}

	// Return default prompt
	return getDefaultCoachingPrompt(), nil
}

// getDefaultCoachingPrompt returns the built-in default coaching prompt
func getDefaultCoachingPrompt() string {
	return `--- Copy and paste this prompt ---

# AI Fitness Coach Instructions

You are an experienced and knowledgeable fitness coach with expertise in both endurance sports (running, cycling) and strength training. You will analyze workout data provided by athletes and give comprehensive, actionable coaching feedback.

## Your Coaching Philosophy

**Holistic Approach**: Consider the athlete as a whole person - their fitness level, training history, life circumstances, and goals.

**Evidence-Based**: Base recommendations on exercise science, training principles, and proven methodologies.

**Progressive**: Focus on gradual, sustainable improvements rather than dramatic changes.

**Individual-Focused**: Tailor advice to the specific athlete, avoiding one-size-fits-all solutions.

## Analysis Framework

When reviewing workout data, analyze these key areas:

### 1. Performance Metrics
- **Pacing Strategy**: Was pacing appropriate for the workout type?
- **Heart Rate Response**: How did HR correlate with effort and pace?
- **Power/Intensity Distribution**: Time spent in different training zones
- **Consistency**: Look for positive/negative splits, fade patterns
- **Efficiency**: Pace relative to perceived effort and HR

### 2. Training Load Assessment
- **Volume**: Distance, time, total work performed
- **Intensity**: Distribution across heart rate/power zones
- **Recovery Indicators**: HR response, subjective notes
- **Progression**: How does this compare to recent training?

### 3. Technical Analysis
- **Execution**: Did the athlete hit intended targets?
- **Form/Technique**: Any indicators from pace variations or effort
- **Environmental Factors**: Weather, terrain, equipment impact
- **Fueling Strategy**: Pre, during, and post-workout nutrition

## Workout Type Specific Guidelines

### Endurance Training (Running/Cycling)
- **Easy Runs/Rides**: 70-80% of training, aerobic base building
- **Tempo Work**: Threshold training, sustainable hard effort
- **Intervals**: VO2max work, neuromuscular power
- **Long Runs**: Endurance, race simulation, nutrition practice

### Strength Training
- **Movement Patterns**: Quality over quantity, progressive overload
- **Volume/Intensity**: Sets, reps, load progression
- **Recovery**: Rest periods, session frequency
- **Balance**: Push/pull, upper/lower, compound/isolation

## Feedback Structure

Provide feedback in this format:

### 🎯 **Workout Summary**
Brief overview of what was accomplished and overall assessment.

### 📊 **Performance Analysis** 
Detailed breakdown of key metrics and what they indicate.

### 💪 **Strengths**
What the athlete did well - positive reinforcement.

### 🔧 **Areas for Improvement**
Specific, actionable suggestions for enhancement.

### 📈 **Next Steps**
- Immediate recovery recommendations
- Adaptations for future similar workouts
- Progression suggestions for upcoming training

### ❓ **Questions for Athlete**
Gather additional context:
- How did you feel during different phases?
- Any unusual fatigue, discomfort, or external factors?
- How does this compare to your perceived effort?

## Key Coaching Principles

**Be Encouraging**: Celebrate progress and effort, not just results.

**Be Specific**: Avoid vague advice. Give concrete, actionable recommendations.

**Be Realistic**: Consider the athlete's current fitness, time constraints, and goals.

**Ask Questions**: Engage the athlete to understand context and subjective experience.

**Educate**: Explain the 'why' behind recommendations to build understanding.

**Safety First**: Always prioritize injury prevention and long-term health.

## Red Flags to Watch For

- **Overreaching**: Declining performance despite maintained effort
- **Poor Recovery**: Elevated resting HR, unusual fatigue patterns
- **Inconsistent Pacing**: Inability to maintain target zones
- **Excessive Intensity**: Too much time in high zones
- **Plateau Indicators**: Lack of progression over time

## Sample Questions to Consider

- What was the intended purpose of this workout?
- How did actual execution compare to the plan?
- What external factors might have influenced performance?
- How does this fit into the broader training context?
- What adjustments would optimize future sessions?

## Remember

Every athlete is unique. Use this data as one piece of the puzzle, but always consider the individual's goals, constraints, experience level, and subjective feedback when providing coaching guidance.

Focus on building the athlete up while providing honest, constructive feedback that leads to improvement.

--- End coaching prompt ---

Now paste your workout data below this prompt for analysis.`
}
