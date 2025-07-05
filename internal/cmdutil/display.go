package cmdutil

import (
	"fmt"
	"imap-backup/internal/config"
	"imap-backup/internal/keychain"
	"strings"
	"time"
)

// DisplayAccountInfo prints standardized account information
// This eliminates duplication of account display logic across commands
func DisplayAccountInfo(account config.StoredAccount, verbose, showPasswords bool) {
	fmt.Printf("Account: %s\n", account.Name)
	fmt.Printf("   Host: %s:%d\n", account.Host, account.Port)
	fmt.Printf("   Username: %s\n", account.Username)
	fmt.Printf("   SSL: %t\n", account.UseSSL)
	fmt.Printf("   Auth Type: %s\n", account.AuthType)
	
	if verbose {
		fmt.Printf("   ID: %s\n", account.ID)
		if !account.CreatedAt.IsZero() {
			fmt.Printf("   Created: %s\n", account.CreatedAt.Format(time.RFC3339))
		}
		if !account.UpdatedAt.IsZero() {
			fmt.Printf("   Updated: %s\n", account.UpdatedAt.Format(time.RFC3339))
		}
	}
	
	if showPasswords && account.AuthType == "password" {
		password := getPasswordForDisplay(account)
		if password != "" {
			fmt.Printf("   Password: %s\n", password)
		} else {
			fmt.Printf("   Password: (not found in keychain)\n")
		}
	}
}

// DisplayAccountList prints a list of accounts with consistent formatting
func DisplayAccountList(accounts []config.StoredAccount, verbose, showPasswords bool) {
	if len(accounts) == 0 {
		fmt.Println("No accounts configured.")
		return
	}
	
	fmt.Printf("Found %d account(s):\n\n", len(accounts))
	
	for i, account := range accounts {
		if i > 0 {
			fmt.Println()
		}
		DisplayAccountInfo(account, verbose, showPasswords)
	}
}

// DisplayDiscoveredAccounts prints discovered accounts from various sources
func DisplayDiscoveredAccounts(accounts []config.Account, source string, showPasswords bool) {
	if len(accounts) == 0 {
		return
	}
	
	fmt.Printf("=== %s ===\n", source)
	
	for _, account := range accounts {
		fmt.Printf("Account: %s\n", account.Name)
		fmt.Printf("   Host: %s:%d\n", account.Host, account.Port)
		fmt.Printf("   Username: %s\n", account.Username)
		fmt.Printf("   SSL: %t\n", account.UseSSL)
		fmt.Printf("   Auth Type: %s\n", account.AuthType)
		
		if showPasswords && account.Password != "" {
			fmt.Printf("   Password: %s\n", account.Password)
		}
		fmt.Println()
	}
}

// getPasswordForDisplay safely retrieves password for display purposes
func getPasswordForDisplay(account config.StoredAccount) string {
	if account.AuthType != "password" {
		return ""
	}
	
	keychainSvc := keychain.NewKeychainService()
	password, err := keychainSvc.GetPassword(account.Host, account.Username)
	if err != nil {
		return ""
	}
	return password
}

// MaskPassword masks a password for safe display
func MaskPassword(password string) string {
	if len(password) <= 4 {
		return strings.Repeat("*", len(password))
	}
	return password[:2] + strings.Repeat("*", len(password)-4) + password[len(password)-2:]
}

// FormatAccountSummary returns a one-line summary of an account
func FormatAccountSummary(account config.StoredAccount) string {
	return fmt.Sprintf("%s (%s@%s:%d)", account.Name, account.Username, account.Host, account.Port)
}