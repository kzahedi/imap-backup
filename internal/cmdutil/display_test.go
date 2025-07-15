package cmdutil

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"imap-backup/internal/config"
)

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{
			name:     "empty password",
			password: "",
			expected: "",
		},
		{
			name:     "single character",
			password: "a",
			expected: "*",
		},
		{
			name:     "two characters",
			password: "ab",
			expected: "**",
		},
		{
			name:     "three characters",
			password: "abc",
			expected: "***",
		},
		{
			name:     "four characters",
			password: "abcd",
			expected: "****",
		},
		{
			name:     "five characters",
			password: "abcde",
			expected: "ab*de",
		},
		{
			name:     "six characters",
			password: "abcdef",
			expected: "ab**ef",
		},
		{
			name:     "long password",
			password: "abcdefghijklmnop",
			expected: "ab************op",
		},
		{
			name:     "password with special chars",
			password: "a!@#$%^&*()b",
			expected: "a!********)b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPassword(tt.password)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatAccountSummary(t *testing.T) {
	tests := []struct {
		name     string
		account  config.StoredAccount
		expected string
	}{
		{
			name: "complete account",
			account: config.StoredAccount{
				BaseAccount: config.BaseAccount{
					Name:     "Gmail",
					Host:     "imap.gmail.com",
					Port:     993,
					Username: "test@gmail.com",
				},
			},
			expected: "Gmail (test@gmail.com@imap.gmail.com:993)",
		},
		{
			name: "account with spaces in name",
			account: config.StoredAccount{
				BaseAccount: config.BaseAccount{
					Name:     "Work Email",
					Host:     "mail.company.com",
					Port:     143,
					Username: "john.doe@company.com",
				},
			},
			expected: "Work Email (john.doe@company.com@mail.company.com:143)",
		},
		{
			name: "account with special characters",
			account: config.StoredAccount{
				BaseAccount: config.BaseAccount{
					Name:     "Test-Account_123",
					Host:     "test.example.com",
					Port:     993,
					Username: "user+test@example.com",
				},
			},
			expected: "Test-Account_123 (user+test@example.com@test.example.com:993)",
		},
		{
			name: "empty fields",
			account: config.StoredAccount{
				BaseAccount: config.BaseAccount{
					Name:     "",
					Host:     "",
					Port:     0,
					Username: "",
				},
			},
			expected: " (@:0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAccountSummary(tt.account)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDisplayAccountInfo(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	account := config.StoredAccount{
		BaseAccount: config.BaseAccount{
			Name:     "Test Account",
			Host:     "imap.example.com",
			Port:     993,
			Username: "test@example.com",
			UseSSL:   true,
			AuthType: "password",
		},
		ID:       "test-id-123",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
	}

	// Test non-verbose output
	DisplayAccountInfo(account, false, false)

	// Restore stdout
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that basic info is displayed
	expectedSubstrings := []string{
		"Account: Test Account",
		"Host: imap.example.com:993",
		"Username: test@example.com",
		"SSL: true",
		"Auth Type: password",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("Expected output to contain %q, got:\n%s", substr, output)
		}
	}

	// Check that verbose info is NOT displayed
	if strings.Contains(output, "ID: test-id-123") {
		t.Error("ID should not be displayed in non-verbose mode")
	}

	if strings.Contains(output, "Created:") {
		t.Error("Created timestamp should not be displayed in non-verbose mode")
	}
}

func TestDisplayAccountInfo_Verbose(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	account := config.StoredAccount{
		BaseAccount: config.BaseAccount{
			Name:     "Test Account",
			Host:     "imap.example.com",
			Port:     993,
			Username: "test@example.com",
			UseSSL:   true,
			AuthType: "password",
		},
		ID:       "test-id-123",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
	}

	// Test verbose output
	DisplayAccountInfo(account, true, false)

	// Restore stdout
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that verbose info is displayed
	expectedSubstrings := []string{
		"Account: Test Account",
		"ID: test-id-123",
		"Created: 2023-01-01T12:00:00Z",
		"Updated: 2023-01-02T12:00:00Z",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("Expected output to contain %q, got:\n%s", substr, output)
		}
	}
}

func TestDisplayAccountInfo_ZeroTimestamps(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	account := config.StoredAccount{
		BaseAccount: config.BaseAccount{
			Name:     "Test Account",
			Host:     "imap.example.com",
			Port:     993,
			Username: "test@example.com",
			UseSSL:   true,
			AuthType: "password",
		},
		ID:       "test-id-123",
		// CreatedAt and UpdatedAt are zero values
	}

	// Test verbose output with zero timestamps
	DisplayAccountInfo(account, true, false)

	// Restore stdout
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that zero timestamps are not displayed
	if strings.Contains(output, "Created:") {
		t.Error("Zero CreatedAt timestamp should not be displayed")
	}

	if strings.Contains(output, "Updated:") {
		t.Error("Zero UpdatedAt timestamp should not be displayed")
	}
}

