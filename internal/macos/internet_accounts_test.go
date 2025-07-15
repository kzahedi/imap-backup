package macos

import (
	"testing"
	"time"

	"imap-backup/internal/auth"
)

func TestNewInternetAccountsService(t *testing.T) {
	service := NewInternetAccountsService()
	
	if service == nil {
		t.Error("Expected service to be created")
	}
	
	if service.accountsPath == "" {
		t.Error("Expected accountsPath to be set")
	}
}

func TestParseAccountInfo(t *testing.T) {
	service := NewInternetAccountsService()
	
	tests := []struct {
		name        string
		accountMap  map[string]interface{}
		expected    *AccountInfo
		shouldBeNil bool
	}{
		{
			name: "Gmail account",
			accountMap: map[string]interface{}{
				"_name":        "Gmail",
				"username":     "test@gmail.com",
				"email":        "test@gmail.com",
				"account_type": "Mail",
			},
			expected: &AccountInfo{
				Name:         "Gmail",
				Username:     "test@gmail.com",
				EmailAddress: "test@gmail.com",
				AccountType:  "Mail",
				Provider:     "gmail",
				Enabled:      true,
			},
		},
		{
			name: "Outlook account",
			accountMap: map[string]interface{}{
				"_name":        "Outlook",
				"username":     "test@outlook.com",
				"email":        "test@outlook.com",
				"account_type": "Mail",
			},
			expected: &AccountInfo{
				Name:         "Outlook",
				Username:     "test@outlook.com",
				EmailAddress: "test@outlook.com",
				AccountType:  "Mail",
				Provider:     "outlook",
				Enabled:      true,
			},
		},
		{
			name: "Non-email account",
			accountMap: map[string]interface{}{
				"_name":        "Calendar",
				"username":     "test@example.com",
				"account_type": "Calendar",
			},
			shouldBeNil: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseAccountInfo(tt.accountMap)
			
			if tt.shouldBeNil {
				if result != nil {
					t.Errorf("Expected nil result for %s", tt.name)
				}
				return
			}
			
			if result == nil {
				t.Errorf("Expected non-nil result for %s", tt.name)
				return
			}
			
			if result.Name != tt.expected.Name {
				t.Errorf("Expected name %s, got %s", tt.expected.Name, result.Name)
			}
			
			if result.Username != tt.expected.Username {
				t.Errorf("Expected username %s, got %s", tt.expected.Username, result.Username)
			}
			
			if result.EmailAddress != tt.expected.EmailAddress {
				t.Errorf("Expected email %s, got %s", tt.expected.EmailAddress, result.EmailAddress)
			}
			
			if result.Provider != tt.expected.Provider {
				t.Errorf("Expected provider %s, got %s", tt.expected.Provider, result.Provider)
			}
		})
	}
}

