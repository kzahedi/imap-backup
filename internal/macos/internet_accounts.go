package macos

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"imap-backup/internal/auth"
	"imap-backup/internal/config"
)

// InternetAccountsService handles integration with macOS Internet Accounts
type InternetAccountsService struct {
	accountsPath string
}

// NewInternetAccountsService creates a new Internet Accounts service
func NewInternetAccountsService() *InternetAccountsService {
	homeDir, _ := os.UserHomeDir()
	return &InternetAccountsService{
		accountsPath: filepath.Join(homeDir, "Library", "Accounts"),
	}
}

// AccountInfo represents account information from Internet Accounts
type AccountInfo struct {
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	EmailAddress string    `json:"email_address"`
	AccountType  string    `json:"account_type"`
	Provider     string    `json:"provider"`
	LastSync     time.Time `json:"last_sync"`
	Enabled      bool      `json:"enabled"`
	OAuth2Token  *auth.OAuth2Token `json:"oauth2_token,omitempty"`
}

// GetAllInternetAccounts gets all Internet Accounts using system_profiler
func (s *InternetAccountsService) GetAllInternetAccounts() ([]AccountInfo, error) {
	cmd := exec.Command("system_profiler", "SPAccountsDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Internet Accounts: %w", err)
	}

	var profileData map[string]interface{}
	if err := json.Unmarshal(output, &profileData); err != nil {
		return nil, fmt.Errorf("failed to parse system_profiler output: %w", err)
	}

	var accounts []AccountInfo
	if accountsData, ok := profileData["SPAccountsDataType"].([]interface{}); ok {
		for _, accountData := range accountsData {
			if accountMap, ok := accountData.(map[string]interface{}); ok {
				if account := s.parseAccountInfo(accountMap); account != nil {
					accounts = append(accounts, *account)
				}
			}
		}
	}

	return accounts, nil
}

// GetEmailAccounts gets only email-related Internet Accounts
func (s *InternetAccountsService) GetEmailAccounts() ([]AccountInfo, error) {
	allAccounts, err := s.GetAllInternetAccounts()
	if err != nil {
		return nil, err
	}

	var emailAccounts []AccountInfo
	for _, account := range allAccounts {
		if s.isEmailAccount(account) {
			emailAccounts = append(emailAccounts, account)
		}
	}

	return emailAccounts, nil
}

// GetAccountByEmail gets a specific account by email address
func (s *InternetAccountsService) GetAccountByEmail(email string) (*AccountInfo, error) {
	accounts, err := s.GetEmailAccounts()
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		if account.EmailAddress == email || account.Username == email {
			return &account, nil
		}
	}

	return nil, fmt.Errorf("account not found for email: %s", email)
}

// ConvertToBackupAccount converts an Internet Account to a backup account
func (s *InternetAccountsService) ConvertToBackupAccount(account AccountInfo) (*config.Account, error) {
	backupAccount := &config.Account{
		BaseAccount: config.BaseAccount{
			Name:     account.Name,
			Username: account.Username,
			UseSSL:   true, // Default to SSL
		},
	}

	// Set authentication type based on provider
	switch strings.ToLower(account.Provider) {
	case "gmail", "google":
		backupAccount.AuthType = "oauth2"
		backupAccount.Host = "imap.gmail.com"
		backupAccount.Port = 993
	case "outlook", "hotmail", "live":
		backupAccount.AuthType = "oauth2"
		backupAccount.Host = "outlook.office365.com"
		backupAccount.Port = 993
	case "yahoo":
		backupAccount.AuthType = "oauth2"
		backupAccount.Host = "imap.mail.yahoo.com"
		backupAccount.Port = 993
	case "icloud":
		backupAccount.AuthType = "password"
		backupAccount.Host = "imap.mail.me.com"
		backupAccount.Port = 993
	default:
		// For unknown providers, try to detect settings
		host, port, useSSL := s.detectIMAPSettings(account.EmailAddress)
		backupAccount.Host = host
		backupAccount.Port = port
		backupAccount.UseSSL = useSSL
		backupAccount.AuthType = "password"
	}

	// Try to get OAuth2 token if applicable
	if backupAccount.AuthType == "oauth2" {
		token, err := s.getOAuth2TokenForAccount(account)
		if err == nil && token != nil {
			// Store the token in keychain for later use
			auth.StoreOAuth2TokenInKeychain(account.Username, account.Provider, token)
		}
	}

	// Try to get password from keychain
	if backupAccount.AuthType == "password" {
		if password, err := s.getPasswordFromKeychain(account); err == nil {
			backupAccount.Password = password
		}
	}

	return backupAccount, nil
}

