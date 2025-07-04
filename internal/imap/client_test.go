package imap

import (
	"context"
	"imap-backup/internal/config"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestValidateAccount(t *testing.T) {
	tests := []struct {
		name    string
		account config.Account
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid account",
			account: config.Account{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     993,
				Username: "user@example.com",
				UseSSL:   true,
			},
			wantErr: false,
		},
		{
			name: "invalid hostname",
			account: config.Account{
				Name:     "Test Account",
				Host:     "invalid;host",
				Port:     993,
				Username: "user@example.com",
				UseSSL:   true,
			},
			wantErr: true,
			errMsg:  "invalid hostname",
		},
		{
			name: "invalid username",
			account: config.Account{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     993,
				Username: "notanemail",
				UseSSL:   true,
			},
			wantErr: true,
			errMsg:  "invalid username",
		},
		{
			name: "invalid port",
			account: config.Account{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     0,
				Username: "user@example.com",
				UseSSL:   true,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "port too high",
			account: config.Account{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     70000,
				Username: "user@example.com",
				UseSSL:   true,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "empty account name",
			account: config.Account{
				Name:     "",
				Host:     "imap.example.com",
				Port:     993,
				Username: "user@example.com",
				UseSSL:   true,
			},
			wantErr: true,
			errMsg:  "account name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAccount(tt.account)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateAccount() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateAccount() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateAccount() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGetAddressString(t *testing.T) {
	tests := []struct {
		name      string
		addresses []*imap.Address
		expected  string
	}{
		{
			name:      "nil addresses",
			addresses: nil,
			expected:  "",
		},
		{
			name:      "empty addresses",
			addresses: []*imap.Address{},
			expected:  "",
		},
		{
			name: "single address with name",
			addresses: []*imap.Address{
				{
					PersonalName: "John Doe",
					MailboxName:  "john",
					HostName:     "example.com",
				},
			},
			expected: "John Doe <john@example.com>",
		},
		{
			name: "single address without name",
			addresses: []*imap.Address{
				{
					PersonalName: "",
					MailboxName:  "john",
					HostName:     "example.com",
				},
			},
			expected: "john@example.com",
		},
		{
			name: "multiple addresses",
			addresses: []*imap.Address{
				{
					PersonalName: "John Doe",
					MailboxName:  "john",
					HostName:     "example.com",
				},
				{
					PersonalName: "",
					MailboxName:  "jane",
					HostName:     "example.org",
				},
			},
			expected: "John Doe <john@example.com>, jane@example.org",
		},
		{
			name: "address with empty fields",
			addresses: []*imap.Address{
				{
					PersonalName: "John Doe",
					MailboxName:  "",
					HostName:     "example.com",
				},
			},
			expected: "",
		},
		{
			name: "mixed valid and invalid addresses",
			addresses: []*imap.Address{
				{
					PersonalName: "John Doe",
					MailboxName:  "john",
					HostName:     "example.com",
				},
				{
					PersonalName: "",
					MailboxName:  "",
					HostName:     "example.org",
				},
				{
					PersonalName: "",
					MailboxName:  "jane",
					HostName:     "example.org",
				},
			},
			expected: "John Doe <john@example.com>, jane@example.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAddressString(tt.addresses)
			if result != tt.expected {
				t.Errorf("getAddressString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Test that constants are reasonable values
	if DefaultDialTimeout <= 0 {
		t.Error("DefaultDialTimeout should be positive")
	}
	if DefaultReadTimeout <= 0 {
		t.Error("DefaultReadTimeout should be positive")
	}
	if DefaultWriteTimeout <= 0 {
		t.Error("DefaultWriteTimeout should be positive")
	}
	if MaxMessageSize <= 0 {
		t.Error("MaxMessageSize should be positive")
	}
	if MaxConcurrentMessages <= 0 {
		t.Error("MaxConcurrentMessages should be positive")
	}
	if IMAPSelectTimeout <= 0 {
		t.Error("IMAPSelectTimeout should be positive")
	}
	if IMAPFetchTimeout <= 0 {
		t.Error("IMAPFetchTimeout should be positive")
	}

	// Test reasonable timeout values
	if DefaultDialTimeout > 5*time.Minute {
		t.Error("DefaultDialTimeout seems too long")
	}
	if DefaultReadTimeout > 10*time.Minute {
		t.Error("DefaultReadTimeout seems too long")
	}
	if DefaultWriteTimeout > 10*time.Minute {
		t.Error("DefaultWriteTimeout seems too long")
	}

	// Test reasonable size limits
	if MaxMessageSize > 1024*1024*1024 {
		t.Error("MaxMessageSize seems too large (>1GB)")
	}
	if MaxMessageSize < 1024*1024 {
		t.Error("MaxMessageSize seems too small (<1MB)")
	}
}

// Test NewClient with invalid configuration (integration test)
func TestNewClientValidation(t *testing.T) {
	ctx := context.Background()
	
	invalidAccount := config.Account{
		Name:     "Test",
		Host:     "invalid;host",
		Port:     993,
		Username: "user@example.com",
		UseSSL:   true,
	}

	_, err := NewClient(ctx, invalidAccount)
	if err == nil {
		t.Error("NewClient() should fail with invalid hostname")
	}
	if !contains(err.Error(), "invalid account configuration") {
		t.Errorf("NewClient() error should mention invalid configuration, got: %v", err)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Benchmark tests
func BenchmarkGetAddressString(b *testing.B) {
	addresses := []*imap.Address{
		{
			PersonalName: "John Doe",
			MailboxName:  "john",
			HostName:     "example.com",
		},
		{
			PersonalName: "Jane Smith",
			MailboxName:  "jane",
			HostName:     "example.org",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getAddressString(addresses)
	}
}

func BenchmarkValidateAccount(b *testing.B) {
	account := config.Account{
		Name:     "Test Account",
		Host:     "imap.example.com",
		Port:     993,
		Username: "user@example.com",
		UseSSL:   true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateAccount(account)
	}
}