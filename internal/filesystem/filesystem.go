package filesystem

import (
	"imap-backup/internal/errors"
	"imap-backup/internal/security"
	"os"
	"path/filepath"
)

// EnsureDir creates a directory with the specified permissions if it doesn't exist
func EnsureDir(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return errors.WrapWithContext(err, "create directory", path)
	}
	return nil
}

// EnsureSecureDir creates a directory with secure permissions
func EnsureSecureDir(path string) error {
	return EnsureDir(path, security.SecureDirectoryMode)
}

// SecureJoin safely joins path elements using security validation
func SecureJoin(basePath string, elements ...string) (string, error) {
	if len(elements) == 0 {
		return basePath, nil
	}
	
	userPath := filepath.Join(elements...)
	return security.SecurePath(basePath, userPath)
}

// WriteSecureFile writes data to a file with secure permissions
func WriteSecureFile(filePath string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := EnsureSecureDir(dir); err != nil {
		return err
	}
	
	// Write file with secure permissions
	if err := os.WriteFile(filePath, data, security.SecureFileMode); err != nil {
		return errors.WrapWithContext(err, "write file", filePath)
	}
	
	return nil
}

// CreateSecureFile creates a file with secure permissions and ensures the directory exists
func CreateSecureFile(filePath string) (*os.File, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := EnsureSecureDir(dir); err != nil {
		return nil, err
	}
	
	// Create file with secure permissions
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, security.SecureFileMode)
	if err != nil {
		return nil, errors.WrapWithContext(err, "create file", filePath)
	}
	
	return file, nil
}

// PathExists checks if a path exists
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsDirectory checks if a path exists and is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if a path exists and is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// SafeFileOperation performs a file operation with proper error handling
func SafeFileOperation(filePath string, operation func() error) error {
	if err := operation(); err != nil {
		return errors.WrapWithContext(err, "file operation", filePath)
	}
	return nil
}