// RefreshOAuth2Tokens refreshes OAuth2 tokens for all accounts
func (s *InternetAccountsService) RefreshOAuth2Tokens() error {
	accounts, err := s.GetEmailAccounts()
	if err != nil {
		return err
	}

	for _, account := range accounts {
		if account.OAuth2Token != nil {
			// Check if token is expired
			if auth.IsTokenExpired(account.OAuth2Token) {
				if err := s.refreshAccountToken(account); err != nil {
					// Log error but continue with other accounts
					fmt.Printf("Failed to refresh token for %s: %v\n", account.EmailAddress, err)
				}
			}
		}
	}

	return nil
}

// SyncWithMailApp syncs Internet Accounts with Mail.app configuration
func (s *InternetAccountsService) SyncWithMailApp() error {
	// Get Internet Accounts
	internetAccounts, err := s.GetEmailAccounts()
	if err != nil {
		return err
	}

	// Get Mail.app accounts
	mailAccounts, err := s.getMailAppAccounts()
	if err != nil {
		return err
	}

	// Sync configurations
	for _, internetAccount := range internetAccounts {
		for _, mailAccount := range mailAccounts {
			if s.accountsMatch(internetAccount, mailAccount) {
				if err := s.syncAccountSettings(internetAccount, mailAccount); err != nil {
					fmt.Printf("Failed to sync account %s: %v\n", internetAccount.EmailAddress, err)
				}
			}
		}
	}

	return nil
}

// parseAccountInfo parses account information from system_profiler output
func (s *InternetAccountsService) parseAccountInfo(accountMap map[string]interface{}) *AccountInfo {
	account := &AccountInfo{
		Enabled: true, // Default to enabled
	}

	// Extract basic information
	if name, ok := accountMap["_name"].(string); ok {
		account.Name = name
	}

	if username, ok := accountMap["username"].(string); ok {
		account.Username = username
		account.EmailAddress = username // Often the same
	}

	if email, ok := accountMap["email"].(string); ok {
		account.EmailAddress = email
	}

	if accountType, ok := accountMap["account_type"].(string); ok {
		account.AccountType = accountType
	}

	// Determine provider from account type or name
	account.Provider = s.determineProvider(account.AccountType, account.Name)

	// Only return email-related accounts
	if !s.isEmailAccount(*account) {
		return nil
	}

	// Try to extract OAuth2 token if available
	if token := s.extractOAuth2Token(accountMap); token != nil {
		account.OAuth2Token = token
	}

	return account
}

// isEmailAccount checks if an account is email-related
func (s *InternetAccountsService) isEmailAccount(account AccountInfo) bool {
	emailTypes := []string{"mail", "imap", "gmail", "outlook", "yahoo", "icloud", "exchange"}
	accountType := strings.ToLower(account.AccountType)
	provider := strings.ToLower(account.Provider)

	for _, emailType := range emailTypes {
		if strings.Contains(accountType, emailType) || strings.Contains(provider, emailType) {
			return true
		}
	}

	return false
}

// determineProvider determines the email provider from account information
func (s *InternetAccountsService) determineProvider(accountType, name string) string {
	combined := strings.ToLower(accountType + " " + name)

	if strings.Contains(combined, "gmail") || strings.Contains(combined, "google") {
		return "gmail"
	}
	if strings.Contains(combined, "outlook") || strings.Contains(combined, "hotmail") || strings.Contains(combined, "live") {
		return "outlook"
	}
	if strings.Contains(combined, "yahoo") {
		return "yahoo"
	}
	if strings.Contains(combined, "icloud") {
		return "icloud"
	}
	if strings.Contains(combined, "exchange") {
		return "exchange"
	}

	return "unknown"
}

// extractOAuth2Token extracts OAuth2 token from account data
func (s *InternetAccountsService) extractOAuth2Token(accountMap map[string]interface{}) *auth.OAuth2Token {
	// Look for authentication data
	if authData, ok := accountMap["authentication"].(map[string]interface{}); ok {
		if authType, ok := authData["type"].(string); ok {
			if strings.ToLower(authType) == "oauth2" || strings.ToLower(authType) == "oauth" {
				token := &auth.OAuth2Token{
					TokenType: "Bearer",
					Expiry:    time.Now().Add(time.Hour), // Default expiry
				}

				if accessToken, ok := authData["access_token"].(string); ok {
					token.AccessToken = accessToken
				}

				if refreshToken, ok := authData["refresh_token"].(string); ok {
					token.RefreshToken = refreshToken
				}

				if expiryStr, ok := authData["expiry"].(string); ok {
					if expiry, err := time.Parse(time.RFC3339, expiryStr); err == nil {
						token.Expiry = expiry
					}
				}

				if token.AccessToken != "" {
					return token
				}
			}
		}
	}

	return nil
}

