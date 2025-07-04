package cmd

import (
	"fmt"
	"imap-backup/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List discovered email accounts",
	Long: `List email accounts discovered from Mac's Internet Accounts and Mail.app.
This helps you see what accounts are available for backup.`,
	RunE: runAccounts,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.Flags().BoolP("show-passwords", "p", false, "show passwords (use with caution)")
}

func runAccounts(cmd *cobra.Command, args []string) error {
	showPasswords, _ := cmd.Flags().GetBool("show-passwords")
	
	fmt.Println("Discovering email accounts from Mac's Internet Accounts and Mail.app...")
	fmt.Println()
	
	// Try to load Mac accounts
	macAccounts, err := config.LoadMacInternetAccounts()
	if err != nil {
		fmt.Printf("Error loading Mac accounts: %v\n", err)
		fmt.Println()
		fmt.Println("Trying configuration file...")
		
		// Fall back to config file
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("no accounts found in Mac accounts or config file: %w", err)
		}
		
		fmt.Printf("Found %d account(s) in configuration file:\n", len(cfg.Accounts))
		for i, account := range cfg.Accounts {
			printAccount(i+1, account, showPasswords)
		}
		return nil
	}
	
	fmt.Printf("Found %d account(s) from Mac:\n", len(macAccounts))
	for i, account := range macAccounts {
		printAccount(i+1, account, showPasswords)
	}
	
	if len(macAccounts) == 0 {
		fmt.Println("No email accounts found in Mac's Internet Accounts or Mail.app.")
		fmt.Println("You may need to:")
		fmt.Println("1. Add email accounts to Mail.app")
		fmt.Println("2. Run 'imap-backup setup' to create a configuration file")
		fmt.Println("3. Grant necessary permissions to access keychain")
	}
	
	return nil
}

func printAccount(index int, account config.Account, showPasswords bool) {
	fmt.Printf("%d. %s\n", index, account.Name)
	fmt.Printf("   Host: %s:%d\n", account.Host, account.Port)
	fmt.Printf("   Username: %s\n", account.Username)
	fmt.Printf("   SSL: %t\n", account.UseSSL)
	
	if showPasswords {
		if account.Password != "" {
			fmt.Printf("   Password: %s\n", account.Password)
		} else {
			fmt.Printf("   Password: (not found in keychain)\n")
		}
	} else {
		if account.Password != "" {
			fmt.Printf("   Password: %s\n", strings.Repeat("*", len(account.Password)))
		} else {
			fmt.Printf("   Password: (not found)\n")
		}
	}
	fmt.Println()
}