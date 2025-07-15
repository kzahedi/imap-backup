package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuth2Token represents stored OAuth2 credentials
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// OAuth2Provider represents an OAuth2 provider configuration
type OAuth2Provider struct {
	Name         string
	ClientID     string
	ClientSecret string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	RedirectURL  string
}

// GetGoogleOAuth2Config returns OAuth2 configuration for Google/Gmail
func GetGoogleOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "", // Will be filled from Mac keychain or config
		ClientSecret: "", // Will be filled from Mac keychain or config
		Scopes:       []string{"https://mail.google.com/"},
		Endpoint:     google.Endpoint,
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob", // For installed applications
	}
}

// ValidateOAuth2Config validates that OAuth2 configuration has required fields
func ValidateOAuth2Config(config *oauth2.Config) error {
	if config.ClientID == "" {
		return fmt.Errorf("OAuth2 ClientID is required but not configured")
	}
	if config.ClientSecret == "" {
		return fmt.Errorf("OAuth2 ClientSecret is required but not configured")
	}
	if len(config.Scopes) == 0 {
		return fmt.Errorf("OAuth2 Scopes are required but not configured")
	}
	return nil
}

// GetOAuth2TokenFromMac attempts to get OAuth2 token from Mac's keychain
func GetOAuth2TokenFromMac(accountName, service string) (*OAuth2Token, error) {
	// Try multiple approaches to find OAuth2 tokens
	token, err := tryKeychainOAuth2Token(accountName, service)
	if err == nil {
		return token, nil
	}
	
	// Try Internet Accounts integration
	token, err = tryInternetAccountsOAuth2Token(accountName)
	if err == nil {
		return token, nil
	}
	
	// Try Mail.app OAuth2 tokens
	token, err = tryMailAppOAuth2Token(accountName)
	if err == nil {
		return token, nil
	}
	
	return nil, fmt.Errorf("OAuth2 token not found in keychain for %s", accountName)
}

// tryKeychainOAuth2Token attempts to get OAuth2 token from keychain
func tryKeychainOAuth2Token(accountName, service string) (*OAuth2Token, error) {
	// Try different keychain service names
	serviceNames := []string{
		service,
		fmt.Sprintf("Gmail OAuth2 - %s", accountName),
		fmt.Sprintf("Google OAuth2 - %s", accountName),
		fmt.Sprintf("imap-backup-oauth2-%s", service),
	}
	
	for _, serviceName := range serviceNames {
		cmd := exec.Command("security", "find-generic-password", 
			"-s", serviceName, "-a", accountName, "-w")
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		
		tokenData := strings.TrimSpace(string(output))
		if tokenData == "" {
			continue
		}
		
		// Try to parse as JSON
		var token OAuth2Token
		if err := json.Unmarshal([]byte(tokenData), &token); err != nil {
			// If not JSON, treat as simple access token
			token.AccessToken = tokenData
			token.TokenType = "Bearer"
			token.Expiry = time.Now().Add(time.Hour) // Assume 1 hour expiry
		}
		
		return &token, nil
	}
	
	return nil, fmt.Errorf("OAuth2 token not found in keychain")
}

// tryInternetAccountsOAuth2Token attempts to get OAuth2 token from Internet Accounts
func tryInternetAccountsOAuth2Token(accountName string) (*OAuth2Token, error) {
	// Use system_profiler to get Internet Accounts information
	cmd := exec.Command("system_profiler", "SPAccountsDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to access Internet Accounts: %w", err)
	}
	
	var profileData map[string]interface{}
	if err := json.Unmarshal(output, &profileData); err != nil {
		return nil, fmt.Errorf("failed to parse Internet Accounts data: %w", err)
	}
	
	// Look for OAuth2 tokens in the accounts data
	if accountsData, ok := profileData["SPAccountsDataType"].([]interface{}); ok {
		for _, accountData := range accountsData {
			if accountMap, ok := accountData.(map[string]interface{}); ok {
				if token := extractOAuth2FromAccountMap(accountMap, accountName); token != nil {
					return token, nil
				}
			}
		}
	}
	
	return nil, fmt.Errorf("OAuth2 token not found in Internet Accounts")
}

// tryMailAppOAuth2Token attempts to get OAuth2 token from Mail.app
func tryMailAppOAuth2Token(accountName string) (*OAuth2Token, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// Look for Mail.app OAuth2 tokens in different versions
	mailDirs := []string{
		filepath.Join(homeDir, "Library", "Mail", "V10", "MailData"),
		filepath.Join(homeDir, "Library", "Mail", "V9", "MailData"),
		filepath.Join(homeDir, "Library", "Mail", "V8", "MailData"),
	}
	
	for _, mailDir := range mailDirs {
		accountsFile := filepath.Join(mailDir, "Accounts.plist")
		if _, err := os.Stat(accountsFile); err == nil {
			if token := extractOAuth2FromMailApp(accountsFile, accountName); token != nil {
				return token, nil
			}
		}
	}
	
	return nil, fmt.Errorf("OAuth2 token not found in Mail.app")
}

