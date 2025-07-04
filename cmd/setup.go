package cmd

import (
	"fmt"
	"imap-backup/internal/config"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create a sample configuration file",
	Long: `Creates a sample configuration file at ~/.imap-backup.yaml
You can edit this file to add your IMAP account details.`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	if err := config.CreateSampleConfig(); err != nil {
		return fmt.Errorf("failed to create sample config: %w", err)
	}

	fmt.Println("Sample configuration file created at ~/.imap-backup.yaml")
	fmt.Println("Please edit this file to add your IMAP account details.")
	fmt.Println()
	fmt.Println("For Gmail accounts, you'll need to:")
	fmt.Println("1. Enable 2-factor authentication")
	fmt.Println("2. Generate an app-specific password")
	fmt.Println("3. Use the app password in the configuration")
	fmt.Println()
	fmt.Println("The tool will attempt to read account information from")
	fmt.Println("Mac's Internet Accounts if no config file is found.")

	return nil
}