package cmdutil

import (
	"imap-backup/internal/config"
	"imap-backup/internal/errors"
	"imap-backup/internal/keychain"
)

// GetAccountStore creates and returns a new JSON account store
// This eliminates duplication of store creation logic across commands
func GetAccountStore() (*config.JSONAccountStore, error) {
	store, err := config.NewJSONAccountStore()
	if err != nil {
		return nil, errors.WrapStore(err, "create")
	}
	return store, nil
}

// LoadAccountStore creates a store and returns it (accounts loaded on-demand)
func LoadAccountStore() (*config.JSONAccountStore, error) {
	store, err := GetAccountStore()
	if err != nil {
		return nil, err
	}
	return store, nil
}

// GetAccountsFromStore loads accounts from the store
func GetAccountsFromStore(store *config.JSONAccountStore) ([]config.StoredAccount, error) {
	accounts, err := store.LoadAccounts()
	if err != nil {
		return nil, errors.WrapStore(err, "load accounts from")
	}
	return accounts, nil
}

// SaveAccountsToStore saves accounts to the store
func SaveAccountsToStore(store *config.JSONAccountStore, accounts []config.StoredAccount) error {
	if err := store.SaveAccounts(accounts); err != nil {
		return errors.WrapStore(err, "save accounts to")
	}
	return nil
}

// GetPasswordForAccount retrieves password for an account from keychain
// This eliminates duplication of password retrieval logic
func GetPasswordForAccount(account config.StoredAccount) (string, error) {
	if account.AuthType != "password" {
		return "", nil
	}
	
	keychainSvc := keychain.NewKeychainService()
	password, err := keychainSvc.GetPassword(account.Host, account.Username)
	if err != nil {
		return "", errors.WrapKeychain(err, "get")
	}
	return password, nil
}

// ConvertStoredAccountToAccount converts a stored account to a full account with password
func ConvertStoredAccountToAccount(store *config.JSONAccountStore, storedAccount config.StoredAccount) (config.Account, error) {
	password, err := GetPasswordForAccount(storedAccount)
	if err != nil {
		return config.Account{}, err
	}
	
	return store.ConvertToAccount(storedAccount, password), nil
}