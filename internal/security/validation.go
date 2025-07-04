package security

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// Maximum lengths for various inputs
	MaxFolderNameLength = 255
	MaxAccountNameLength = 100
	MaxHostnameLength = 253
	MaxUsernameLength = 320 // RFC 5321 maximum email length
	
	// Secure file permissions
	SecureDirectoryMode = 0700
	SecureFileMode = 0600
)

var (
	// Valid patterns
	validFolderNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._\-\s/\\]+$`)
	validHostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	validUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateFolderName validates IMAP folder names to prevent path traversal
func ValidateFolderName(name string) error {
	if name == "" {
		return errors.New("folder name cannot be empty")
	}
	
	if len(name) > MaxFolderNameLength {
		return errors.New("folder name too long")
	}
	
	// Check for path traversal attempts
	if strings.Contains(name, "..") {
		return errors.New("folder name contains path traversal sequence")
	}
	
	// Check for absolute paths
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return errors.New("folder name cannot be absolute path")
	}
	
	// Check for reserved names
	reservedNames := []string{".", "..", "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	upperName := strings.ToUpper(name)
	for _, reserved := range reservedNames {
		if upperName == reserved {
			return errors.New("folder name is reserved")
		}
	}
	
	return nil
}

// SanitizeFolderName safely sanitizes folder names for filesystem use
func SanitizeFolderName(name string) string {
	// Replace problematic characters
	replacements := map[string]string{
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
		"\x00": "_", // null byte
	}
	
	result := name
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	
	// Trim whitespace and dots from ends
	result = strings.Trim(result, " .")
	
	// Ensure it's not empty after sanitization
	if result == "" || result == strings.Repeat("_", len(result)) {
		result = "unknown"
	}
	
	// Truncate if too long
	if len(result) > MaxFolderNameLength {
		result = result[:MaxFolderNameLength]
	}
	
	return result
}

// ValidateHostname validates hostnames to prevent injection
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return errors.New("hostname cannot be empty")
	}
	
	if len(hostname) > MaxHostnameLength {
		return errors.New("hostname too long")
	}
	
	if !validHostnamePattern.MatchString(hostname) {
		return errors.New("hostname contains invalid characters")
	}
	
	return nil
}

// ValidateUsername validates email addresses/usernames
func ValidateUsername(username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}
	
	if len(username) > MaxUsernameLength {
		return errors.New("username too long")
	}
	
	if !validUsernamePattern.MatchString(username) {
		return errors.New("username is not a valid email address")
	}
	
	return nil
}

// ValidateAccountName validates account names for keychain operations
func ValidateAccountName(name string) error {
	if name == "" {
		return errors.New("account name cannot be empty")
	}
	
	if len(name) > MaxAccountNameLength {
		return errors.New("account name too long")
	}
	
	// Check for shell injection attempts
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "\"", "'", "\\", "\n", "\r", "\t"}
	for _, char := range dangerousChars {
		if strings.Contains(name, char) {
			return errors.New("account name contains dangerous characters")
		}
	}
	
	return nil
}

// SecurePath ensures a path is within the expected directory
func SecurePath(basePath, userPath string) (string, error) {
	// Clean and resolve the paths
	cleanBase := filepath.Clean(basePath)
	cleanUser := filepath.Clean(userPath)
	
	// Check if user path is absolute - this is dangerous
	if filepath.IsAbs(cleanUser) {
		return "", errors.New("path escapes base directory")
	}
	
	// Join them
	fullPath := filepath.Join(cleanBase, cleanUser)
	
	// Resolve any symlinks for both paths to handle systems where /var -> /private/var
	resolvedBase, err := filepath.EvalSymlinks(cleanBase)
	if err != nil {
		// If EvalSymlinks fails, use the clean path
		resolvedBase = cleanBase
	}
	
	// Resolve the full path as well
	resolvedFull := filepath.Join(resolvedBase, cleanUser)
	
	// Ensure the full path is still within the resolved base directory
	relPath, err := filepath.Rel(resolvedBase, resolvedFull)
	if err != nil {
		return "", errors.New("unable to determine relative path")
	}
	
	if strings.HasPrefix(relPath, "..") {
		return "", errors.New("path escapes base directory")
	}
	
	return fullPath, nil
}