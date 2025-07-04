package cmd

import (
	"fmt"
	"imap-backup/internal/config"
	"imap-backup/internal/keychain"
	"strings"

	"github.com/spf13/cobra"
)

var addAccountCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new email account",
	Long: `Add a new email account to backup. The password will be stored securely in Mac's keychain.
Example: imap-backup account add --name "Gmail" --host imap.gmail.com --username john@gmail.com`,
	RunE: runAddAccount,
}

func init() {
	accountCmd.AddCommand(addAccountCmd)
	
	addAccountCmd.Flags().StringP("name", "n", "", "account name (required)")
	addAccountCmd.Flags().String("host", "", "IMAP server hostname (required)")
	addAccountCmd.Flags().IntP("port", "p", 993, "IMAP server port")
	addAccountCmd.Flags().StringP("username", "u", "", "username/email address (required)")
	addAccountCmd.Flags().StringP("password", "w", "", "password (will prompt if not provided)")
	addAccountCmd.Flags().BoolP("ssl", "s", true, "use SSL/TLS")
	addAccountCmd.Flags().String("auth-type", "auto", "authentication type: auto, password, oauth2")
	
	addAccountCmd.MarkFlagRequired("name")
	addAccountCmd.MarkFlagRequired("username")
	// host is auto-detected for common providers
}

func runAddAccount(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	useSSL, _ := cmd.Flags().GetBool("ssl")
	authType, _ := cmd.Flags().GetString("auth-type")
	
	// Auto-detect authentication type if not specified
	if authType == "auto" {
		authType = detectAuthTypeForEmail(username)
	}
	
	// Auto-configure for common providers
	if host == "" {
		detectedHost, detectedPort, detectedSSL := getProviderSettings(username)
		if detectedHost != "" {
			host = detectedHost
			port = detectedPort
			useSSL = detectedSSL
			fmt.Printf("Auto-detected settings for %s: %s:%d (SSL: %t)\n", 
				username, host, port, useSSL)
		} else {
			return fmt.Errorf("could not auto-detect IMAP settings for %s. Please specify --host manually", username)
		}
	}
	
	// Check if password already exists in keychain
	var passwordFromKeychain bool
	if password == "" && authType != "oauth2" {
		keychainSvc := keychain.NewKeychainService()
		existingPassword, err := keychainSvc.GetPassword(host, username)
		if err == nil && existingPassword != "" {
			password = existingPassword
			passwordFromKeychain = true
			fmt.Printf("Found existing password in keychain for %s@%s\n", username, host)
		} else {
			// Prompt for password if not found in keychain
			fmt.Print("Enter password: ")
			if _, err := fmt.Scanln(&password); err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
		}
	}
	
	// Create account store
	store, err := config.NewJSONAccountStore()
	if err != nil {
		return fmt.Errorf("failed to create account store: %w", err)
	}
	
	// Create stored account
	account := config.StoredAccount{
		Name:     name,
		Host:     host,
		Port:     port,
		Username: username,
		UseSSL:   useSSL,
		AuthType: authType,
	}
	
	// Save account to JSON
	if err := store.AddAccount(account); err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}
	
	// Store password in keychain if provided and not already there
	if password != "" && authType != "oauth2" {
		if !passwordFromKeychain {
			keychainSvc := keychain.NewKeychainService()
			if err := keychainSvc.StorePassword(host, username, password); err != nil {
				fmt.Printf("Warning: Failed to store password in keychain: %v\n", err)
				fmt.Println("Account saved, but you'll need to provide the password manually.")
			} else {
				fmt.Printf("Password stored securely in keychain.\n")
			}
		} else {
			fmt.Printf("Using existing password from keychain.\n")
		}
	}
	
	fmt.Printf("Account '%s' added successfully!\n", name)
	fmt.Printf("Configuration saved to: %s\n", store.GetConfigPath())
	
	if authType == "oauth2" {
		fmt.Println("\nNote: This account uses OAuth2 authentication.")
		fmt.Println("The tool will attempt to use existing OAuth2 tokens from your system.")
	}
	
	return nil
}

func detectAuthTypeForEmail(email string) string {
	email = strings.ToLower(email)
	
	oauthProviders := []string{
		"gmail.com", "googlemail.com",
		"outlook.com", "hotmail.com", "live.com",
		"yahoo.com",
	}
	
	for _, provider := range oauthProviders {
		if strings.Contains(email, provider) {
			return "oauth2"
		}
	}
	
	return "password"
}

func getProviderSettings(email string) (host string, port int, useSSL bool) {
	email = strings.ToLower(email)
	
	providers := map[string]struct {
		host   string
		port   int
		useSSL bool
	}{
		"gmail.com":      {"imap.gmail.com", 993, true},
		"googlemail.com": {"imap.gmail.com", 993, true},
		"outlook.com":    {"outlook.office365.com", 993, true},
		"hotmail.com":    {"outlook.office365.com", 993, true},
		"live.com":       {"outlook.office365.com", 993, true},
		"yahoo.com":      {"imap.mail.yahoo.com", 993, true},
		"icloud.com":     {"imap.mail.me.com", 993, true},
		"me.com":         {"imap.mail.me.com", 993, true},
		"mac.com":        {"imap.mail.me.com", 993, true},
		"aol.com":        {"imap.aol.com", 993, true},
	}
	
	for domain, settings := range providers {
		if strings.Contains(email, domain) {
			return settings.host, settings.port, settings.useSSL
		}
	}
	
	// Default settings
	return "", 993, true
}