package cmd

import (
	"fmt"
	"imap-backup/internal/keychain"

	"github.com/spf13/cobra"
)

var keychainCmd = &cobra.Command{
	Use:   "keychain",
	Short: "Manage keychain passwords",
	Long: `Manage passwords stored in Mac's keychain independently of account configuration.
Useful for cleaning up orphaned passwords or managing credentials separately.`,
}

var listKeychainCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored keychain passwords",
	Long:  `List passwords stored in Mac's keychain for IMAP accounts.`,
	RunE:  runListKeychain,
}

var removeKeychainCmd = &cobra.Command{
	Use:   "remove [server] [username]",
	Short: "Remove a password from keychain",
	Long:  `Remove a specific password from Mac's keychain.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runRemoveKeychain,
}

var addKeychainCmd = &cobra.Command{
	Use:   "add [server] [username]",
	Short: "Add a password to keychain",
	Long:  `Add a password to Mac's keychain for an IMAP server.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runAddKeychain,
}

var testKeychainCmd = &cobra.Command{
	Use:   "test",
	Short: "Test keychain access",
	Long:  `Test if the application can access Mac's keychain.`,
	RunE:  runTestKeychain,
}

func init() {
	rootCmd.AddCommand(keychainCmd)
	keychainCmd.AddCommand(listKeychainCmd)
	keychainCmd.AddCommand(removeKeychainCmd)
	keychainCmd.AddCommand(addKeychainCmd)
	keychainCmd.AddCommand(testKeychainCmd)
	
	addKeychainCmd.Flags().StringP("password", "p", "", "password (will prompt if not provided)")
}

func runListKeychain(cmd *cobra.Command, args []string) error {
	keychainSvc := keychain.NewKeychainService()
	
	fmt.Println("Testing keychain access...")
	if err := keychainSvc.TestKeychainAccess(); err != nil {
		return fmt.Errorf("cannot access keychain: %w", err)
	}
	
	fmt.Println("✓ Keychain access successful")
	
	accounts, err := keychainSvc.ListStoredAccounts()
	if err != nil {
		return fmt.Errorf("failed to list keychain accounts: %w", err)
	}
	
	if len(accounts) == 0 {
		fmt.Println("No IMAP-related passwords found in keychain.")
		return nil
	}
	
	fmt.Printf("Found %d IMAP-related entries in keychain:\n", len(accounts))
	for i, account := range accounts {
		fmt.Printf("%d. %s\n", i+1, account)
	}
	
	fmt.Println("\nNote: This is a simplified search. Use 'Keychain Access.app' for detailed management.")
	
	return nil
}

func runRemoveKeychain(cmd *cobra.Command, args []string) error {
	server := args[0]
	username := args[1]
	
	keychainSvc := keychain.NewKeychainService()
	
	fmt.Printf("Removing password for %s@%s from keychain...\n", username, server)
	
	if err := keychainSvc.DeletePassword(server, username); err != nil {
		return fmt.Errorf("failed to remove password from keychain: %w", err)
	}
	
	fmt.Printf("✓ Password removed successfully.\n")
	
	return nil
}

func runAddKeychain(cmd *cobra.Command, args []string) error {
	server := args[0]
	username := args[1]
	password, _ := cmd.Flags().GetString("password")
	
	// Prompt for password if not provided
	if password == "" {
		fmt.Printf("Enter password for %s@%s: ", username, server)
		if _, err := fmt.Scanln(&password); err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
	}
	
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	
	keychainSvc := keychain.NewKeychainService()
	
	fmt.Printf("Storing password for %s@%s in keychain...\n", username, server)
	
	if err := keychainSvc.StorePassword(server, username, password); err != nil {
		return fmt.Errorf("failed to store password in keychain: %w", err)
	}
	
	fmt.Printf("✓ Password stored successfully.\n")
	
	return nil
}

func runTestKeychain(cmd *cobra.Command, args []string) error {
	keychainSvc := keychain.NewKeychainService()
	
	fmt.Println("Testing keychain access...")
	
	if err := keychainSvc.TestKeychainAccess(); err != nil {
		return fmt.Errorf("keychain access failed: %w", err)
	}
	
	fmt.Println("✓ Keychain access successful")
	fmt.Println("✓ Can read keychain items")
	fmt.Println("✓ Ready to store and retrieve passwords")
	
	return nil
}