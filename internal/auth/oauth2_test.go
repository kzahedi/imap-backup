package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGetGoogleOAuth2Config(t *testing.T) {
	config := GetGoogleOAuth2Config()
	
	if config == nil {
		t.Fatal("GetGoogleOAuth2Config returned nil")
	}
	
	if len(config.Scopes) != 1 || config.Scopes[0] != "https://mail.google.com/" {
		t.Errorf("Expected scope 'https://mail.google.com/', got %v", config.Scopes)
	}
	
	if config.RedirectURL != "urn:ietf:wg:oauth:2.0:oob" {
		t.Errorf("Expected redirect URL 'urn:ietf:wg:oauth:2.0:oob', got %s", config.RedirectURL)
	}
	
	if config.Endpoint.AuthURL == "" {
		t.Error("Expected non-empty AuthURL")
	}
	
	if config.Endpoint.TokenURL == "" {
		t.Error("Expected non-empty TokenURL")
	}
}

func TestOAuth2Token_JSONMarshaling(t *testing.T) {
	originalToken := &OAuth2Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	
	// Test marshaling
	data, err := json.Marshal(originalToken)
	if err != nil {
		t.Fatalf("Failed to marshal token: %v", err)
	}
	
	// Test unmarshaling
	var parsedToken OAuth2Token
	err = json.Unmarshal(data, &parsedToken)
	if err != nil {
		t.Fatalf("Failed to unmarshal token: %v", err)
	}
	
	if parsedToken.AccessToken != originalToken.AccessToken {
		t.Errorf("AccessToken mismatch: expected %s, got %s", originalToken.AccessToken, parsedToken.AccessToken)
	}
	
	if parsedToken.RefreshToken != originalToken.RefreshToken {
		t.Errorf("RefreshToken mismatch: expected %s, got %s", originalToken.RefreshToken, parsedToken.RefreshToken)
	}
	
	if parsedToken.TokenType != originalToken.TokenType {
		t.Errorf("TokenType mismatch: expected %s, got %s", originalToken.TokenType, parsedToken.TokenType)
	}
	
	// Allow small time difference due to JSON precision
	if parsedToken.Expiry.Unix() != originalToken.Expiry.Unix() {
		t.Errorf("Expiry mismatch: expected %v, got %v", originalToken.Expiry, parsedToken.Expiry)
	}
}

