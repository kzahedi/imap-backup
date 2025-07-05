package providers

import (
	"testing"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected ProviderConfig
		found    bool
	}{
		{
			name:  "Gmail domain",
			email: "user@gmail.com",
			expected: ProviderConfig{
				Domain:   "gmail.com",
				AuthType: AuthTypeOAuth2,
				Host:     "imap.gmail.com",
				Port:     993,
				UseSSL:   true,
			},
			found: true,
		},
		{
			name:  "Outlook domain",
			email: "user@outlook.com",
			expected: ProviderConfig{
				Domain:   "outlook.com",
				AuthType: AuthTypeOAuth2,
				Host:     "outlook.office365.com",
				Port:     993,
				UseSSL:   true,
			},
			found: true,
		},
		{
			name:  "iCloud domain",
			email: "user@icloud.com",
			expected: ProviderConfig{
				Domain:   "icloud.com",
				AuthType: AuthTypePassword,
				Host:     "imap.mail.me.com",
				Port:     993,
				UseSSL:   true,
			},
			found: true,
		},
		{
			name:  "Unknown domain",
			email: "user@unknown.com",
			expected: ProviderConfig{
				AuthType: AuthTypePassword,
			},
			found: false,
		},
		{
			name:  "Invalid email format",
			email: "invalid-email",
			expected: ProviderConfig{
				AuthType: AuthTypePassword,
			},
			found: false,
		},
		{
			name:  "Case insensitive matching",
			email: "USER@GMAIL.COM",
			expected: ProviderConfig{
				Domain:   "gmail.com",
				AuthType: AuthTypeOAuth2,
				Host:     "imap.gmail.com",
				Port:     993,
				UseSSL:   true,
			},
			found: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, found := DetectProvider(tt.email)
			
			if found != tt.found {
				t.Errorf("DetectProvider() found = %v, want %v", found, tt.found)
			}
			
			if config.AuthType != tt.expected.AuthType {
				t.Errorf("DetectProvider() AuthType = %v, want %v", config.AuthType, tt.expected.AuthType)
			}
			
			if found {
				if config.Host != tt.expected.Host {
					t.Errorf("DetectProvider() Host = %v, want %v", config.Host, tt.expected.Host)
				}
				if config.Port != tt.expected.Port {
					t.Errorf("DetectProvider() Port = %v, want %v", config.Port, tt.expected.Port)
				}
				if config.UseSSL != tt.expected.UseSSL {
					t.Errorf("DetectProvider() UseSSL = %v, want %v", config.UseSSL, tt.expected.UseSSL)
				}
			}
		})
	}
}

func TestIsOAuth2Provider(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "Gmail is OAuth2",
			email:    "user@gmail.com",
			expected: true,
		},
		{
			name:     "Outlook is OAuth2",
			email:    "user@outlook.com",
			expected: true,
		},
		{
			name:     "Yahoo is OAuth2",
			email:    "user@yahoo.com",
			expected: true,
		},
		{
			name:     "iCloud is not OAuth2",
			email:    "user@icloud.com",
			expected: false,
		},
		{
			name:     "Unknown provider defaults to password",
			email:    "user@unknown.com",
			expected: false,
		},
		{
			name:     "Case insensitive",
			email:    "USER@GMAIL.COM",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOAuth2Provider(tt.email)
			if result != tt.expected {
				t.Errorf("IsOAuth2Provider() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetIMAPSettings(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		expectedHost string
		expectedPort int
		expectedSSL  bool
	}{
		{
			name:         "Gmail settings",
			email:        "user@gmail.com",
			expectedHost: "imap.gmail.com",
			expectedPort: 993,
			expectedSSL:  true,
		},
		{
			name:         "Outlook settings",
			email:        "user@outlook.com",
			expectedHost: "outlook.office365.com",
			expectedPort: 993,
			expectedSSL:  true,
		},
		{
			name:         "Unknown provider defaults",
			email:        "user@unknown.com",
			expectedHost: "",
			expectedPort: 993,
			expectedSSL:  true,
		},
		{
			name:         "iCloud settings",
			email:        "user@icloud.com",
			expectedHost: "imap.mail.me.com",
			expectedPort: 993,
			expectedSSL:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, useSSL := GetIMAPSettings(tt.email)
			
			if host != tt.expectedHost {
				t.Errorf("GetIMAPSettings() host = %v, want %v", host, tt.expectedHost)
			}
			if port != tt.expectedPort {
				t.Errorf("GetIMAPSettings() port = %v, want %v", port, tt.expectedPort)
			}
			if useSSL != tt.expectedSSL {
				t.Errorf("GetIMAPSettings() useSSL = %v, want %v", useSSL, tt.expectedSSL)
			}
		})
	}
}

func TestListOAuth2Domains(t *testing.T) {
	domains := ListOAuth2Domains()
	
	// Check that we have the expected OAuth2 domains
	expectedDomains := []string{"gmail.com", "googlemail.com", "outlook.com", "hotmail.com", "live.com", "yahoo.com"}
	
	for _, expected := range expectedDomains {
		found := false
		for _, domain := range domains {
			if domain == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListOAuth2Domains() missing expected domain: %s", expected)
		}
	}
	
	// Check that non-OAuth2 domains are not included
	nonOAuth2Domains := []string{"icloud.com", "me.com", "mac.com", "aol.com"}
	
	for _, nonOAuth2 := range nonOAuth2Domains {
		for _, domain := range domains {
			if domain == nonOAuth2 {
				t.Errorf("ListOAuth2Domains() includes non-OAuth2 domain: %s", nonOAuth2)
			}
		}
	}
}

func BenchmarkDetectProvider(b *testing.B) {
	email := "user@gmail.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectProvider(email)
	}
}

func BenchmarkIsOAuth2Provider(b *testing.B) {
	email := "user@outlook.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsOAuth2Provider(email)
	}
}