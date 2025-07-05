package providers

import (
	"strings"
)

// AuthType represents the authentication method for a provider
type AuthType string

const (
	AuthTypePassword AuthType = "password"
	AuthTypeOAuth2   AuthType = "oauth2"
)

// ProviderConfig holds configuration for email providers
type ProviderConfig struct {
	Domain   string
	AuthType AuthType
	Host     string
	Port     int
	UseSSL   bool
}

// KnownProviders maps domains to their provider configurations
var KnownProviders = map[string]ProviderConfig{
	"gmail.com": {
		Domain:   "gmail.com",
		AuthType: AuthTypeOAuth2,
		Host:     "imap.gmail.com",
		Port:     993,
		UseSSL:   true,
	},
	"googlemail.com": {
		Domain:   "googlemail.com",
		AuthType: AuthTypeOAuth2,
		Host:     "imap.gmail.com",
		Port:     993,
		UseSSL:   true,
	},
	"outlook.com": {
		Domain:   "outlook.com",
		AuthType: AuthTypeOAuth2,
		Host:     "outlook.office365.com",
		Port:     993,
		UseSSL:   true,
	},
	"hotmail.com": {
		Domain:   "hotmail.com",
		AuthType: AuthTypeOAuth2,
		Host:     "outlook.office365.com",
		Port:     993,
		UseSSL:   true,
	},
	"live.com": {
		Domain:   "live.com",
		AuthType: AuthTypeOAuth2,
		Host:     "outlook.office365.com",
		Port:     993,
		UseSSL:   true,
	},
	"yahoo.com": {
		Domain:   "yahoo.com",
		AuthType: AuthTypeOAuth2,
		Host:     "imap.mail.yahoo.com",
		Port:     993,
		UseSSL:   true,
	},
	"icloud.com": {
		Domain:   "icloud.com",
		AuthType: AuthTypePassword,
		Host:     "imap.mail.me.com",
		Port:     993,
		UseSSL:   true,
	},
	"me.com": {
		Domain:   "me.com",
		AuthType: AuthTypePassword,
		Host:     "imap.mail.me.com",
		Port:     993,
		UseSSL:   true,
	},
	"mac.com": {
		Domain:   "mac.com",
		AuthType: AuthTypePassword,
		Host:     "imap.mail.me.com",
		Port:     993,
		UseSSL:   true,
	},
	"aol.com": {
		Domain:   "aol.com",
		AuthType: AuthTypePassword,
		Host:     "imap.aol.com",
		Port:     993,
		UseSSL:   true,
	},
}

// DetectProvider determines the provider configuration for an email address
func DetectProvider(email string) (ProviderConfig, bool) {
	email = strings.ToLower(email)
	
	// Extract domain from email
	atIndex := strings.LastIndex(email, "@")
	if atIndex == -1 {
		return ProviderConfig{AuthType: AuthTypePassword}, false
	}
	
	domain := email[atIndex+1:]
	
	// Look for exact domain match
	if config, exists := KnownProviders[domain]; exists {
		return config, true
	}
	
	// For backward compatibility, also check if domain is contained in email
	for providerDomain, config := range KnownProviders {
		if strings.Contains(email, providerDomain) {
			return config, true
		}
	}
	
	// Default to password authentication for unknown providers
	return ProviderConfig{AuthType: AuthTypePassword}, false
}

// IsOAuth2Provider checks if the given email uses OAuth2 authentication
func IsOAuth2Provider(email string) bool {
	config, _ := DetectProvider(email)
	return config.AuthType == AuthTypeOAuth2
}

// GetIMAPSettings returns IMAP configuration for a given email address
func GetIMAPSettings(email string) (host string, port int, useSSL bool) {
	config, found := DetectProvider(email)
	if !found {
		// Return common IMAP defaults
		return "", 993, true
	}
	
	return config.Host, config.Port, config.UseSSL
}

// ListOAuth2Domains returns all domains that support OAuth2
func ListOAuth2Domains() []string {
	var domains []string
	for domain, config := range KnownProviders {
		if config.AuthType == AuthTypeOAuth2 {
			domains = append(domains, domain)
		}
	}
	return domains
}