func TestGenerateXOAuth2String(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		accessToken string
		expected    string
	}{
		{
			name:        "valid credentials",
			username:    "test@example.com",
			accessToken: "test-access-token",
			expected:    "user=test@example.com\x01auth=Bearer test-access-token\x01\x01",
		},
		{
			name:        "empty username",
			username:    "",
			accessToken: "test-access-token",
			expected:    "user=\x01auth=Bearer test-access-token\x01\x01",
		},
		{
			name:        "empty access token",
			username:    "test@example.com",
			accessToken: "",
			expected:    "user=test@example.com\x01auth=Bearer \x01\x01",
		},
		{
			name:        "both empty",
			username:    "",
			accessToken: "",
			expected:    "user=\x01auth=Bearer \x01\x01",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateXOAuth2String(tt.username, tt.accessToken)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    *OAuth2Token
		expected bool
	}{
		{
			name: "token expired",
			token: &OAuth2Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(-time.Hour),
			},
			expected: true,
		},
		{
			name: "token not expired",
			token: &OAuth2Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(time.Hour),
			},
			expected: false,
		},
		{
			name: "token expires soon (within 5 minutes)",
			token: &OAuth2Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(3 * time.Minute),
			},
			expected: true,
		},
		{
			name: "token expires later (more than 5 minutes)",
			token: &OAuth2Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(10 * time.Minute),
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTokenExpired(tt.token)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDetectAuthType(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "gmail.com",
			email:    "test@gmail.com",
			expected: "oauth2",
		},
		{
			name:     "googlemail.com",
			email:    "test@googlemail.com",
			expected: "oauth2",
		},
		{
			name:     "outlook.com",
			email:    "test@outlook.com",
			expected: "oauth2",
		},
		{
			name:     "hotmail.com",
			email:    "test@hotmail.com",
			expected: "oauth2",
		},
		{
			name:     "live.com",
			email:    "test@live.com",
			expected: "oauth2",
		},
		{
			name:     "yahoo.com",
			email:    "test@yahoo.com",
			expected: "oauth2",
		},
		{
			name:     "case insensitive gmail",
			email:    "test@GMAIL.COM",
			expected: "oauth2",
		},
		{
			name:     "case insensitive outlook",
			email:    "test@OUTLOOK.COM",
			expected: "oauth2",
		},
		{
			name:     "unknown provider",
			email:    "test@example.com",
			expected: "password",
		},
		{
			name:     "icloud",
			email:    "test@icloud.com",
			expected: "password",
		},
		{
			name:     "custom domain",
			email:    "test@company.com",
			expected: "password",
		},
		{
			name:     "empty email",
			email:    "",
			expected: "password",
		},
		{
			name:     "malformed email",
			email:    "not-an-email",
			expected: "password",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAuthType(tt.email)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRefreshOAuth2Token(t *testing.T) {
	tests := []struct {
		name        string
		token       *OAuth2Token
		expectError bool
	}{
		{
			name: "no refresh token",
			token: &OAuth2Token{
				AccessToken: "test-access-token",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(-time.Hour),
			},
			expectError: true,
		},
		{
			name: "with refresh token",
			token: &OAuth2Token{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
				TokenType:    "Bearer",
				Expiry:       time.Now().Add(-time.Hour),
			},
			expectError: true, // Will fail because we don't have valid OAuth2 config
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://accounts.google.com/o/oauth2/auth",
					TokenURL: "https://oauth2.googleapis.com/token",
				},
			}
			
			_, err := RefreshOAuth2Token(config, tt.token)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGetOAuth2TokenFromAccounts(t *testing.T) {
	// Test with non-existent account
	_, err := GetOAuth2TokenFromAccounts("nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent account")
	}
	
	// Test error message
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestGetOAuth2TokenFromMac(t *testing.T) {
	// Test with invalid service/account
	_, err := GetOAuth2TokenFromMac("nonexistent@example.com", "nonexistent-service")
	if err == nil {
		t.Error("Expected error for non-existent account")
	}
	
	// Test error message
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestExtractOAuth2FromPlist(t *testing.T) {
	// Create a temporary plist file for testing
	tempDir := t.TempDir()
	plistFile := filepath.Join(tempDir, "test.plist")
	
	// Create a mock plist file (binary format wouldn't work with plutil)
	// We'll test the error case since we can't easily create a proper plist
	err := os.WriteFile(plistFile, []byte("invalid plist content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test plist file: %v", err)
	}
	
	_, err = extractOAuth2FromPlist(plistFile, "test@example.com")
	if err == nil {
		t.Error("Expected error for invalid plist file")
	}
	
	// Test with non-existent file
	_, err = extractOAuth2FromPlist("/nonexistent/path/test.plist", "test@example.com")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestStartOAuth2Flow(t *testing.T) {
	// We can't test the interactive flow, but we can test the configuration
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"https://mail.google.com/"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: "urn:ietf:wg:oauth:2.0:oob",
	}
	
	// Test URL generation (this won't complete the flow)
	authURL := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	if authURL == "" {
		t.Error("Expected non-empty auth URL")
	}
	
	if !contains(authURL, "https://accounts.google.com/o/oauth2/auth") {
		t.Error("Auth URL should contain the auth endpoint")
	}
}

func TestOAuth2Provider(t *testing.T) {
	provider := OAuth2Provider{
		Name:         "Google",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"https://mail.google.com/"},
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
	}
	
	// Test that provider fields are set correctly
	if provider.Name != "Google" {
		t.Errorf("Expected Name 'Google', got %s", provider.Name)
	}
	
	if provider.ClientID != "test-client-id" {
		t.Errorf("Expected ClientID 'test-client-id', got %s", provider.ClientID)
	}
	
	if len(provider.Scopes) != 1 || provider.Scopes[0] != "https://mail.google.com/" {
		t.Errorf("Expected Scopes ['https://mail.google.com/'], got %v", provider.Scopes)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		   (len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}