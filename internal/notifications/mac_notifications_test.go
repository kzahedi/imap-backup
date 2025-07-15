package notifications

import (
	"strings"
	"testing"
)

func TestNewMacNotificationCenter(t *testing.T) {
	appName := "imap-backup"
	appID := "com.example.imap-backup"
	
	center := NewMacNotificationCenter(appName, appID)
	
	if center.AppName != appName {
		t.Errorf("Expected AppName %s, got %s", appName, center.AppName)
	}
	
	if center.AppID != appID {
		t.Errorf("Expected AppID %s, got %s", appID, center.AppID)
	}
}

func TestBuildAppleScript(t *testing.T) {
	center := NewMacNotificationCenter("imap-backup", "com.example.imap-backup")
	
	tests := []struct {
		name         string
		notification *Notification
		expected     string
	}{
		{
			name: "basic notification",
			notification: &Notification{
				Title:   "Test Title",
				Message: "Test Message",
				Level:   NotificationInfo,
			},
			expected: `display notification "Test Message" with title "Test Title" subtitle "imap-backup"`,
		},
		{
			name: "notification with sound",
			notification: &Notification{
				Title:   "Test Title",
				Message: "Test Message",
				Level:   NotificationInfo,
				Sound:   "Glass",
			},
			expected: `display notification "Test Message" with title "Test Title" subtitle "imap-backup" sound name "Glass"`,
		},
		{
			name: "notification with special characters",
			notification: &Notification{
				Title:   `Test "Title"`,
				Message: `Test "Message"`,
				Level:   NotificationInfo,
			},
			expected: `display notification "Test \"Message\"" with title "Test \"Title\"" subtitle "imap-backup"`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := center.buildAppleScript(tt.notification)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEscapeAppleScriptString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "simple string",
			expected: "simple string",
		},
		{
			input:    `string with "quotes"`,
			expected: `string with \"quotes\"`,
		},
		{
			input:    `string with \backslash`,
			expected: `string with \\backslash`,
		},
		{
			input:    `string with "quotes" and \backslash`,
			expected: `string with \"quotes\" and \\backslash`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeAppleScriptString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNotificationLevels(t *testing.T) {
	// Test that notification levels are properly defined
	levels := []NotificationLevel{
		NotificationInfo,
		NotificationWarning,
		NotificationError,
		NotificationSuccess,
	}
	
	expectedValues := []int{0, 1, 2, 3}
	
	for i, level := range levels {
		if int(level) != expectedValues[i] {
			t.Errorf("Expected level %d to have value %d, got %d", i, expectedValues[i], int(level))
		}
	}
}

func TestSendBackupCompleteNotification(t *testing.T) {
	center := NewMacNotificationCenter("imap-backup", "com.example.imap-backup")
	
	// This test verifies the notification structure, not the actual sending
	// since that requires macOS system integration
	accountName := "test@example.com"
	messageCount := 150
	duration := "2m30s"
	
	// We can't easily test the actual notification sending without mocking
	// So we'll test the notification structure by checking the AppleScript
	notification := &Notification{
		Title:   "Email Backup Complete",
		Message: "Successfully backed up 150 messages from test@example.com in 2m30s",
		Level:   NotificationSuccess,
		Sound:   "Glass",
	}
	
	script := center.buildAppleScript(notification)
	
	if !strings.Contains(script, "Email Backup Complete") {
		t.Error("AppleScript should contain notification title")
	}
	
	if !strings.Contains(script, "Successfully backed up 150 messages") {
		t.Error("AppleScript should contain notification message")
	}
	
	if !strings.Contains(script, "Glass") {
		t.Error("AppleScript should contain sound name")
	}
	
	// Test the actual method (this will only work on macOS with proper permissions)
	err := center.SendBackupCompleteNotification(accountName, messageCount, duration)
	if err != nil {
		t.Logf("Note: SendBackupCompleteNotification failed (expected on non-macOS or without permissions): %v", err)
	}
}

func TestSendBackupErrorNotification(t *testing.T) {
	center := NewMacNotificationCenter("imap-backup", "com.example.imap-backup")
	
	accountName := "test@example.com"
	errorMsg := "Connection timeout"
	
	notification := &Notification{
		Title:   "Email Backup Failed",
		Message: "Failed to backup test@example.com: Connection timeout",
		Level:   NotificationError,
		Sound:   "Sosumi",
	}
	
	script := center.buildAppleScript(notification)
	
	if !strings.Contains(script, "Email Backup Failed") {
		t.Error("AppleScript should contain error title")
	}
	
	if !strings.Contains(script, "Connection timeout") {
		t.Error("AppleScript should contain error message")
	}
	
	if !strings.Contains(script, "Sosumi") {
		t.Error("AppleScript should contain error sound")
	}
	
	// Test the actual method
	err := center.SendBackupErrorNotification(accountName, errorMsg)
	if err != nil {
		t.Logf("Note: SendBackupErrorNotification failed (expected on non-macOS or without permissions): %v", err)
	}
}

func TestSendOAuth2TokenExpiredNotification(t *testing.T) {
	center := NewMacNotificationCenter("imap-backup", "com.example.imap-backup")
	
	accountName := "test@gmail.com"
	
	// Test the actual method
	err := center.SendOAuth2TokenExpiredNotification(accountName)
	if err != nil {
		t.Logf("Note: SendOAuth2TokenExpiredNotification failed (expected on non-macOS or without permissions): %v", err)
	}
}

func TestIsNotificationCenterAvailable(t *testing.T) {
	// This will return false on non-macOS systems
	available := IsNotificationCenterAvailable()
	t.Logf("Notification Center available: %v", available)
	
	// We can't assert a specific value since this depends on the OS
	// Just verify the function doesn't panic
}

func TestGetNotificationCenterPermission(t *testing.T) {
	appName := "imap-backup"
	
	permission, err := GetNotificationCenterPermission(appName)
	if err != nil {
		t.Logf("Note: GetNotificationCenterPermission failed (expected on non-macOS): %v", err)
	} else {
		t.Logf("Notification permission for %s: %v", appName, permission)
	}
}

func TestGetSystemNotificationSettings(t *testing.T) {
	settings, err := GetSystemNotificationSettings()
	if err != nil {
		t.Logf("Note: GetSystemNotificationSettings failed (expected on non-macOS): %v", err)
	} else {
		t.Logf("Got %d notification settings", len(settings))
		
		// Verify we got some settings
		if len(settings) == 0 {
			t.Error("Expected some notification settings")
		}
	}
}

// Benchmark tests
func BenchmarkBuildAppleScript(b *testing.B) {
	center := NewMacNotificationCenter("imap-backup", "com.example.imap-backup")
	notification := &Notification{
		Title:   "Test Title",
		Message: "Test Message",
		Level:   NotificationInfo,
		Sound:   "Glass",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		center.buildAppleScript(notification)
	}
}

func BenchmarkEscapeAppleScriptString(b *testing.B) {
	testString := `This is a test string with "quotes" and \backslashes`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeAppleScriptString(testString)
	}
}