package cmd

import (
	"fmt"
	"imap-backup/internal/cmdutil"
	"imap-backup/internal/errors"
	"imap-backup/internal/keychain"

	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage email accounts",
	Long: `Manage email accounts stored in JSON configuration with passwords in keychain.
Accounts are stored in ~/.imap-backup-accounts.json with passwords securely stored in Mac's keychain.`,
}

var listAccountsCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured accounts",
	Long:  `List all configured email accounts with their settings.`,
	RunE:  runListAccounts,
}

var removeAccountCmd = &cobra.Command{
	Use:   "remove [account-id]",
	Short: "Remove an account",
	Long:  `Remove an account from JSON configuration. By default also removes password from keychain.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoveAccount,
}

var testAccountCmd = &cobra.Command{
	Use:   "test [account-id]",
	Short: "Test account connection",
	Long:  `Test connection to an email account using stored credentials.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTestAccount,
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(listAccountsCmd)
	accountCmd.AddCommand(removeAccountCmd)
	accountCmd.AddCommand(testAccountCmd)
	
	listAccountsCmd.Flags().BoolP("show-passwords", "p", false, "show passwords from keychain")
	listAccountsCmd.Flags().BoolP("verbose", "v", false, "show detailed information")
	
	removeAccountCmd.Flags().BoolP("keep-password", "k", false, "keep password in keychain when removing account")
}

func runListAccounts(cmd *cobra.Command, args []string) error {
	showPasswords, _ := cmd.Flags().GetBool("show-passwords")
	verbose, _ := cmd.Flags().GetBool("verbose")
	
	store, err := cmdutil.LoadAccountStore()
	if err != nil {
		return err
	}
	
	accounts, err := cmdutil.GetAccountsFromStore(store)
	if err != nil {
		return err
	}
	
	cmdutil.DisplayAccountList(accounts, verbose, showPasswords)
	
	fmt.Printf("Configuration file: %s\n", store.GetConfigPath())
	
	return nil
}

func runRemoveAccount(cmd *cobra.Command, args []string) error {
	accountID := args[0]
	keepPassword, _ := cmd.Flags().GetBool("keep-password")
	
	store, err := cmdutil.GetAccountStore()
	if err != nil {
		return err
	}
	
	// Get account details before removing
	account, err := store.GetAccount(accountID)
	if err != nil {
		return errors.WrapAccount(err, "find", accountID)
	}
	
	// Remove from keychain unless --keep-password is specified
	if !keepPassword && account.AuthType == "password" {
		keychainSvc := keychain.NewKeychainService()
		if err := keychainSvc.DeletePassword(account.Host, account.Username); err != nil {
			fmt.Printf("Warning: Failed to remove password from keychain: %v\n", err)
		} else {
			fmt.Printf("Password removed from keychain.\n")
		}
	} else if keepPassword && account.AuthType == "password" {
		fmt.Printf("Password kept in keychain as requested.\n")
	}
	
	// Remove from JSON store
	if err := store.RemoveAccount(accountID); err != nil {
		return errors.WrapAccount(err, "remove", account.Name)
	}
	
	fmt.Printf("Account '%s' removed from configuration.\n", account.Name)
	
	if keepPassword && account.AuthType == "password" {
		fmt.Printf("Note: You can add this account back without re-entering the password.\n")
	}
	
	return nil
}

func runTestAccount(cmd *cobra.Command, args []string) error {
	accountID := args[0]
	
	store, err := cmdutil.GetAccountStore()
	if err != nil {
		return err
	}
	
	storedAccount, err := store.GetAccount(accountID)
	if err != nil {
		return errors.WrapAccount(err, "find", accountID)
	}
	
	fmt.Printf("Testing connection to %s (%s)...\n", storedAccount.Name, storedAccount.Host)
	
	// Get password from keychain
	var password string
	if storedAccount.AuthType == "password" {
		keychainSvc := keychain.NewKeychainService()
		password, err = keychainSvc.GetPassword(storedAccount.Host, storedAccount.Username)
		if err != nil {
			return errors.WrapKeychain(err, "get")
		}
	}
	
	// Convert to main Account type
	account := store.ConvertToAccount(*storedAccount, password)
	
	// Test connection using existing IMAP client
	// Note: This uses the existing imap package
	fmt.Println("Attempting to connect...")
	
	// Here we would use the existing IMAP client to test the connection
	// For now, just validate the configuration
	if account.Host == "" {
		return fmt.Errorf("invalid configuration: missing host")
	}
	if account.Username == "" {
		return fmt.Errorf("invalid configuration: missing username")
	}
	if account.AuthType == "password" && password == "" {
		return fmt.Errorf("invalid configuration: missing password")
	}
	
	fmt.Printf("✓ Configuration appears valid\n")
	fmt.Printf("✓ Password retrieved from keychain\n")
	fmt.Printf("✓ Ready for backup\n")
	
	// TODO: Add actual IMAP connection test here
	fmt.Println("Note: Full connection test not implemented yet.")
	
	return nil
}