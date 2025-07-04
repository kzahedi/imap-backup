package config

import (
	"fmt"
	"os"
	"path/filepath"

	"imap-backup/internal/keychain"
	"github.com/spf13/viper"
)

type Account struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	UseSSL   bool   `mapstructure:"use_ssl"`
	AuthType string `mapstructure:"auth_type"`
}

type Config struct {
	Accounts []Account `mapstructure:"accounts"`
}

func Load() (*Config, error) {
	var cfg Config
	
	// Try to load from JSON store first
	jsonAccounts, err := loadFromJSONStore()
	if err == nil && len(jsonAccounts) > 0 {
		cfg.Accounts = jsonAccounts
		return &cfg, nil
	}
	
	// Try to load from Mac's Internet Accounts
	macAccounts, err := loadMacInternetAccounts()
	if err == nil && len(macAccounts) > 0 {
		cfg.Accounts = macAccounts
		return &cfg, nil
	}

	// Fallback to YAML configuration file
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if len(cfg.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts configured")
	}

	return &cfg, nil
}

// loadFromJSONStore loads accounts from JSON store with keychain passwords
func loadFromJSONStore() ([]Account, error) {
	store, err := NewJSONAccountStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON store: %w", err)
	}
	
	storedAccounts, err := store.LoadAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts from JSON: %w", err)
	}
	
	if len(storedAccounts) == 0 {
		return nil, fmt.Errorf("no accounts found in JSON store")
	}
	
	keychainSvc := keychain.NewKeychainService()
	var accounts []Account
	
	for _, stored := range storedAccounts {
		var password string
		
		// Get password from keychain for password-based auth
		if stored.AuthType == "password" {
			password, err = keychainSvc.GetPassword(stored.Host, stored.Username)
			if err != nil {
				// Log warning but continue - OAuth2 accounts don't need passwords
				fmt.Printf("Warning: Could not retrieve password for %s: %v\n", stored.Username, err)
			}
		}
		
		account := store.ConvertToAccount(stored, password)
		accounts = append(accounts, account)
	}
	
	return accounts, nil
}

// loadMacInternetAccounts is now implemented in mac_accounts.go

func CreateSampleConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".imap-backup.yaml")
	
	sampleConfig := `# IMAP Backup Configuration
accounts:
  - name: "Gmail"
    host: "imap.gmail.com"
    port: 993
    username: "your-email@gmail.com"
    password: "your-app-password"
    use_ssl: true
  - name: "Outlook"
    host: "outlook.office365.com"
    port: 993
    username: "your-email@outlook.com"
    password: "your-password"
    use_ssl: true
`

	return os.WriteFile(configPath, []byte(sampleConfig), 0600)
}