func TestDisplayAccountList(t *testing.T) {
	// Test with empty accounts
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayAccountList([]config.StoredAccount{}, false, false)

	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No accounts configured.") {
		t.Errorf("Expected 'No accounts configured.' for empty list, got:\n%s", output)
	}

	// Test with multiple accounts
	accounts := []config.StoredAccount{
		{
			BaseAccount: config.BaseAccount{
				Name:     "Gmail",
				Host:     "imap.gmail.com",
				Port:     993,
				Username: "test@gmail.com",
				UseSSL:   true,
				AuthType: "oauth2",
			},
		},
		{
			BaseAccount: config.BaseAccount{
				Name:     "Outlook",
				Host:     "outlook.office365.com",
				Port:     993,
				Username: "test@outlook.com",
				UseSSL:   true,
				AuthType: "password",
			},
		},
	}

	// Capture stdout again
	r, w, _ = os.Pipe()
	os.Stdout = w

	DisplayAccountList(accounts, false, false)

	w.Close()
	os.Stdout = old
	
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	expectedSubstrings := []string{
		"Found 2 account(s):",
		"Account: Gmail",
		"Account: Outlook",
		"imap.gmail.com:993",
		"outlook.office365.com:993",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("Expected output to contain %q, got:\n%s", substr, output)
		}
	}
}

func TestDisplayDiscoveredAccounts(t *testing.T) {
	// Test with empty accounts
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayDiscoveredAccounts([]config.Account{}, "Test Source", false)

	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should produce no output for empty accounts
	if output != "" {
		t.Errorf("Expected no output for empty accounts, got:\n%s", output)
	}

	// Test with accounts
	accounts := []config.Account{
		{
			BaseAccount: config.BaseAccount{
				Name:     "Gmail",
				Host:     "imap.gmail.com",
				Port:     993,
				Username: "test@gmail.com",
				UseSSL:   true,
				AuthType: "oauth2",
			},
			Password: "secret-password",
		},
		{
			BaseAccount: config.BaseAccount{
				Name:     "Outlook",
				Host:     "outlook.office365.com",
				Port:     993,
				Username: "test@outlook.com",
				UseSSL:   true,
				AuthType: "password",
			},
			Password: "another-secret",
		},
	}

	// Capture stdout again
	r, w, _ = os.Pipe()
	os.Stdout = w

	DisplayDiscoveredAccounts(accounts, "Internet Accounts", false)

	w.Close()
	os.Stdout = old
	
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	expectedSubstrings := []string{
		"=== Internet Accounts ===",
		"Account: Gmail",
		"Account: Outlook",
		"imap.gmail.com:993",
		"outlook.office365.com:993",
		"Auth Type: oauth2",
		"Auth Type: password",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("Expected output to contain %q, got:\n%s", substr, output)
		}
	}

	// Passwords should not be shown when showPasswords is false
	if strings.Contains(output, "secret-password") {
		t.Error("Password should not be displayed when showPasswords is false")
	}
}

func TestDisplayDiscoveredAccounts_ShowPasswords(t *testing.T) {
	accounts := []config.Account{
		{
			BaseAccount: config.BaseAccount{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     993,
				Username: "test@example.com",
				UseSSL:   true,
				AuthType: "password",
			},
			Password: "secret-password",
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayDiscoveredAccounts(accounts, "Test Source", true)

	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Password should be shown when showPasswords is true
	if !strings.Contains(output, "Password: secret-password") {
		t.Errorf("Expected password to be displayed when showPasswords is true, got:\n%s", output)
	}
}

func TestDisplayDiscoveredAccounts_NoPassword(t *testing.T) {
	accounts := []config.Account{
		{
			BaseAccount: config.BaseAccount{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     993,
				Username: "test@example.com",
				UseSSL:   true,
				AuthType: "oauth2",
			},
			Password: "", // Empty password
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayDiscoveredAccounts(accounts, "Test Source", true)

	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Password should not be shown when it's empty, even with showPasswords true
	if strings.Contains(output, "Password:") {
		t.Errorf("Expected no password line for empty password, got:\n%s", output)
	}
}

func TestGetPasswordForDisplay(t *testing.T) {
	// Test with non-password auth type
	account := config.StoredAccount{
		BaseAccount: config.BaseAccount{
			AuthType: "oauth2",
			Host:     "imap.gmail.com",
			Username: "test@gmail.com",
		},
	}

	password := getPasswordForDisplay(account)
	if password != "" {
		t.Errorf("Expected empty password for oauth2 auth type, got %q", password)
	}

	// Test with password auth type (will fail to get password from keychain)
	account.AuthType = "password"
	password = getPasswordForDisplay(account)
	// Should return empty string since keychain access will fail in test
	if password != "" {
		t.Errorf("Expected empty password when keychain access fails, got %q", password)
	}
}