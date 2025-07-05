package testutil

import (
	"imap-backup/internal/config"
	"imap-backup/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

// SetupTestStorage creates a temporary storage directory and returns a FileStorage instance
// This eliminates duplication of test storage setup across test files
func SetupTestStorage(t *testing.T) (*storage.FileStorage, string) {
	t.Helper()
	
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "backup")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	
	return storage.NewFileStorage(baseDir), baseDir
}

// SetupTestAccount creates a test account with common defaults
func SetupTestAccount(name string) config.Account {
	return config.Account{
		BaseAccount: config.BaseAccount{
			Name:     name,
			Host:     "imap.example.com",
			Port:     993,
			Username: "user@example.com",
			UseSSL:   true,
			AuthType: "password",
		},
		Password: "test-password",
	}
}

// SetupTestStoredAccount creates a test stored account with common defaults
func SetupTestStoredAccount(name string) config.StoredAccount {
	return config.StoredAccount{
		BaseAccount: config.BaseAccount{
			Name:     name,
			Host:     "imap.example.com",
			Port:     993,
			Username: "user@example.com",
			UseSSL:   true,
			AuthType: "password",
		},
		ID: "test-" + name,
	}
}

// CheckFilePermissions verifies that a file has the expected permissions
// This eliminates duplication of permission checking in tests
func CheckFilePermissions(t *testing.T, filePath string, expectedMode os.FileMode) {
	t.Helper()
	
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	
	actualMode := info.Mode().Perm()
	if actualMode != expectedMode {
		t.Errorf("File permissions = %v, want %v", actualMode, expectedMode)
	}
}

// CheckFileExists verifies that a file exists
func CheckFileExists(t *testing.T, filePath string) {
	t.Helper()
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File should exist: %s", filePath)
	}
}

// CheckFileNotExists verifies that a file does not exist
func CheckFileNotExists(t *testing.T, filePath string) {
	t.Helper()
	
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("File should not exist: %s", filePath)
	}
}

// SetupTempConfigFile creates a temporary config file for testing
func SetupTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	err := os.WriteFile(configPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	return configPath
}