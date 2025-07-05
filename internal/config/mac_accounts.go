package config

import (
	"encoding/json"
	"fmt"
	"imap-backup/internal/providers"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MacAccount represents an email account from Mac's Internet Accounts
type MacAccount struct {
	AccountName     string `json:"account_name"`
	Username        string `json:"username"`
	EmailAddress    string `json:"email_address"`
	IMAPServer      string `json:"imap_server"`
	IMAPPort        int    `json:"imap_port"`
	IMAPUseSSL      bool   `json:"imap_use_ssl"`
	SMTPServer      string `json:"smtp_server"`
	SMTPPort        int    `json:"smtp_port"`
	SMTPUseSSL      bool   `json:"smtp_use_ssl"`
	AuthType        string `json:"auth_type"`
	KeychainService string `json:"keychain_service"`
}

// LoadMacInternetAccounts reads email accounts from Mac's Internet Accounts and Mail.app
func LoadMacInternetAccounts() ([]Account, error) {
	return loadMacInternetAccounts()
}

// loadMacInternetAccounts reads email accounts from Mac's Internet Accounts and Mail.app
func loadMacInternetAccounts() ([]Account, error) {
	var accounts []Account
	
	// Try to get accounts from Mail.app preferences
	mailAccounts, err := getMailAppAccounts()
	if err == nil {
		accounts = append(accounts, mailAccounts...)
	}
	
	// Try to get accounts from Internet Accounts
	internetAccounts, err := getInternetAccounts()
	if err == nil {
		accounts = append(accounts, internetAccounts...)
	}
	
	// Remove duplicates
	accounts = removeDuplicateAccounts(accounts)
	
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no email accounts found in Mac's Internet Accounts or Mail.app")
	}
	
	return accounts, nil
}

// getMailAppAccounts reads accounts from Mail.app preferences
func getMailAppAccounts() ([]Account, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// Look for Mail.app preferences in different versions
	mailDirs := []string{
		filepath.Join(homeDir, "Library", "Mail", "V10", "MailData"),
		filepath.Join(homeDir, "Library", "Mail", "V9", "MailData"),
		filepath.Join(homeDir, "Library", "Mail", "V8", "MailData"),
	}
	
	for _, mailDir := range mailDirs {
		accountsFile := filepath.Join(mailDir, "Accounts.plist")
		if _, err := os.Stat(accountsFile); err == nil {
			return parseMailAppAccounts(accountsFile)
		}
	}
	
	return nil, fmt.Errorf("Mail.app accounts file not found")
}

// parseMailAppAccounts parses Mail.app's Accounts.plist file
func parseMailAppAccounts(accountsFile string) ([]Account, error) {
	// Use plutil to convert plist to JSON
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", accountsFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to parse Mail.app accounts: %w", err)
	}
	
	var plistData map[string]interface{}
	if err := json.Unmarshal(output, &plistData); err != nil {
		return nil, fmt.Errorf("failed to parse plist JSON: %w", err)
	}
	
	var accounts []Account
	
	// Parse accounts from the plist structure
	if mailAccounts, ok := plistData["MailAccounts"].([]interface{}); ok {
		for _, accountData := range mailAccounts {
			if accountMap, ok := accountData.(map[string]interface{}); ok {
				account := parseMailAccount(accountMap)
				if account != nil {
					accounts = append(accounts, *account)
				}
			}
		}
	}
	
	return accounts, nil
}

