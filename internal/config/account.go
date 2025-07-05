package config

import "time"

// BaseAccount contains the common fields for all account types
type BaseAccount struct {
	Name     string `json:"name" mapstructure:"name"`
	Host     string `json:"host" mapstructure:"host"`
	Port     int    `json:"port" mapstructure:"port"`
	Username string `json:"username" mapstructure:"username"`
	UseSSL   bool   `json:"use_ssl" mapstructure:"use_ssl"`
	AuthType string `json:"auth_type" mapstructure:"auth_type"`
}

// Account represents a basic email account configuration
type Account struct {
	BaseAccount
	Password string `mapstructure:"password"`
}

// StoredAccount represents an account with additional metadata for JSON storage
type StoredAccount struct {
	BaseAccount
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastBackup time.Time `json:"last_backup,omitempty"`
}

// ToAccount converts a StoredAccount to a basic Account
func (sa *StoredAccount) ToAccount() Account {
	return Account{
		BaseAccount: sa.BaseAccount,
		// Password is retrieved from keychain separately
	}
}

// UpdateBaseAccount updates the base account fields
func (sa *StoredAccount) UpdateBaseAccount(base BaseAccount) {
	sa.BaseAccount = base
	sa.UpdatedAt = time.Now()
}