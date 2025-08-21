package providers

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BrianGreenhill/coachgpt/internal/config"
)

// SetupWizard guides users through setting up their configuration and saves to config file
func SetupWizard(registry *Registry) error {
	fmt.Println("🏃‍♂️ CoachGPT Configuration Setup")
	fmt.Println("This wizard will help you configure CoachGPT with your fitness data providers.")
	fmt.Println()

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Show current status
	showCurrentStatus(registry)

	reader := bufio.NewReader(os.Stdin)

	// Choose providers to set up
	selectedProviders := selectProviders(reader, registry)

	// Setup selected providers and update config
	for _, provider := range selectedProviders {
		if err := provider.SetupConfig(reader, cfg); err != nil {
			return fmt.Errorf("%s setup failed: %v", provider.Name(), err)
		}
	}

	// Save the updated config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Println()
	fmt.Printf("✅ Configuration saved to config file!\n")
	fmt.Println("You can now run 'coachgpt' to fetch your latest workout data.")

	return nil
}

func showCurrentStatus(registry *Registry) {
	fmt.Println("📋 Current Configuration Status:")

	allProviders := registry.All()
	if len(allProviders) == 0 {
		fmt.Println("No providers available.")
		return
	}

	for _, provider := range allProviders {
		fmt.Println(provider.ShowConfig())
	}

	fmt.Println()
}

func selectProviders(reader *bufio.Reader, registry *Registry) []Provider {
	allProviders := registry.All()
	configured := registry.Configured()
	unconfigured := registry.Unconfigured()

	// If everything is already configured, ask if they want to reconfigure
	if len(configured) == len(allProviders) && len(configured) > 0 {
		fmt.Println("All providers are already configured. Would you like to:")

		for i, provider := range allProviders {
			fmt.Printf("%d. Reconfigure %s\n", i+1, provider.Description())
		}

		if len(allProviders) > 1 {
			fmt.Printf("%d. Reconfigure all\n", len(allProviders)+1)
		}

		fmt.Printf("%d. Exit (nothing to do)\n", len(allProviders)+2)
		fmt.Printf("Enter choice (1-%d): ", len(allProviders)+2)

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		choiceNum, err := strconv.Atoi(choice)
		if err != nil || choiceNum < 1 {
			fmt.Println("Invalid choice, exiting.")
			return []Provider{}
		}

		if choiceNum <= len(allProviders) {
			return []Provider{allProviders[choiceNum-1]}
		} else if choiceNum == len(allProviders)+1 && len(allProviders) > 1 {
			return allProviders
		} else {
			fmt.Println("No changes made.")
			return []Provider{}
		}
	}

	// If nothing is configured, show all available options
	if len(configured) == 0 {
		return selectFromUnconfigured(reader, allProviders)
	}

	// Mixed state - some configured, some not
	fmt.Println("Choose what to set up:")

	options := []Provider{}
	optionNum := 1

	// Add unconfigured providers
	for _, provider := range unconfigured {
		fmt.Printf("%d. Set up %s\n", optionNum, provider.Description())
		options = append(options, provider)
		optionNum++
	}

	// Add reconfigure options for configured providers
	for _, provider := range configured {
		fmt.Printf("%d. Reconfigure %s\n", optionNum, provider.Description())
		options = append(options, provider)
		optionNum++
	}

	// Add "all unconfigured" option if there are multiple unconfigured
	allUnconfiguredIndex := -1
	if len(unconfigured) > 1 {
		fmt.Printf("%d. Set up all unconfigured providers\n", optionNum)
		allUnconfiguredIndex = optionNum - 1
		optionNum++
	}

	fmt.Printf("Enter choice (1-%d): ", optionNum-1)

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	choiceNum, err := strconv.Atoi(choice)
	if err != nil || choiceNum < 1 || choiceNum > len(options)+1 {
		fmt.Println("Invalid choice, defaulting to first option.")
		if len(options) > 0 {
			return []Provider{options[0]}
		}
		return []Provider{}
	}

	if allUnconfiguredIndex != -1 && choiceNum-1 == allUnconfiguredIndex {
		return unconfigured
	}

	if choiceNum <= len(options) {
		return []Provider{options[choiceNum-1]}
	}

	return []Provider{}
}

func selectFromUnconfigured(reader *bufio.Reader, allProviders []Provider) []Provider {
	fmt.Println("Which providers would you like to set up?")

	for i, provider := range allProviders {
		fmt.Printf("%d. %s\n", i+1, provider.Description())
	}

	if len(allProviders) > 1 {
		fmt.Printf("%d. All providers\n", len(allProviders)+1)
	}

	fmt.Printf("Enter choice (1-%d): ", len(allProviders)+1)

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	choiceNum, err := strconv.Atoi(choice)
	if err != nil || choiceNum < 1 {
		fmt.Println("Invalid choice, defaulting to first option.")
		if len(allProviders) > 0 {
			return []Provider{allProviders[0]}
		}
		return []Provider{}
	}

	if choiceNum <= len(allProviders) {
		return []Provider{allProviders[choiceNum-1]}
	} else if choiceNum == len(allProviders)+1 && len(allProviders) > 1 {
		return allProviders
	} else {
		fmt.Println("Invalid choice, defaulting to first option.")
		if len(allProviders) > 0 {
			return []Provider{allProviders[0]}
		}
		return []Provider{}
	}
}