// detectIMAPSettings detects IMAP settings for unknown providers
func (s *InternetAccountsService) detectIMAPSettings(email string) (string, int, bool) {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", 0, false
	}
	domain := parts[1]

	// Common IMAP settings
	commonSettings := map[string]struct {
		host   string
		port   int
		useSSL bool
	}{
		"gmail.com":     {"imap.gmail.com", 993, true},
		"googlemail.com": {"imap.gmail.com", 993, true},
		"outlook.com":   {"outlook.office365.com", 993, true},
		"hotmail.com":   {"outlook.office365.com", 993, true},
		"live.com":      {"outlook.office365.com", 993, true},
		"yahoo.com":     {"imap.mail.yahoo.com", 993, true},
		"icloud.com":    {"imap.mail.me.com", 993, true},
		"me.com":        {"imap.mail.me.com", 993, true},
	}

	if settings, exists := commonSettings[domain]; exists {
		return settings.host, settings.port, settings.useSSL
	}

	// Try common patterns
	commonPatterns := []string{
		"imap." + domain,
		"mail." + domain,
		domain,
	}

	for _, pattern := range commonPatterns {
		// This is a simplified check - in a real implementation,
		// you might want to actually test the connection
		return pattern, 993, true
	}

	return "", 0, false
}

// getOAuth2TokenForAccount gets OAuth2 token for a specific account
func (s *InternetAccountsService) getOAuth2TokenForAccount(account AccountInfo) (*auth.OAuth2Token, error) {
	if account.OAuth2Token != nil {
		return account.OAuth2Token, nil
	}

	// Try to get token from keychain
	return auth.GetOAuth2TokenFromMac(account.Username, account.Provider)
}

// getPasswordFromKeychain gets password from keychain for an account
func (s *InternetAccountsService) getPasswordFromKeychain(account AccountInfo) (string, error) {
	// Try different service names
	serviceNames := []string{
		account.Provider,
		"Mail",
		fmt.Sprintf("imap.%s", account.Provider),
	}

	for _, serviceName := range serviceNames {
		cmd := exec.Command("security", "find-internet-password", "-s", serviceName, "-a", account.Username, "-w")
		if output, err := cmd.Output(); err == nil {
			password := strings.TrimSpace(string(output))
			if password != "" {
				return password, nil
			}
		}
	}

	return "", fmt.Errorf("password not found in keychain")
}

// refreshAccountToken refreshes OAuth2 token for an account
func (s *InternetAccountsService) refreshAccountToken(account AccountInfo) error {
	if account.OAuth2Token == nil {
		return fmt.Errorf("no OAuth2 token to refresh")
	}

	// Get OAuth2 config for the provider
	var config *oauth2.Config
	switch strings.ToLower(account.Provider) {
	case "gmail", "google":
		config = auth.GetGoogleOAuth2Config()
	default:
		return fmt.Errorf("unsupported provider for token refresh: %s", account.Provider)
	}

	// Refresh the token
	newToken, err := auth.RefreshOAuth2Token(config, account.OAuth2Token)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Store the new token in keychain
	return auth.StoreOAuth2TokenInKeychain(account.Username, account.Provider, newToken)
}

// getMailAppAccounts gets accounts from Mail.app (placeholder implementation)
func (s *InternetAccountsService) getMailAppAccounts() ([]AccountInfo, error) {
	// This would read from Mail.app configuration
	// For now, return empty slice
	return []AccountInfo{}, nil
}

// accountsMatch checks if Internet Account matches Mail.app account
func (s *InternetAccountsService) accountsMatch(internetAccount, mailAccount AccountInfo) bool {
	return internetAccount.EmailAddress == mailAccount.EmailAddress
}

// syncAccountSettings syncs settings between Internet Account and Mail.app
func (s *InternetAccountsService) syncAccountSettings(internetAccount, mailAccount AccountInfo) error {
	// This would sync settings between the two
	// For now, return nil
	return nil
}

// IsInternetAccountsAvailable checks if Internet Accounts system is available
func IsInternetAccountsAvailable() bool {
	cmd := exec.Command("system_profiler", "SPAccountsDataType", "-json")
	return cmd.Run() == nil
}

// GetInternetAccountsPermission checks if app has permission to access Internet Accounts
func GetInternetAccountsPermission() (bool, error) {
	// Try to access Internet Accounts
	cmd := exec.Command("system_profiler", "SPAccountsDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("cannot access Internet Accounts: %w", err)
	}

	// Check if we got meaningful data
	var profileData map[string]interface{}
	if err := json.Unmarshal(output, &profileData); err != nil {
		return false, fmt.Errorf("cannot parse Internet Accounts data: %w", err)
	}

	return len(profileData) > 0, nil
}