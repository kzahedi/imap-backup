package security

import (
	"strings"
	"testing"
)

func TestValidateFolderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "valid simple folder",
			input:   "INBOX",
			wantErr: false,
		},
		{
			name:    "valid folder with spaces",
			input:   "Sent Items",
			wantErr: false,
		},
		{
			name:    "valid folder with hierarchy",
			input:   "Work/Projects",
			wantErr: false,
		},
		{
			name:    "empty folder name",
			input:   "",
			wantErr: true,
			errMsg:  "folder name cannot be empty",
		},
		{
			name:    "path traversal attempt",
			input:   "../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal sequence",
		},
		{
			name:    "absolute path attempt",
			input:   "/etc/passwd",
			wantErr: true,
			errMsg:  "absolute path",
		},
		{
			name:    "windows absolute path",
			input:   "\\Windows\\System32",
			wantErr: true,
			errMsg:  "absolute path",
		},
		{
			name:    "reserved name",
			input:   "CON",
			wantErr: true,
			errMsg:  "reserved",
		},
		{
			name:    "reserved name lowercase",
			input:   "aux",
			wantErr: true,
			errMsg:  "reserved",
		},
		{
			name:    "too long folder name",
			input:   strings.Repeat("a", MaxFolderNameLength+1),
			wantErr: true,
			errMsg:  "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFolderName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateFolderName() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateFolderName() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFolderName() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal folder",
			input:    "INBOX",
			expected: "INBOX",
		},
		{
			name:     "folder with problematic chars",
			input:    "Work:Projects*?",
			expected: "Work_Projects__",
		},
		{
			name:     "folder with null byte",
			input:    "folder\x00name",
			expected: "folder_name",
		},
		{
			name:     "empty after sanitization",
			input:    "***",
			expected: "unknown",
		},
		{
			name:     "too long folder",
			input:    strings.Repeat("a", MaxFolderNameLength+10),
			expected: strings.Repeat("a", MaxFolderNameLength),
		},
		{
			name:     "folder with leading/trailing spaces",
			input:    "  folder name  ",
			expected: "folder name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFolderName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFolderName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid hostname",
			input:   "imap.gmail.com",
			wantErr: false,
		},
		{
			name:    "valid IP address",
			input:   "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "empty hostname",
			input:   "",
			wantErr: true,
		},
		{
			name:    "hostname with invalid chars",
			input:   "host;name",
			wantErr: true,
		},
		{
			name:    "too long hostname",
			input:   strings.Repeat("a", MaxHostnameLength+1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostname(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostname() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid email",
			input:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			input:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty username",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid email format",
			input:   "notanemail",
			wantErr: true,
		},
		{
			name:    "too long username",
			input:   strings.Repeat("a", MaxUsernameLength+1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAccountName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid account name",
			input:   "MyAccount",
			wantErr: false,
		},
		{
			name:    "account name with spaces",
			input:   "My Account",
			wantErr: false,
		},
		{
			name:    "empty account name",
			input:   "",
			wantErr: true,
		},
		{
			name:    "account name with dangerous chars",
			input:   "account;rm -rf /",
			wantErr: true,
		},
		{
			name:    "account name with shell injection",
			input:   "account`whoami`",
			wantErr: true,
		},
		{
			name:    "too long account name",
			input:   strings.Repeat("a", MaxAccountNameLength+1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccountName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccountName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		userPath string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid relative path",
			basePath: "/safe/base",
			userPath: "folder/subfolder",
			wantErr:  false,
		},
		{
			name:     "path traversal attempt",
			basePath: "/safe/base",
			userPath: "../../../etc/passwd",
			wantErr:  true,
			errMsg:   "escapes base directory",
		},
		{
			name:     "complex path traversal",
			basePath: "/safe/base",
			userPath: "folder/../../etc/passwd",
			wantErr:  true,
			errMsg:   "escapes base directory",
		},
		{
			name:     "absolute path attempt",
			basePath: "/safe/base",
			userPath: "/etc/passwd",
			wantErr:  true,
			errMsg:   "escapes base directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SecurePath(tt.basePath, tt.userPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SecurePath() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SecurePath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SecurePath() unexpected error = %v", err)
				}
				if result == "" {
					t.Errorf("SecurePath() returned empty result")
				}
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkValidateFolderName(b *testing.B) {
	folderName := "Work/Projects/MyProject"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateFolderName(folderName)
	}
}

func BenchmarkSanitizeFolderName(b *testing.B) {
	folderName := "Work:Projects*With?Special<Characters>"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeFolderName(folderName)
	}
}

func BenchmarkValidateHostname(b *testing.B) {
	hostname := "imap.gmail.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateHostname(hostname)
	}
}

func BenchmarkSecurePath(b *testing.B) {
	basePath := "/safe/backup/directory"
	userPath := "folder/subfolder/deep"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SecurePath(basePath, userPath)
	}
}