// parseMailAccount parses a single Mail.app account
func parseMailAccount(accountMap map[string]interface{}) *Account {
	account := &Account{
		BaseAccount: BaseAccount{
			UseSSL: true, // Default to SSL
		},
	}
	
	// Get account name
	if displayName, ok := accountMap["DisplayName"].(string); ok {
		account.Name = displayName
	}
	
	// Get username/email
	if username, ok := accountMap["Username"].(string); ok {
		account.Username = username
	}
	
	// Get IMAP settings
	if imapSettings, ok := accountMap["IMAPAccount"].(map[string]interface{}); ok {
		if hostname, ok := imapSettings["Hostname"].(string); ok {
			account.Host = hostname
		}
		if port, ok := imapSettings["PortNumber"].(float64); ok {
			account.Port = int(port)
		}
		if useSSL, ok := imapSettings["SSLEnabled"].(bool); ok {
			account.UseSSL = useSSL
		}
	}
	
	// Set defaults if not specified
	if account.Port == 0 {
		if account.UseSSL {
			account.Port = 993
		} else {
			account.Port = 143
		}
	}
	
	// Try to get password from keychain
	if account.Host != "" && account.Username != "" {
		password, err := getPasswordFromKeychain(account.Host, account.Username)
		if err == nil {
			account.Password = password
		}
	}
	
	// Only return account if we have essential information
	if account.Host != "" && account.Username != "" {
		if account.Name == "" {
			account.Name = account.Username
		}
		return account
	}
	
	return nil
}

// getInternetAccounts reads accounts from Internet Accounts system preferences
func getInternetAccounts() ([]Account, error) {
	// Use system_profiler to get Internet Accounts information
	cmd := exec.Command("system_profiler", "SPAccountsDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Internet Accounts: %w", err)
	}
	
	var profileData map[string]interface{}
	if err := json.Unmarshal(output, &profileData); err != nil {
		return nil, fmt.Errorf("failed to parse system_profiler JSON: %w", err)
	}
	
	var accounts []Account
	
	// Parse accounts from system_profiler output
	if accountsData, ok := profileData["SPAccountsDataType"].([]interface{}); ok {
		for _, accountData := range accountsData {
			if accountMap, ok := accountData.(map[string]interface{}); ok {
				account := parseInternetAccount(accountMap)
				if account != nil {
					accounts = append(accounts, *account)
				}
			}
		}
	}
	
	return accounts, nil
}

// parseInternetAccount parses an Internet Account from system_profiler
func parseInternetAccount(accountMap map[string]interface{}) *Account {
	// Check if this is an email account
	if accountType, ok := accountMap["_name"].(string); ok {
		if !strings.Contains(strings.ToLower(accountType), "mail") &&
		   !strings.Contains(strings.ToLower(accountType), "imap") &&
		   !strings.Contains(strings.ToLower(accountType), "exchange") {
			return nil // Not an email account
		}
	}
	
	account := &Account{
		BaseAccount: BaseAccount{
			UseSSL: true, // Default to SSL
		},
	}
	
	// Extract account information
	if name, ok := accountMap["_name"].(string); ok {
		account.Name = name
	}
	
	// For Internet Accounts, we need to look up the actual IMAP settings
	// This is more complex as they're often OAuth-based
	
	return nil // Placeholder - Internet Accounts are typically OAuth-based
}

// getPasswordFromKeychain retrieves password from Mac's keychain
func getPasswordFromKeychain(server, username string) (string, error) {
	// Try different keychain item types
	keychainQueries := [][]string{
		{"find-internet-password", "-s", server, "-a", username, "-w"},
		{"find-generic-password", "-s", server, "-a", username, "-w"},
		{"find-generic-password", "-s", "Mail", "-a", username, "-w"},
	}
	
	for _, query := range keychainQueries {
		cmd := exec.Command("security", query...)
		output, err := cmd.Output()
		if err == nil {
			password := strings.TrimSpace(string(output))
			if password != "" {
				return password, nil
			}
		}
	}
	
	return "", fmt.Errorf("password not found in keychain")
}

// removeDuplicateAccounts removes duplicate accounts based on username and host
func removeDuplicateAccounts(accounts []Account) []Account {
	seen := make(map[string]bool)
	var result []Account
	
	for _, account := range accounts {
		key := fmt.Sprintf("%s@%s", account.Username, account.Host)
		if !seen[key] {
			seen[key] = true
			result = append(result, account)
		}
	}
	
	return result
}

// getCommonIMAPSettings returns common IMAP settings for popular email providers
func getCommonIMAPSettings(emailAddress string) (string, int, bool) {
	return providers.GetIMAPSettings(emailAddress)
}