package config

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/spf13/viper"
)

func TestCreateSampleConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Temporarily change HOME to our temp directory
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", origHome)
	}()
	os.Setenv("HOME", tmpDir)
	
	err := CreateSampleConfig()
	if err != nil {
		t.Fatalf("CreateSampleConfig failed: %v", err)
	}
	
	// Check if the file was created
	configPath := filepath.Join(tmpDir, ".imap-backup.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}
	
	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", info.Mode().Perm())
	}
	
	// Check file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	
	contentStr := string(content)
	expectedSubstrings := []string{
		"# IMAP Backup Configuration",
		"accounts:",
		"Gmail",
		"imap.gmail.com",
		"Outlook",
		"outlook.office365.com",
		"use_ssl: true",
	}
	
	for _, substr := range expectedSubstrings {
		if !contains(contentStr, substr) {
			t.Errorf("Expected config file to contain %q", substr)
		}
	}
}

func TestConfig_Structure(t *testing.T) {
	// Test that Config struct can be created and has expected fields
	cfg := Config{
		Accounts: []Account{
			{
				Name:     "Test Account",
				Host:     "imap.example.com",
				Port:     993,
				Username: "test@example.com",
				Password: "test-password",
				UseSSL:   true,
			},
		},
	}
	
	if len(cfg.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(cfg.Accounts))
	}
	
	account := cfg.Accounts[0]
	if account.Name != "Test Account" {
		t.Errorf("Expected account name 'Test Account', got %s", account.Name)
	}
	
	if account.Host != "imap.example.com" {
		t.Errorf("Expected host 'imap.example.com', got %s", account.Host)
	}
	
	if account.Port != 993 {
		t.Errorf("Expected port 993, got %d", account.Port)
	}
	
	if account.Username != "test@example.com" {
		t.Errorf("Expected username 'test@example.com', got %s", account.Username)
	}
	
	if account.Password != "test-password" {
		t.Errorf("Expected password 'test-password', got %s", account.Password)
	}
	
	if !account.UseSSL {
		t.Error("Expected UseSSL to be true")
	}
}

func TestLoad_WithViperConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	
	configContent := `
accounts:
  - name: "Test Gmail"
    host: "imap.gmail.com"
    port: 993
    username: "test@gmail.com"
    password: "test-password"
    use_ssl: true
  - name: "Test Outlook"
    host: "outlook.office365.com"
    port: 993
    username: "test@outlook.com"
    password: "test-password"
    use_ssl: true
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// Configure viper to use our test config file
	viper.SetConfigFile(configFile)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config with viper: %v", err)
	}
	
	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	// Verify accounts
	if len(cfg.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(cfg.Accounts))
	}
	
	// Check first account
	if cfg.Accounts[0].Name != "Test Gmail" {
		t.Errorf("Expected first account name 'Test Gmail', got %s", cfg.Accounts[0].Name)
	}
	
	if cfg.Accounts[0].Host != "imap.gmail.com" {
		t.Errorf("Expected first account host 'imap.gmail.com', got %s", cfg.Accounts[0].Host)
	}
	
	// Check second account
	if cfg.Accounts[1].Name != "Test Outlook" {
		t.Errorf("Expected second account name 'Test Outlook', got %s", cfg.Accounts[1].Name)
	}
	
	if cfg.Accounts[1].Host != "outlook.office365.com" {
		t.Errorf("Expected second account host 'outlook.office365.com', got %s", cfg.Accounts[1].Host)
	}
}

func TestLoad_NoAccounts(t *testing.T) {
	// Create a temporary config file with no accounts
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty-config.yaml")
	
	configContent := `
accounts: []
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// Configure viper to use our test config file
	viper.SetConfigFile(configFile)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config with viper: %v", err)
	}
	
	// Load config - should fail
	_, err = Load()
	if err == nil {
		t.Error("Expected error when loading config with no accounts")
	}
	
	if !contains(err.Error(), "no accounts configured") {
		t.Errorf("Expected error message to contain 'no accounts configured', got: %v", err)
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-config.yaml")
	
	configContent := `
accounts:
  - name: "Test Account"
    host: imap.example.com
    port: "invalid-port"  # This should be a number
    username: test@example.com
    password: test-password
    use_ssl: true
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// Configure viper to use our test config file
	viper.SetConfigFile(configFile)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config with viper: %v", err)
	}
	
	// Load config - should fail due to invalid port
	_, err = Load()
	if err == nil {
		t.Error("Expected error when loading config with invalid port")
	}
}

func TestLoadFromJSONStore_NoStore(t *testing.T) {
	// This test checks the behavior when no JSON store exists
	accounts, err := loadFromJSONStore()
	if err == nil {
		t.Error("Expected error when JSON store doesn't exist")
	}
	
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(accounts))
	}
}

func TestCreateSampleConfig_InvalidHome(t *testing.T) {
	// Test behavior when HOME environment variable is invalid
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", origHome)
	}()
	
	// Set HOME to an invalid path
	os.Setenv("HOME", "/nonexistent/invalid/path")
	
	err := CreateSampleConfig()
	if err == nil {
		t.Error("Expected error when HOME points to invalid path")
	}
}

func TestConfig_EmptyAccounts(t *testing.T) {
	// Test Config with empty accounts slice
	cfg := Config{
		Accounts: []Account{},
	}
	
	if len(cfg.Accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(cfg.Accounts))
	}
}

func TestConfig_MultipleAccounts(t *testing.T) {
	// Test Config with multiple accounts
	cfg := Config{
		Accounts: []Account{
			{
				Name:     "Gmail",
				Host:     "imap.gmail.com",
				Port:     993,
				Username: "test@gmail.com",
				Password: "gmail-password",
				UseSSL:   true,
			},
			{
				Name:     "Outlook",
				Host:     "outlook.office365.com",
				Port:     993,
				Username: "test@outlook.com",
				Password: "outlook-password",
				UseSSL:   true,
			},
			{
				Name:     "Custom IMAP",
				Host:     "mail.example.com",
				Port:     143,
				Username: "test@example.com",
				Password: "custom-password",
				UseSSL:   false,
			},
		},
	}
	
	if len(cfg.Accounts) != 3 {
		t.Errorf("Expected 3 accounts, got %d", len(cfg.Accounts))
	}
	
	// Check account names
	expectedNames := []string{"Gmail", "Outlook", "Custom IMAP"}
	for i, expected := range expectedNames {
		if cfg.Accounts[i].Name != expected {
			t.Errorf("Expected account %d name '%s', got '%s'", i, expected, cfg.Accounts[i].Name)
		}
	}
	
	// Check ports
	expectedPorts := []int{993, 993, 143}
	for i, expected := range expectedPorts {
		if cfg.Accounts[i].Port != expected {
			t.Errorf("Expected account %d port %d, got %d", i, expected, cfg.Accounts[i].Port)
		}
	}
	
	// Check SSL settings
	expectedSSL := []bool{true, true, false}
	for i, expected := range expectedSSL {
		if cfg.Accounts[i].UseSSL != expected {
			t.Errorf("Expected account %d SSL %v, got %v", i, expected, cfg.Accounts[i].UseSSL)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}