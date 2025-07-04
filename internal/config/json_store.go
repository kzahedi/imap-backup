package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JSONAccountStore handles JSON-based account storage
type JSONAccountStore struct {
	configPath string
}

// StoredAccount represents an account stored in JSON (without password)
type StoredAccount struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	UseSSL      bool      `json:"use_ssl"`
	AuthType    string    `json:"auth_type"` // "password", "oauth2"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastBackup  time.Time `json:"last_backup,omitempty"`
}

// AccountsConfig represents the JSON configuration structure
type AccountsConfig struct {
	Version  string          `json:"version"`
	Accounts []StoredAccount `json:"accounts"`
}

// NewJSONAccountStore creates a new JSON account store
func NewJSONAccountStore() (*JSONAccountStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configPath := filepath.Join(homeDir, ".imap-backup-accounts.json")
	
	return &JSONAccountStore{
		configPath: configPath,
	}, nil
}

// LoadAccounts loads accounts from JSON file
func (j *JSONAccountStore) LoadAccounts() ([]StoredAccount, error) {
	if _, err := os.Stat(j.configPath); os.IsNotExist(err) {
		// File doesn't exist, return empty slice
		return []StoredAccount{}, nil
	}
	
	data, err := os.ReadFile(j.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read accounts file: %w", err)
	}
	
	var config AccountsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse accounts JSON: %w", err)
	}
	
	return config.Accounts, nil
}

// SaveAccounts saves accounts to JSON file
func (j *JSONAccountStore) SaveAccounts(accounts []StoredAccount) error {
	config := AccountsConfig{
		Version:  "1.0",
		Accounts: accounts,
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal accounts JSON: %w", err)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(j.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Write with restricted permissions
	if err := os.WriteFile(j.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write accounts file: %w", err)
	}
	
	return nil
}

// AddAccount adds a new account to the JSON store
func (j *JSONAccountStore) AddAccount(account StoredAccount) error {
	accounts, err := j.LoadAccounts()
	if err != nil {
		return fmt.Errorf("failed to load existing accounts: %w", err)
	}
	
	// Generate ID if not provided
	if account.ID == "" {
		account.ID = generateAccountID(account.Username, account.Host)
	}
	
	// Set timestamps
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now
	
	// Check for duplicates
	for i, existing := range accounts {
		if existing.ID == account.ID {
			// Update existing account
			account.CreatedAt = existing.CreatedAt // Preserve creation time
			accounts[i] = account
			return j.SaveAccounts(accounts)
		}
	}
	
	// Add new account
	accounts = append(accounts, account)
	return j.SaveAccounts(accounts)
}

// GetAccount retrieves a specific account by ID
func (j *JSONAccountStore) GetAccount(id string) (*StoredAccount, error) {
	accounts, err := j.LoadAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}
	
	for _, account := range accounts {
		if account.ID == id {
			return &account, nil
		}
	}
	
	return nil, fmt.Errorf("account with ID '%s' not found", id)
}

// RemoveAccount removes an account from the JSON store
func (j *JSONAccountStore) RemoveAccount(id string) error {
	accounts, err := j.LoadAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	
	var filteredAccounts []StoredAccount
	found := false
	
	for _, account := range accounts {
		if account.ID != id {
			filteredAccounts = append(filteredAccounts, account)
		} else {
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("account with ID '%s' not found", id)
	}
	
	return j.SaveAccounts(filteredAccounts)
}

// UpdateLastBackup updates the last backup time for an account
func (j *JSONAccountStore) UpdateLastBackup(id string) error {
	accounts, err := j.LoadAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	
	for i, account := range accounts {
		if account.ID == id {
			accounts[i].LastBackup = time.Now()
			accounts[i].UpdatedAt = time.Now()
			return j.SaveAccounts(accounts)
		}
	}
	
	return fmt.Errorf("account with ID '%s' not found", id)
}

// ConvertToAccount converts a StoredAccount to the main Account type
func (j *JSONAccountStore) ConvertToAccount(stored StoredAccount, password string) Account {
	return Account{
		Name:     stored.Name,
		Host:     stored.Host,
		Port:     stored.Port,
		Username: stored.Username,
		Password: password,
		UseSSL:   stored.UseSSL,
		AuthType: stored.AuthType,
	}
}

// ConvertFromAccount converts a main Account to StoredAccount
func (j *JSONAccountStore) ConvertFromAccount(account Account) StoredAccount {
	authType := account.AuthType
	if authType == "" {
		authType = detectAuthType(account.Username)
	}
	
	return StoredAccount{
		ID:       generateAccountID(account.Username, account.Host),
		Name:     account.Name,
		Host:     account.Host,
		Port:     account.Port,
		Username: account.Username,
		UseSSL:   account.UseSSL,
		AuthType: authType,
	}
}

// generateAccountID generates a unique ID for an account
func generateAccountID(username, host string) string {
	return fmt.Sprintf("%s@%s", username, host)
}

// detectAuthType detects the likely authentication type for an email address
func detectAuthType(email string) string {
	email = strings.ToLower(email)
	
	oauthProviders := []string{
		"gmail.com", "googlemail.com",
		"outlook.com", "hotmail.com", "live.com",
		"yahoo.com",
	}
	
	for _, provider := range oauthProviders {
		if strings.Contains(email, provider) {
			return "oauth2"
		}
	}
	
	return "password"
}

// GetConfigPath returns the path to the JSON config file
func (j *JSONAccountStore) GetConfigPath() string {
	return j.configPath
}