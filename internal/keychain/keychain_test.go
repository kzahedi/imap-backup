package keychain

import (
	"strings"
	"testing"
)

func TestNewKeychainService(t *testing.T) {
	service := NewKeychainService()
	if service == nil {
		t.Error("NewKeychainService() returned nil")
	}
}

func TestValidationInGetPassword(t *testing.T) {
	service := NewKeychainService()

	tests := []struct {
		name     string
		server   string
		username string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid inputs",
			server:   "imap.gmail.com",
			username: "user@gmail.com",
			wantErr:  true, // Will fail in test environment (no keychain entry)
			errMsg:   "", // Don't check error message for this case
		},
		{
			name:     "invalid server with semicolon",
			server:   "invalid;server",
			username: "user@gmail.com",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
		{
			name:     "invalid username format",
			server:   "imap.gmail.com",
			username: "notanemail",
			wantErr:  true,
			errMsg:   "invalid username",
		},
		{
			name:     "empty server",
			server:   "",
			username: "user@gmail.com",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
		{
			name:     "empty username",
			server:   "imap.gmail.com",
			username: "",
			wantErr:  true,
			errMsg:   "invalid username",
		},
		{
			name:     "server with shell injection attempt",
			server:   "host;rm -rf /",
			username: "user@gmail.com",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetPassword(tt.server, tt.username)
			if !tt.wantErr {
				if err != nil {
					t.Errorf("GetPassword() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("GetPassword() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("GetPassword() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidationInStorePassword(t *testing.T) {
	service := NewKeychainService()

	tests := []struct {
		name     string
		server   string
		username string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid inputs",
			server:   "imap.gmail.com",
			username: "user@gmail.com",
			password: "secret123",
			wantErr:  false, // May succeed in some environments
			errMsg:   "", // Don't check error message for this case
		},
		{
			name:     "invalid server",
			server:   "invalid;server",
			username: "user@gmail.com",
			password: "secret123",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
		{
			name:     "invalid username",
			server:   "imap.gmail.com",
			username: "notanemail",
			password: "secret123",
			wantErr:  true,
			errMsg:   "invalid username",
		},
		{
			name:     "empty password",
			server:   "imap.gmail.com",
			username: "user@gmail.com",
			password: "",
			wantErr:  true,
			errMsg:   "password cannot be empty",
		},
		{
			name:     "server with dangerous chars",
			server:   "host`whoami`",
			username: "user@gmail.com",
			password: "secret123",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.StorePassword(tt.server, tt.username, tt.password)
			if !tt.wantErr {
				if err != nil {
					t.Errorf("StorePassword() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("StorePassword() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("StorePassword() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidationInDeletePassword(t *testing.T) {
	service := NewKeychainService()

	tests := []struct {
		name     string
		server   string
		username string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid inputs",
			server:   "imap.gmail.com",
			username: "user@gmail.com",
			wantErr:  false, // May succeed in some environments
			errMsg:   "", // Don't check error message for this case
		},
		{
			name:     "invalid server",
			server:   "invalid;server",
			username: "user@gmail.com",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
		{
			name:     "invalid username",
			server:   "imap.gmail.com",
			username: "notanemail",
			wantErr:  true,
			errMsg:   "invalid username",
		},
		{
			name:     "server with shell metacharacters",
			server:   "host$(echo hack)",
			username: "user@gmail.com",
			wantErr:  true,
			errMsg:   "invalid server name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeletePassword(tt.server, tt.username)
			if !tt.wantErr {
				if err != nil {
					t.Errorf("DeletePassword() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("DeletePassword() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DeletePassword() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// Test that validation catches common injection patterns
func TestSecurityValidation(t *testing.T) {
	service := NewKeychainService()

	dangerousInputs := []string{
		";rm -rf /",
		"host;cat /etc/passwd",
		"host`whoami`",
		"host$(echo hack)",
		"host|cat /etc/passwd",
		"host&whoami",
		"host\nwhoami",
		"host\rwhoami",
		"host\twhoami",
		"host\"hack\"",
		"host'hack'",
		"host\\hack",
		"host{hack}",
		"host[hack]",
		"host(hack)",
	}

	for _, dangerous := range dangerousInputs {
		t.Run("dangerous_server_"+dangerous, func(t *testing.T) {
			_, err := service.GetPassword(dangerous, "user@example.com")
			if err == nil {
				t.Errorf("GetPassword() should reject dangerous server input: %q", dangerous)
			}
			if !strings.Contains(err.Error(), "invalid server name") {
				t.Errorf("GetPassword() should report invalid server name for: %q, got: %v", dangerous, err)
			}
		})

		t.Run("dangerous_store_"+dangerous, func(t *testing.T) {
			err := service.StorePassword(dangerous, "user@example.com", "password")
			if err == nil {
				t.Errorf("StorePassword() should reject dangerous server input: %q", dangerous)
			}
			if !strings.Contains(err.Error(), "invalid server name") {
				t.Errorf("StorePassword() should report invalid server name for: %q, got: %v", dangerous, err)
			}
		})

		t.Run("dangerous_delete_"+dangerous, func(t *testing.T) {
			err := service.DeletePassword(dangerous, "user@example.com")
			if err == nil {
				t.Errorf("DeletePassword() should reject dangerous server input: %q", dangerous)
			}
			if !strings.Contains(err.Error(), "invalid server name") {
				t.Errorf("DeletePassword() should report invalid server name for: %q, got: %v", dangerous, err)
			}
		})
	}
}

// Test maximum length validation
func TestLengthValidation(t *testing.T) {
	service := NewKeychainService()

	// Test extremely long hostname
	longServer := strings.Repeat("a", 300)
	_, err := service.GetPassword(longServer, "user@example.com")
	if err == nil {
		t.Error("GetPassword() should reject extremely long server name")
	}

	// Test extremely long username
	longUsername := strings.Repeat("a", 400) + "@example.com"
	_, err = service.GetPassword("imap.example.com", longUsername)
	if err == nil {
		t.Error("GetPassword() should reject extremely long username")
	}
}

// Benchmark tests
func BenchmarkGetPasswordValidation(b *testing.B) {
	service := NewKeychainService()
	server := "imap.gmail.com"
	username := "user@gmail.com"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will fail at execution but we're testing validation performance
		service.GetPassword(server, username)
	}
}

func BenchmarkStorePasswordValidation(b *testing.B) {
	service := NewKeychainService()
	server := "imap.gmail.com"
	username := "user@gmail.com"
	password := "secret123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will fail at execution but we're testing validation performance
		service.StorePassword(server, username, password)
	}
}