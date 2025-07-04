package keychain

import (
	"fmt"
	"imap-backup/internal/security"
	"os/exec"
	"strings"
)

// KeychainService handles keychain operations for email accounts
type KeychainService struct{}

// NewKeychainService creates a new keychain service
func NewKeychainService() *KeychainService {
	return &KeychainService{}
}

// StorePassword stores a password in Mac's keychain for an email account
func (k *KeychainService) StorePassword(server, username, password string) error {
	// Validate inputs to prevent command injection
	if err := security.ValidateHostname(server); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}
	
	if err := security.ValidateUsername(username); err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}
	
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	
	// First, try to update existing entry
	cmd := exec.Command("security", "add-internet-password",
		"-s", server,
		"-a", username,
		"-w", password,
		"-U", // Update if exists
		"-T", "", // Allow access by all applications
	)
	
	if err := cmd.Run(); err != nil {
		// If update fails, try to add new entry
		cmd = exec.Command("security", "add-internet-password",
			"-s", server,
			"-a", username,
			"-w", password,
			"-T", "", // Allow access by all applications
		)
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to store password in keychain: %w", err)
		}
	}
	
	return nil
}

// GetPassword retrieves a password from Mac's keychain
func (k *KeychainService) GetPassword(server, username string) (string, error) {
	// Validate inputs to prevent command injection
	if err := security.ValidateHostname(server); err != nil {
		return "", fmt.Errorf("invalid server name: %w", err)
	}
	
	if err := security.ValidateUsername(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}
	
	// Try internet password first
	cmd := exec.Command("security", "find-internet-password",
		"-s", server,
		"-a", username,
		"-w", // Print password only
	)
	
	output, err := cmd.Output()
	if err == nil {
		password := strings.TrimSpace(string(output))
		if password != "" {
			return password, nil
		}
	}
	
	// Try generic password as fallback
	serviceName := fmt.Sprintf("imap-backup-%s", server)
	cmd = exec.Command("security", "find-generic-password",
		"-s", serviceName,
		"-a", username,
		"-w", // Print password only
	)
	
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("password not found in keychain for %s@%s", username, server)
	}
	
	password := strings.TrimSpace(string(output))
	if password == "" {
		return "", fmt.Errorf("empty password found in keychain for %s@%s", username, server)
	}
	
	return password, nil
}

// DeletePassword removes a password from Mac's keychain
func (k *KeychainService) DeletePassword(server, username string) error {
	// Validate inputs to prevent command injection
	if err := security.ValidateHostname(server); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}
	
	if err := security.ValidateUsername(username); err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}
	
	// Delete internet password
	cmd := exec.Command("security", "delete-internet-password",
		"-s", server,
		"-a", username,
	)
	
	internetErr := cmd.Run()
	
	// Delete generic password
	serviceName := fmt.Sprintf("imap-backup-%s", server)
	cmd = exec.Command("security", "delete-generic-password",
		"-s", serviceName,
		"-a", username,
	)
	
	genericErr := cmd.Run()
	
	// Return error only if both failed
	if internetErr != nil && genericErr != nil {
		return fmt.Errorf("failed to delete password from keychain: %v", internetErr)
	}
	
	return nil
}

// ListStoredAccounts lists accounts that have passwords stored in keychain
func (k *KeychainService) ListStoredAccounts() ([]string, error) {
	// Search for internet passwords related to IMAP
	cmd := exec.Command("security", "dump-keychain", "-d")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list keychain items: %w", err)
	}
	
	var accounts []string
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "imap") || strings.Contains(line, "mail") {
			// Parse account information from keychain dump
			// This is a simplified parser
			if strings.Contains(line, "@") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.Contains(part, "@") && strings.Contains(part, ".") {
						accounts = append(accounts, part)
					}
				}
			}
		}
	}
	
	return accounts, nil
}

// TestKeychainAccess tests if we can access the keychain
func (k *KeychainService) TestKeychainAccess() error {
	// Try to list keychain items to test access
	cmd := exec.Command("security", "list-keychains")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot access keychain: %w", err)
	}
	
	return nil
}