// extractOAuth2FromAccountMap extracts OAuth2 token from Internet Accounts data
func extractOAuth2FromAccountMap(accountMap map[string]interface{}, accountName string) *OAuth2Token {
	// Check if this account matches the requested account name
	if username, ok := accountMap["username"].(string); ok {
		if username != accountName {
			return nil
		}
	}
	
	// Look for OAuth2 authentication data
	if authData, ok := accountMap["authentication"].(map[string]interface{}); ok {
		if authType, ok := authData["type"].(string); ok {
			if strings.ToLower(authType) == "oauth2" || strings.ToLower(authType) == "oauth" {
				if accessToken, ok := authData["access_token"].(string); ok {
					token := &OAuth2Token{
						AccessToken: accessToken,
						TokenType:   "Bearer",
						Expiry:      time.Now().Add(time.Hour),
					}
					
					if refreshToken, ok := authData["refresh_token"].(string); ok {
						token.RefreshToken = refreshToken
					}
					
					if expiryStr, ok := authData["expiry"].(string); ok {
						if expiry, err := time.Parse(time.RFC3339, expiryStr); err == nil {
							token.Expiry = expiry
						}
					}
					
					return token
				}
			}
		}
	}
	
	return nil
}

// extractOAuth2FromMailApp extracts OAuth2 token from Mail.app accounts
func extractOAuth2FromMailApp(accountsFile, accountName string) *OAuth2Token {
	// Use plutil to convert plist to JSON
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", accountsFile)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	
	var plistData map[string]interface{}
	if err := json.Unmarshal(output, &plistData); err != nil {
		return nil
	}
	
	// Look for OAuth2 tokens in Mail.app accounts
	if mailAccounts, ok := plistData["MailAccounts"].([]interface{}); ok {
		for _, accountData := range mailAccounts {
			if accountMap, ok := accountData.(map[string]interface{}); ok {
				if username, ok := accountMap["Username"].(string); ok {
					if username == accountName {
						// Look for OAuth2 authentication in various possible locations
						if authData, ok := accountMap["Authentication"].(map[string]interface{}); ok {
							if token := extractOAuth2FromAuthData(authData); token != nil {
								return token
							}
						}
						
						// Check IMAP account settings
						if imapSettings, ok := accountMap["IMAPAccount"].(map[string]interface{}); ok {
							if authData, ok := imapSettings["Authentication"].(map[string]interface{}); ok {
								if token := extractOAuth2FromAuthData(authData); token != nil {
									return token
								}
							}
						}
					}
				}
			}
		}
	}
	
	return nil
}

// extractOAuth2FromAuthData extracts OAuth2 token from authentication data
func extractOAuth2FromAuthData(authData map[string]interface{}) *OAuth2Token {
	if authType, ok := authData["Type"].(string); ok {
		if strings.ToLower(authType) == "oauth2" || strings.ToLower(authType) == "oauth" {
			if accessToken, ok := authData["AccessToken"].(string); ok {
				token := &OAuth2Token{
					AccessToken: accessToken,
					TokenType:   "Bearer",
					Expiry:      time.Now().Add(time.Hour),
				}
				
				if refreshToken, ok := authData["RefreshToken"].(string); ok {
					token.RefreshToken = refreshToken
				}
				
				if expiryStr, ok := authData["Expiry"].(string); ok {
					if expiry, err := time.Parse(time.RFC3339, expiryStr); err == nil {
						token.Expiry = expiry
					}
				}
				
				return token
			}
		}
	}
	
	return nil
}

// GetOAuth2TokenFromAccounts attempts to get OAuth2 token from Internet Accounts
func GetOAuth2TokenFromAccounts(accountName string) (*OAuth2Token, error) {
	// Try modern Internet Accounts approach first
	token, err := tryInternetAccountsOAuth2Token(accountName)
	if err == nil {
		return token, nil
	}
	
	// Try to extract OAuth2 token from Internet Accounts plist files
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	accountsDir := filepath.Join(homeDir, "Library", "Accounts")
	if _, err := os.Stat(accountsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Internet Accounts directory not found")
	}
	
	// Look for account plist files
	accountFiles, err := filepath.Glob(filepath.Join(accountsDir, "*.plist"))
	if err != nil {
		return nil, fmt.Errorf("failed to find account files: %w", err)
	}
	
	for _, accountFile := range accountFiles {
		token, err := extractOAuth2FromPlist(accountFile, accountName)
		if err == nil && token != nil {
			return token, nil
		}
	}
	
	return nil, fmt.Errorf("OAuth2 token not found in Internet Accounts")
}

