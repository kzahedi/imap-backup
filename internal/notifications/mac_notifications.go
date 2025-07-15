package notifications

import (
	"fmt"
	"os/exec"
	"strings"
)

// MacNotificationCenter handles macOS notification center integration
type MacNotificationCenter struct {
	AppName string
	AppID   string
}

// NotificationLevel represents the urgency level of a notification
type NotificationLevel int

const (
	NotificationInfo NotificationLevel = iota
	NotificationWarning
	NotificationError
	NotificationSuccess
)

// Notification represents a macOS notification
type Notification struct {
	Title    string
	Message  string
	Level    NotificationLevel
	Sound    string
	Actions  []string
	AppIcon  string
}

// NewMacNotificationCenter creates a new Mac notification center handler
func NewMacNotificationCenter(appName, appID string) *MacNotificationCenter {
	return &MacNotificationCenter{
		AppName: appName,
		AppID:   appID,
	}
}

// SendNotification sends a notification to macOS Notification Center
func (m *MacNotificationCenter) SendNotification(notification *Notification) error {
	// Build AppleScript command
	script := m.buildAppleScript(notification)
	
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to send notification: %w (output: %s)", err, string(output))
	}
	
	return nil
}

// SendBackupCompleteNotification sends a notification when backup is complete
func (m *MacNotificationCenter) SendBackupCompleteNotification(accountName string, messageCount int, duration string) error {
	notification := &Notification{
		Title:   "Email Backup Complete",
		Message: fmt.Sprintf("Successfully backed up %d messages from %s in %s", messageCount, accountName, duration),
		Level:   NotificationSuccess,
		Sound:   "Glass",
	}
	
	return m.SendNotification(notification)
}

// SendBackupErrorNotification sends a notification when backup fails
func (m *MacNotificationCenter) SendBackupErrorNotification(accountName string, errorMsg string) error {
	notification := &Notification{
		Title:   "Email Backup Failed",
		Message: fmt.Sprintf("Failed to backup %s: %s", accountName, errorMsg),
		Level:   NotificationError,
		Sound:   "Sosumi",
	}
	
	return m.SendNotification(notification)
}

// SendBackupStartNotification sends a notification when backup starts
func (m *MacNotificationCenter) SendBackupStartNotification(accountName string) error {
	notification := &Notification{
		Title:   "Email Backup Started",
		Message: fmt.Sprintf("Starting backup of %s", accountName),
		Level:   NotificationInfo,
		Sound:   "Blow",
	}
	
	return m.SendNotification(notification)
}

// SendOAuth2TokenExpiredNotification sends a notification when OAuth2 token expires
func (m *MacNotificationCenter) SendOAuth2TokenExpiredNotification(accountName string) error {
	notification := &Notification{
		Title:   "OAuth2 Token Expired",
		Message: fmt.Sprintf("OAuth2 token for %s has expired. Please re-authenticate.", accountName),
		Level:   NotificationWarning,
		Sound:   "Funk",
		Actions: []string{"Re-authenticate", "Ignore"},
	}
	
	return m.SendNotification(notification)
}

// SendRateLimitWarningNotification sends a notification when rate limiting is active
func (m *MacNotificationCenter) SendRateLimitWarningNotification(accountName string) error {
	notification := &Notification{
		Title:   "Rate Limit Active",
		Message: fmt.Sprintf("Backup of %s is being rate limited to prevent server overload", accountName),
		Level:   NotificationWarning,
		Sound:   "Ping",
	}
	
	return m.SendNotification(notification)
}

// SendLargeAttachmentWarningNotification sends a notification when large attachments are found
func (m *MacNotificationCenter) SendLargeAttachmentWarningNotification(accountName string, attachmentCount int) error {
	notification := &Notification{
		Title:   "Large Attachments Found",
		Message: fmt.Sprintf("Found %d large attachments in %s that may take longer to backup", attachmentCount, accountName),
		Level:   NotificationInfo,
		Sound:   "Submarine",
	}
	
	return m.SendNotification(notification)
}

// buildAppleScript builds the AppleScript command for sending notifications
func (m *MacNotificationCenter) buildAppleScript(notification *Notification) string {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, 
		escapeAppleScriptString(notification.Message), 
		escapeAppleScriptString(notification.Title))
	
	// Add subtitle for app name
	if m.AppName != "" {
		script += fmt.Sprintf(` subtitle "%s"`, escapeAppleScriptString(m.AppName))
	}
	
	// Add sound
	if notification.Sound != "" {
		script += fmt.Sprintf(` sound name "%s"`, notification.Sound)
	}
	
	return script
}

// escapeAppleScriptString escapes special characters in AppleScript strings
func escapeAppleScriptString(s string) string {
	// First escape backslashes, then quotes
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// IsNotificationCenterAvailable checks if Notification Center is available
func IsNotificationCenterAvailable() bool {
	cmd := exec.Command("osascript", "-e", "display notification \"test\"")
	return cmd.Run() == nil
}

// GetNotificationCenterPermission checks if the app has notification permission
func GetNotificationCenterPermission(appName string) (bool, error) {
	// Check if notifications are enabled for the app
	script := fmt.Sprintf(`tell application "System Events"
		tell application process "NotificationCenter"
			return exists (UI elements whose name contains "%s")
		end tell
	end tell`, appName)
	
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check notification permission: %w", err)
	}
	
	return strings.TrimSpace(string(output)) == "true", nil
}

// RequestNotificationPermission requests notification permission from the user
func RequestNotificationPermission(appName string) error {
	script := fmt.Sprintf(`tell application "System Events"
		display notification "imap-backup would like to send you notifications" with title "%s"
	end tell`, appName)
	
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// SetNotificationSchedule sets up notification schedule using launchd
func SetNotificationSchedule(appName string, schedule map[string]interface{}) error {
	// This would integrate with launchd to schedule notifications
	// For now, we'll just return a placeholder
	return fmt.Errorf("notification scheduling not implemented yet")
}

// GetSystemNotificationSettings gets the current notification settings
func GetSystemNotificationSettings() (map[string]interface{}, error) {
	cmd := exec.Command("defaults", "read", "com.apple.ncprefs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read notification settings: %w", err)
	}
	
	// Parse the defaults output (this is a simplified version)
	settings := make(map[string]interface{})
	settings["raw_output"] = string(output)
	
	return settings, nil
}

// EnableDoNotDisturb enables Do Not Disturb mode
func EnableDoNotDisturb() error {
	script := `tell application "System Events"
		tell application process "NotificationCenter"
			click menu bar item 1 of menu bar 1
			click button "Turn On Do Not Disturb" of group 1 of UI element 1 of scroll area 1 of window 1
		end tell
	end tell`
	
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// DisableDoNotDisturb disables Do Not Disturb mode
func DisableDoNotDisturb() error {
	script := `tell application "System Events"
		tell application process "NotificationCenter"
			click menu bar item 1 of menu bar 1
			click button "Turn Off Do Not Disturb" of group 1 of UI element 1 of scroll area 1 of window 1
		end tell
	end tell`
	
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}