package providers

import (
	"fmt"
	"os"
	"strings"
)

// writeEnvVars writes environment variables to the user's shell profile
func writeEnvVars(vars map[string]string, providerName string) error {
	fmt.Printf("\n📝 Setting up %s environment variables...\n", providerName)

	// Detect shell and profile file
	shell := os.Getenv("SHELL")
	var profileFile string

	if strings.Contains(shell, "zsh") {
		profileFile = os.Getenv("HOME") + "/.zshrc"
	} else if strings.Contains(shell, "bash") {
		profileFile = os.Getenv("HOME") + "/.bashrc"
		// Check for .bash_profile on macOS
		if _, err := os.Stat(os.Getenv("HOME") + "/.bash_profile"); err == nil {
			profileFile = os.Getenv("HOME") + "/.bash_profile"
		}
	} else {
		profileFile = os.Getenv("HOME") + "/.profile"
	}

	fmt.Printf("Adding environment variables to: %s\n", profileFile)

	// Open file in append mode
	file, err := os.OpenFile(profileFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open profile file %s: %v", profileFile, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Write header comment
	if _, err := fmt.Fprintf(file, "\n# CoachGPT %s Configuration\n", providerName); err != nil {
		return err
	}

	// Write environment variables
	for key, value := range vars {
		line := fmt.Sprintf("export %s=\"%s\"\n", key, value)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
		fmt.Printf("✅ Added: %s\n", key)
	}

	fmt.Printf("\n⚠️  Important: Run 'source %s' or restart your terminal to load the new environment variables.\n", profileFile)

	return nil
}
