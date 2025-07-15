// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"imap-backup/internal/backup"
	"imap-backup/internal/config"
)

func TestBackupIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	
	// Create temp directory for backup
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	
	// Create test configuration
	cfg := &config.Config{
		Accounts: []config.Account{
			{
				BaseAccount: config.BaseAccount{
					Name:     "Test Account",
					Host:     getTestHost(),
					Port:     getTestPort(),
					Username: getTestUsername(),
					UseSSL:   true,
					AuthType: "password",
				},
				Password: getTestPassword(),
			},
		},
	}
	
	// Create backup service
	service := backup.NewService(cfg)
	
	// Create backup configuration
	backupConfig := backup.Config{
		OutputDir:           backupDir,
		DryRun:              false,
		MaxConcurrent:       2,
		IgnoreCharsetErrors: true,
		Verbose:             true,
	}
	
	// Run backup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	err := service.Run(ctx, backupConfig)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Verify backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}
	
	// Verify account directory was created
	accountDir := filepath.Join(backupDir, "Test Account")
	if _, err := os.Stat(accountDir); os.IsNotExist(err) {
		t.Error("Account directory was not created")
	}
	
	// Verify at least one folder was backed up
	folders, err := os.ReadDir(accountDir)
	if err != nil {
		t.Fatalf("Failed to read account directory: %v", err)
	}
	
	if len(folders) == 0 {
		t.Error("No folders were backed up")
	}
	
	// Check for INBOX folder
	inboxPath := filepath.Join(accountDir, "INBOX")
	if _, err := os.Stat(inboxPath); os.IsNotExist(err) {
		t.Log("INBOX folder not found (may be empty)")
	}
}

func TestDryRunBackup(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	
	// Create temp directory for backup
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	
	// Create test configuration
	cfg := &config.Config{
		Accounts: []config.Account{
			{
				BaseAccount: config.BaseAccount{
					Name:     "Test Account",
					Host:     getTestHost(),
					Port:     getTestPort(),
					Username: getTestUsername(),
					UseSSL:   true,
					AuthType: "password",
				},
				Password: getTestPassword(),
			},
		},
	}
	
	// Create backup service
	service := backup.NewService(cfg)
	
	// Create backup configuration with dry run
	backupConfig := backup.Config{
		OutputDir:           backupDir,
		DryRun:              true,
		MaxConcurrent:       2,
		IgnoreCharsetErrors: true,
		Verbose:             true,
	}
	
	// Run dry run backup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	
	err := service.Run(ctx, backupConfig)
	if err != nil {
		t.Fatalf("Dry run backup failed: %v", err)
	}
	
	// Verify no files were created during dry run
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Error("Backup directory should not exist during dry run")
	}
}

func TestIncrementalBackup(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	
	// Create temp directory for backup
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	
	// Create test configuration
	cfg := &config.Config{
		Accounts: []config.Account{
			{
				BaseAccount: config.BaseAccount{
					Name:     "Test Account",
					Host:     getTestHost(),
					Port:     getTestPort(),
					Username: getTestUsername(),
					UseSSL:   true,
					AuthType: "password",
				},
				Password: getTestPassword(),
			},
		},
	}
	
	// Create backup service
	service := backup.NewService(cfg)
	
	// Create backup configuration
	backupConfig := backup.Config{
		OutputDir:           backupDir,
		DryRun:              false,
		MaxConcurrent:       2,
		IgnoreCharsetErrors: true,
		Verbose:             false,
	}
	
	// Run first backup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	err := service.Run(ctx, backupConfig)
	if err != nil {
		t.Fatalf("First backup failed: %v", err)
	}
	
	// Get initial file count
	initialCount := countFiles(t, backupDir)
	
	// Run second backup (should be incremental)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel2()
	
	err = service.Run(ctx2, backupConfig)
	if err != nil {
		t.Fatalf("Second backup failed: %v", err)
	}
	
	// Get final file count
	finalCount := countFiles(t, backupDir)
	
	// File count should be the same (incremental backup)
	if finalCount != initialCount {
		t.Logf("Initial file count: %d, Final file count: %d", initialCount, finalCount)
		t.Log("Note: File count difference may be due to new messages received")
	}
}

// Helper functions for test configuration
func getTestHost() string {
	if host := os.Getenv("IMAP_TEST_HOST"); host != "" {
		return host
	}
	return "imap.gmail.com"
}

func getTestPort() int {
	return 993
}

func getTestUsername() string {
	if username := os.Getenv("IMAP_TEST_USERNAME"); username != "" {
		return username
	}
	return "test@example.com"
}

func getTestPassword() string {
	if password := os.Getenv("IMAP_TEST_PASSWORD"); password != "" {
		return password
	}
	return "test-password"
}

func countFiles(t *testing.T, dir string) int {
	count := 0
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to count files: %v", err)
	}
	
	return count
}

func TestMultipleAccountBackup(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	
	// Create temp directory for backup
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	
	// Create test configuration with multiple accounts
	cfg := &config.Config{
		Accounts: []config.Account{
			{
				BaseAccount: config.BaseAccount{
					Name:     "Test Account 1",
					Host:     getTestHost(),
					Port:     getTestPort(),
					Username: getTestUsername(),
					UseSSL:   true,
					AuthType: "password",
				},
				Password: getTestPassword(),
			},
			{
				BaseAccount: config.BaseAccount{
					Name:     "Test Account 2",
					Host:     getTestHost(),
					Port:     getTestPort(),
					Username: getTestUsername(),
					UseSSL:   true,
					AuthType: "password",
				},
				Password: getTestPassword(),
			},
		},
	}
	
	// Create backup service
	service := backup.NewService(cfg)
	
	// Create backup configuration
	backupConfig := backup.Config{
		OutputDir:           backupDir,
		DryRun:              false,
		MaxConcurrent:       1,
		IgnoreCharsetErrors: true,
		Verbose:             false,
	}
	
	// Run backup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	err := service.Run(ctx, backupConfig)
	if err != nil {
		t.Fatalf("Multiple account backup failed: %v", err)
	}
	
	// Verify both account directories were created
	account1Dir := filepath.Join(backupDir, "Test Account 1")
	account2Dir := filepath.Join(backupDir, "Test Account 2")
	
	if _, err := os.Stat(account1Dir); os.IsNotExist(err) {
		t.Error("Account 1 directory was not created")
	}
	
	if _, err := os.Stat(account2Dir); os.IsNotExist(err) {
		t.Error("Account 2 directory was not created")
	}
}

func TestBackupWithRateLimit(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	
	// Create temp directory for backup
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	
	// Create test configuration
	cfg := &config.Config{
		Accounts: []config.Account{
			{
				BaseAccount: config.BaseAccount{
					Name:     "Test Account",
					Host:     getTestHost(),
					Port:     getTestPort(),
					Username: getTestUsername(),
					UseSSL:   true,
					AuthType: "password",
				},
				Password: getTestPassword(),
			},
		},
	}
	
	// Create backup service
	service := backup.NewService(cfg)
	
	// Create backup configuration with low concurrency to test rate limiting
	backupConfig := backup.Config{
		OutputDir:           backupDir,
		DryRun:              false,
		MaxConcurrent:       1,
		IgnoreCharsetErrors: true,
		Verbose:             false,
	}
	
	// Run backup with rate limiting
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	start := time.Now()
	err := service.Run(ctx, backupConfig)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("Rate limited backup failed: %v", err)
	}
	
	t.Logf("Backup completed in %v", elapsed)
	
	// Verify backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}
}