func TestIsEmailAccount(t *testing.T) {
	service := NewInternetAccountsService()
	
	tests := []struct {
		name     string
		account  AccountInfo
		expected bool
	}{
		{
			name: "Gmail account",
			account: AccountInfo{
				AccountType: "Mail",
				Provider:    "gmail",
			},
			expected: true,
		},
		{
			name: "Outlook account",
			account: AccountInfo{
				AccountType: "Mail",
				Provider:    "outlook",
			},
			expected: true,
		},
		{
			name: "IMAP account",
			account: AccountInfo{
				AccountType: "IMAP",
				Provider:    "unknown",
			},
			expected: true,
		},
		{
			name: "Calendar account",
			account: AccountInfo{
				AccountType: "Calendar",
				Provider:    "google",
			},
			expected: false,
		},
		{
			name: "Contacts account",
			account: AccountInfo{
				AccountType: "Contacts",
				Provider:    "unknown",
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isEmailAccount(tt.account)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDetermineProvider(t *testing.T) {
	service := NewInternetAccountsService()
	
	tests := []struct {
		name        string
		accountType string
		accountName string
		expected    string
	}{
		{
			name:        "Gmail account",
			accountType: "Mail",
			accountName: "Gmail",
			expected:    "gmail",
		},
		{
			name:        "Google account",
			accountType: "Mail",
			accountName: "Google",
			expected:    "gmail",
		},
		{
			name:        "Outlook account",
			accountType: "Mail",
			accountName: "Outlook",
			expected:    "outlook",
		},
		{
			name:        "Yahoo account",
			accountType: "Mail",
			accountName: "Yahoo",
			expected:    "yahoo",
		},
		{
			name:        "iCloud account",
			accountType: "Mail",
			accountName: "iCloud",
			expected:    "icloud",
		},
		{
			name:        "Exchange account",
			accountType: "Exchange",
			accountName: "Work Email",
			expected:    "exchange",
		},
		{
			name:        "Unknown account",
			accountType: "Mail",
			accountName: "Custom Mail",
			expected:    "unknown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineProvider(tt.accountType, tt.accountName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDetectIMAPSettings(t *testing.T) {
	service := NewInternetAccountsService()
	
	tests := []struct {
		name     string
		email    string
		host     string
		port     int
		useSSL   bool
	}{
		{
			name:   "Gmail",
			email:  "test@gmail.com",
			host:   "imap.gmail.com",
			port:   993,
			useSSL: true,
		},
		{
			name:   "Outlook",
			email:  "test@outlook.com",
			host:   "outlook.office365.com",
			port:   993,
			useSSL: true,
		},
		{
			name:   "Yahoo",
			email:  "test@yahoo.com",
			host:   "imap.mail.yahoo.com",
			port:   993,
			useSSL: true,
		},
		{
			name:   "iCloud",
			email:  "test@icloud.com",
			host:   "imap.mail.me.com",
			port:   993,
			useSSL: true,
		},
		{
			name:   "Unknown domain",
			email:  "test@example.com",
			host:   "imap.example.com",
			port:   993,
			useSSL: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, useSSL := service.detectIMAPSettings(tt.email)
			
			if host != tt.host {
				t.Errorf("Expected host %s, got %s", tt.host, host)
			}
			
			if port != tt.port {
				t.Errorf("Expected port %d, got %d", tt.port, port)
			}
			
			if useSSL != tt.useSSL {
				t.Errorf("Expected useSSL %v, got %v", tt.useSSL, useSSL)
			}
		})
	}
}

func TestExtractOAuth2Token(t *testing.T) {
	service := NewInternetAccountsService()
	
	tests := []struct {
		name       string
		accountMap map[string]interface{}
		expected   *auth.OAuth2Token
	}{
		{
			name: "Valid OAuth2 token",
			accountMap: map[string]interface{}{
				"authentication": map[string]interface{}{
					"type":          "oauth2",
					"access_token":  "access123",
					"refresh_token": "refresh456",
					"expiry":        "2023-12-31T23:59:59Z",
				},
			},
			expected: &auth.OAuth2Token{
				AccessToken:  "access123",
				RefreshToken: "refresh456",
				TokenType:    "Bearer",
				Expiry:       time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
			},
		},
		{
			name: "OAuth token without refresh",
			accountMap: map[string]interface{}{
				"authentication": map[string]interface{}{
					"type":         "oauth",
					"access_token": "access123",
				},
			},
			expected: &auth.OAuth2Token{
				AccessToken: "access123",
				TokenType:   "Bearer",
			},
		},
		{
			name: "Non-OAuth authentication",
			accountMap: map[string]interface{}{
				"authentication": map[string]interface{}{
					"type":     "password",
					"password": "secret123",
				},
			},
			expected: nil,
		},
		{
			name: "No authentication data",
			accountMap: map[string]interface{}{
				"name": "Test Account",
			},
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractOAuth2Token(tt.accountMap)
			
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil token, got %+v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("Expected token, got nil")
				return
			}
			
			if result.AccessToken != tt.expected.AccessToken {
				t.Errorf("Expected access token %s, got %s", tt.expected.AccessToken, result.AccessToken)
			}
			
			if result.RefreshToken != tt.expected.RefreshToken {
				t.Errorf("Expected refresh token %s, got %s", tt.expected.RefreshToken, result.RefreshToken)
			}
			
			if result.TokenType != tt.expected.TokenType {
				t.Errorf("Expected token type %s, got %s", tt.expected.TokenType, result.TokenType)
			}
			
			// Only check expiry if it was set in the test
			if !tt.expected.Expiry.IsZero() {
				if !result.Expiry.Equal(tt.expected.Expiry) {
					t.Errorf("Expected expiry %v, got %v", tt.expected.Expiry, result.Expiry)
				}
			}
		})
	}
}

func TestConvertToBackupAccount(t *testing.T) {
	service := NewInternetAccountsService()
	
	tests := []struct {
		name     string
		account  AccountInfo
		expected struct {
			authType string
			host     string
			port     int
			useSSL   bool
		}
	}{
		{
			name: "Gmail account",
			account: AccountInfo{
				Name:         "Gmail",
				Username:     "test@gmail.com",
				EmailAddress: "test@gmail.com",
				Provider:     "gmail",
			},
			expected: struct {
				authType string
				host     string
				port     int
				useSSL   bool
			}{
				authType: "oauth2",
				host:     "imap.gmail.com",
				port:     993,
				useSSL:   true,
			},
		},
		{
			name: "Outlook account",
			account: AccountInfo{
				Name:         "Outlook",
				Username:     "test@outlook.com",
				EmailAddress: "test@outlook.com",
				Provider:     "outlook",
			},
			expected: struct {
				authType string
				host     string
				port     int
				useSSL   bool
			}{
				authType: "oauth2",
				host:     "outlook.office365.com",
				port:     993,
				useSSL:   true,
			},
		},
		{
			name: "iCloud account",
			account: AccountInfo{
				Name:         "iCloud",
				Username:     "test@icloud.com",
				EmailAddress: "test@icloud.com",
				Provider:     "icloud",
			},
			expected: struct {
				authType string
				host     string
				port     int
				useSSL   bool
			}{
				authType: "password",
				host:     "imap.mail.me.com",
				port:     993,
				useSSL:   true,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ConvertToBackupAccount(tt.account)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if result.AuthType != tt.expected.authType {
				t.Errorf("Expected auth type %s, got %s", tt.expected.authType, result.AuthType)
			}
			
			if result.Host != tt.expected.host {
				t.Errorf("Expected host %s, got %s", tt.expected.host, result.Host)
			}
			
			if result.Port != tt.expected.port {
				t.Errorf("Expected port %d, got %d", tt.expected.port, result.Port)
			}
			
			if result.UseSSL != tt.expected.useSSL {
				t.Errorf("Expected useSSL %v, got %v", tt.expected.useSSL, result.UseSSL)
			}
		})
	}
}

func TestIsInternetAccountsAvailable(t *testing.T) {
	// This test depends on the system, so we just check that it doesn't panic
	available := IsInternetAccountsAvailable()
	t.Logf("Internet Accounts available: %v", available)
}

func TestGetInternetAccountsPermission(t *testing.T) {
	// This test depends on the system, so we just check that it doesn't panic
	permission, err := GetInternetAccountsPermission()
	if err != nil {
		t.Logf("Note: GetInternetAccountsPermission failed (expected on non-macOS): %v", err)
	} else {
		t.Logf("Internet Accounts permission: %v", permission)
	}
}

// Integration tests (commented out as they require actual system accounts)
/*
func TestGetAllInternetAccounts(t *testing.T) {
	service := NewInternetAccountsService()
	
	accounts, err := service.GetAllInternetAccounts()
	if err != nil {
		t.Skip("Skipping integration test: %v", err)
	}
	
	t.Logf("Found %d Internet Accounts", len(accounts))
	for _, account := range accounts {
		t.Logf("Account: %s (%s) - %s", account.Name, account.Provider, account.EmailAddress)
	}
}

func TestGetEmailAccounts(t *testing.T) {
	service := NewInternetAccountsService()
	
	accounts, err := service.GetEmailAccounts()
	if err != nil {
		t.Skip("Skipping integration test: %v", err)
	}
	
	t.Logf("Found %d email accounts", len(accounts))
	for _, account := range accounts {
		t.Logf("Email Account: %s (%s) - %s", account.Name, account.Provider, account.EmailAddress)
	}
}
*/

// Benchmark tests
func BenchmarkParseAccountInfo(b *testing.B) {
	service := NewInternetAccountsService()
	accountMap := map[string]interface{}{
		"_name":        "Gmail",
		"username":     "test@gmail.com",
		"email":        "test@gmail.com",
		"account_type": "Mail",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.parseAccountInfo(accountMap)
	}
}

func BenchmarkDetectIMAPSettings(b *testing.B) {
	service := NewInternetAccountsService()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.detectIMAPSettings("test@gmail.com")
	}
}

func BenchmarkDetermineProvider(b *testing.B) {
	service := NewInternetAccountsService()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.determineProvider("Mail", "Gmail")
	}
}