// extractOAuth2FromPlist extracts OAuth2 token from a plist file
func extractOAuth2FromPlist(plistFile, accountName string) (*OAuth2Token, error) {
	// Use plutil to convert plist to JSON
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", plistFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read plist: %w", err)
	}
	
	var plistData map[string]interface{}
	if err := json.Unmarshal(output, &plistData); err != nil {
		return nil, fmt.Errorf("failed to parse plist JSON: %w", err)
	}
	
	// Look for OAuth2 tokens in the plist data
	// This is a simplified implementation - actual structure varies by provider
	if authData, ok := plistData["Authentication"].(map[string]interface{}); ok {
		if accessToken, ok := authData["AccessToken"].(string); ok {
			token := &OAuth2Token{
				AccessToken: accessToken,
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(time.Hour),
			}
			
			if refreshToken, ok := authData["RefreshToken"].(string); ok {
				token.RefreshToken = refreshToken
			}
			
			return token, nil
		}
	}
	
	return nil, fmt.Errorf("OAuth2 token not found in plist")
}

// RefreshOAuth2Token refreshes an expired OAuth2 token
func RefreshOAuth2Token(config *oauth2.Config, token *OAuth2Token) (*OAuth2Token, error) {
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}
	
	oauth2Token := &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}
	
	ctx := context.Background()
	tokenSource := config.TokenSource(ctx, oauth2Token)
	
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	return &OAuth2Token{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		Expiry:       newToken.Expiry,
	}, nil
}

// GenerateXOAuth2String generates XOAUTH2 string for IMAP authentication
func GenerateXOAuth2String(username, accessToken string) string {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", username, accessToken)
	return authString
}

// IsTokenExpired checks if an OAuth2 token is expired
func IsTokenExpired(token *OAuth2Token) bool {
	// Add 5 minute buffer to account for clock skew
	return time.Now().Add(5 * time.Minute).After(token.Expiry)
}

// StartOAuth2Flow initiates OAuth2 authorization flow (for manual setup)
func StartOAuth2Flow(config *oauth2.Config) (*OAuth2Token, error) {
	// Generate authorization URL
	authURL := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	
	fmt.Printf("Please visit the following URL to authorize this application:\n%s\n", authURL)
	fmt.Print("Enter the authorization code: ")
	
	var code string
	if _, err := fmt.Scanln(&code); err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}
	
	// Exchange authorization code for token
	ctx := context.Background()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	
	return &OAuth2Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}, nil
}

// StoreOAuth2TokenInKeychain stores an OAuth2 token in Mac's keychain
func StoreOAuth2TokenInKeychain(accountName, service string, token *OAuth2Token) error {
	// Convert token to JSON
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal OAuth2 token: %w", err)
	}
	
	// Create service name for keychain storage
	serviceName := fmt.Sprintf("imap-backup-oauth2-%s", service)
	
	// Store in keychain
	cmd := exec.Command("security", "add-generic-password",
		"-s", serviceName,
		"-a", accountName,
		"-w", string(tokenJSON),
		"-U", // Update if exists
		"-T", "", // Allow access by all applications
		"-j", "OAuth2 token for imap-backup", // Comment
	)
	
	if err := cmd.Run(); err != nil {
		// If update fails, try to add new entry
		cmd = exec.Command("security", "add-generic-password",
			"-s", serviceName,
			"-a", accountName,
			"-w", string(tokenJSON),
			"-T", "", // Allow access by all applications
			"-j", "OAuth2 token for imap-backup", // Comment
		)
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to store OAuth2 token in keychain: %w", err)
		}
	}
	
	return nil
}

// DeleteOAuth2TokenFromKeychain removes an OAuth2 token from Mac's keychain
func DeleteOAuth2TokenFromKeychain(accountName, service string) error {
	serviceName := fmt.Sprintf("imap-backup-oauth2-%s", service)
	
	cmd := exec.Command("security", "delete-generic-password",
		"-s", serviceName,
		"-a", accountName,
	)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete OAuth2 token from keychain: %w", err)
	}
	
	return nil
}

// ListOAuth2TokensInKeychain lists all OAuth2 tokens stored in keychain
func ListOAuth2TokensInKeychain() ([]string, error) {
	cmd := exec.Command("security", "dump-keychain", "-d")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list keychain items: %w", err)
	}
	
	var tokens []string
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "imap-backup-oauth2") {
			// Extract account name from keychain dump
			if strings.Contains(line, "acct") {
				parts := strings.Split(line, "\"")
				for i, part := range parts {
					if strings.Contains(part, "acct") && i+1 < len(parts) {
						tokens = append(tokens, parts[i+1])
						break
					}
				}
			}
		}
	}
	
	return tokens, nil
}

// DetectAuthType detects the authentication type needed for an email provider
func DetectAuthType(emailAddress string) string {
	domain := strings.ToLower(emailAddress)
	
	// Providers that typically use OAuth2
	oauthProviders := []string{
		"gmail.com",
		"googlemail.com",
		"outlook.com",
		"hotmail.com",
		"live.com",
		"yahoo.com", // Yahoo also supports OAuth2
	}
	
	for _, provider := range oauthProviders {
		if strings.Contains(domain, provider) {
			return "oauth2"
		}
	}
	
	// Default to password authentication
	return "password"
}