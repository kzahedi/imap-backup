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

// GetOAuth2TokenFromMac attempts to get OAuth2 token from Mac's keychain
func GetOAuth2TokenFromMac(accountName, service string) (*OAuth2Token, error) {
	// Try to get OAuth2 token from Mac's keychain
	// Gmail tokens are often stored in the Internet Accounts system
	
	// First, try to get the token using security command
	cmd := exec.Command("security", "find-generic-password", 
		"-s", service, "-a", accountName, "-w")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("OAuth2 token not found in keychain: %w", err)
	}
	
	tokenData := strings.TrimSpace(string(output))
	if tokenData == "" {
		return nil, fmt.Errorf("empty OAuth2 token in keychain")
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

// GetOAuth2TokenFromAccounts attempts to get OAuth2 token from Internet Accounts
func GetOAuth2TokenFromAccounts(accountName string) (*OAuth2Token, error) {
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