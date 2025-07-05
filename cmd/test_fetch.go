package cmd

import (
	"context"
	"fmt"
	"imap-backup/internal/cmdutil"
	"imap-backup/internal/errors"
	"imap-backup/internal/imap"

	"github.com/spf13/cobra"
)

var testFetchCmd = &cobra.Command{
	Use:   "test-fetch [account-id]",
	Short: "Test message fetching from an account",
	Long:  `Test fetching a few messages from an account to debug .eml file issues.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTestFetch,
}

func init() {
	rootCmd.AddCommand(testFetchCmd)
	testFetchCmd.Flags().IntP("limit", "l", 3, "number of messages to test (default 3)")
	testFetchCmd.Flags().StringP("folder", "f", "INBOX", "folder to test (default INBOX)")
}

func runTestFetch(cmd *cobra.Command, args []string) error {
	accountID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	folder, _ := cmd.Flags().GetString("folder")
	
	// Load configuration
	store, err := cmdutil.GetAccountStore()
	if err != nil {
		return err
	}
	
	storedAccount, err := store.GetAccount(accountID)
	if err != nil {
		return errors.WrapAccount(err, "find", accountID)
	}
	
	// Get password from keychain if needed
	var password string
	if storedAccount.AuthType == "password" {
		// This would normally get password from keychain
		fmt.Println("Note: This test requires manual password entry for testing")
		fmt.Print("Enter password: ")
		if _, err := fmt.Scanln(&password); err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
	}
	
	// Convert to main Account type
	account := store.ConvertToAccount(*storedAccount, password)
	
	fmt.Printf("Testing message fetch from %s (folder: %s)\n", account.Name, folder)
	fmt.Printf("Will fetch up to %d messages for testing\n", limit)
	
	// Create IMAP client
	ctx := context.Background()
	client, err := imap.NewClient(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to create IMAP client: %w", err)
	}
	defer client.Close()
	
	fmt.Println("✓ Connected to IMAP server")
	
	// Get folder list
	folders, err := client.ListFolders()
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}
	
	fmt.Printf("✓ Found %d folders\n", len(folders))
	
	// Find the target folder
	var targetFolder *imap.Folder
	for _, f := range folders {
		if f.Name == folder {
			targetFolder = f
			break
		}
	}
	
	if targetFolder == nil {
		return fmt.Errorf("folder '%s' not found", folder)
	}
	
	fmt.Printf("✓ Found folder '%s' (delimiter: '%s')\n", targetFolder.Name, targetFolder.Delimiter)
	
	// Get a few messages for testing
	existingUIDs := make(map[uint32]bool) // Empty - fetch all
	messages, err := client.GetMessages(targetFolder.Name, existingUIDs)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	
	fmt.Printf("✓ Found %d messages in folder\n", len(messages))
	
	// Test the first few messages
	testCount := limit
	if len(messages) < testCount {
		testCount = len(messages)
	}
	
	for i := 0; i < testCount; i++ {
		msg := messages[i]
		fmt.Printf("\nMessage %d:\n", i+1)
		fmt.Printf("  UID: %d\n", msg.UID)
		fmt.Printf("  Subject: %s\n", msg.Subject)
		fmt.Printf("  From: %s\n", msg.From)
		fmt.Printf("  Raw size: %d bytes\n", len(msg.Raw))
		fmt.Printf("  Body size: %d chars\n", len(msg.Body))
		fmt.Printf("  HTML size: %d chars\n", len(msg.HTMLBody))
		fmt.Printf("  Attachments: %d\n", len(msg.Attachments))
		
		if len(msg.Raw) == 0 {
			fmt.Printf("  ⚠ WARNING: Raw message is empty!\n")
		} else {
			fmt.Printf("  ✓ Raw message data looks good\n")
			// Show first 100 chars of raw message
			preview := string(msg.Raw)
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Printf("  Preview: %s\n", preview)
		}
	}
	
	return nil
}