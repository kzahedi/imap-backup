package storage

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"imap-backup/internal/imap"
	"imap-backup/internal/security"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileStorage struct {
	basePath string
}

type MessageMetadata struct {
	UID         uint32            `json:"uid"`
	Subject     string            `json:"subject"`
	From        string            `json:"from"`
	To          string            `json:"to"`
	Date        time.Time         `json:"date"`
	Flags       []string          `json:"flags"`
	Headers     map[string][]string `json:"headers"`
	Attachments []string          `json:"attachments"`
	Checksum    string            `json:"checksum"`
}

func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{
		basePath: basePath,
	}
}

func (fs *FileStorage) GetExistingUIDs(folderName string) (map[uint32]bool, error) {
	return fs.GetExistingUIDsWithDelimiter(folderName, "")
}

func (fs *FileStorage) GetExistingUIDsWithDelimiter(folderName, delimiter string) (map[uint32]bool, error) {
	folderPath, err := fs.getFolderPathWithDelimiter(folderName, delimiter)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder path: %w", err)
	}
	
	uids := make(map[uint32]bool)

	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return uids, nil
	}

	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		// Read JSON metadata to extract UID
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var metadata MessageMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return nil // Skip files with invalid JSON
		}

		uids[metadata.UID] = true
		return nil
	})

	return uids, err
}

func (fs *FileStorage) SaveMessage(folderName string, msg *imap.Message) error {
	return fs.SaveMessageWithDelimiter(folderName, "", msg)
}

func (fs *FileStorage) SaveMessageWithDelimiter(folderName, delimiter string, msg *imap.Message) error {
	// Validate folder name for security
	if err := security.ValidateFolderName(folderName); err != nil {
		return fmt.Errorf("invalid folder name: %w", err)
	}
	
	folderPath, err := fs.getFolderPathWithDelimiter(folderName, delimiter)
	if err != nil {
		return fmt.Errorf("failed to get secure folder path: %w", err)
	}
	
	// Use secure directory permissions
	if err := os.MkdirAll(folderPath, security.SecureDirectoryMode); err != nil {
		return fmt.Errorf("failed to create folder directory: %w", err)
	}

	// Calculate checksum for integrity
	checksum := fmt.Sprintf("%x", md5.Sum(msg.Raw))

	// Create metadata
	attachmentNames := make([]string, len(msg.Attachments))
	for i, att := range msg.Attachments {
		attachmentNames[i] = att.Filename
	}

	metadata := MessageMetadata{
		UID:         msg.UID,
		Subject:     msg.Subject,
		From:        msg.From,
		To:          msg.To,
		Date:        msg.Date,
		Flags:       msg.Flags,
		Headers:     msg.Headers,
		Attachments: attachmentNames,
		Checksum:    checksum,
	}

	// Generate unique filename based on sender and timestamp
	baseFilename := generateMessageFilename(msg)
	
	// Save metadata
	metadataPath := filepath.Join(folderPath, fmt.Sprintf("%s.json", baseFilename))
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, security.SecureFileMode); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Save raw message
	rawPath := filepath.Join(folderPath, fmt.Sprintf("%s.eml", baseFilename))
	if err := os.WriteFile(rawPath, msg.Raw, security.SecureFileMode); err != nil {
		return fmt.Errorf("failed to write raw message: %w", err)
	}

	return nil
}

func (fs *FileStorage) SaveAttachment(folderName string, messageUID uint32, attachment imap.Attachment) error {
	return fs.SaveAttachmentWithDelimiter(folderName, "", messageUID, attachment)
}

func (fs *FileStorage) SaveAttachmentWithDelimiter(folderName, delimiter string, messageUID uint32, attachment imap.Attachment) error {
	// Validate folder name for security
	if err := security.ValidateFolderName(folderName); err != nil {
		return fmt.Errorf("invalid folder name: %w", err)
	}
	
	attachmentDir, err := fs.getAttachmentDirWithDelimiter(folderName, delimiter, messageUID)
	if err != nil {
		return fmt.Errorf("failed to get secure attachment directory: %w", err)
	}
	
	if err := os.MkdirAll(attachmentDir, security.SecureDirectoryMode); err != nil {
		return fmt.Errorf("failed to create attachment directory: %w", err)
	}

	// Sanitize filename
	filename := sanitizeFilename(attachment.Filename)
	attachmentPath := filepath.Join(attachmentDir, filename)

	// Handle duplicate filenames
	counter := 1
	originalPath := attachmentPath
	for {
		if _, err := os.Stat(attachmentPath); os.IsNotExist(err) {
			break
		}
		
		ext := filepath.Ext(originalPath)
		name := strings.TrimSuffix(filepath.Base(originalPath), ext)
		attachmentPath = filepath.Join(attachmentDir, fmt.Sprintf("%s_%d%s", name, counter, ext))
		counter++
	}

	if err := os.WriteFile(attachmentPath, attachment.Data, security.SecureFileMode); err != nil {
		return fmt.Errorf("failed to write attachment: %w", err)
	}

	return nil
}

func (fs *FileStorage) getFolderPath(folderName string) (string, error) {
	return fs.getFolderPathWithDelimiter(folderName, "")
}

func (fs *FileStorage) getFolderPathWithDelimiter(folderName, delimiter string) (string, error) {
	// Validate folder name first
	if err := security.ValidateFolderName(folderName); err != nil {
		return "", fmt.Errorf("invalid folder name: %w", err)
	}
	
	// Use the actual IMAP delimiter if provided, otherwise try to detect
	if delimiter == "" {
		// Try common IMAP delimiters
		if strings.Contains(folderName, "/") {
			delimiter = "/"
		} else if strings.Contains(folderName, ".") {
			delimiter = "."
		} else if strings.Contains(folderName, "\\") {
			delimiter = "\\"
		} else {
			// No hierarchy detected, treat as single folder
			sanitized := security.SanitizeFolderName(folderName)
			if sanitized == "" {
				sanitized = "INBOX"
			}
			return security.SecurePath(fs.basePath, sanitized)
		}
	}
	
	// Split path using the correct delimiter
	pathComponents := strings.Split(folderName, delimiter)
	var sanitizedComponents []string
	
	for _, component := range pathComponents {
		if component == "" {
			continue // Skip empty components
		}
		// Sanitize each folder component for filesystem safety
		sanitized := security.SanitizeFolderName(component)
		if sanitized != "" {
			sanitizedComponents = append(sanitizedComponents, sanitized)
		}
	}
	
	// Build the final path
	if len(sanitizedComponents) == 0 {
		return security.SecurePath(fs.basePath, "INBOX")
	}
	
	userPath := filepath.Join(sanitizedComponents...)
	return security.SecurePath(fs.basePath, userPath)
}

func (fs *FileStorage) getAttachmentDir(folderName string, messageUID uint32) (string, error) {
	return fs.getAttachmentDirWithDelimiter(folderName, "", messageUID)
}

func (fs *FileStorage) getAttachmentDirWithDelimiter(folderName, delimiter string, messageUID uint32) (string, error) {
	// Build relative path for attachments
	sanitizedFolder := security.SanitizeFolderName(folderName)
	if sanitizedFolder == "" {
		sanitizedFolder = "INBOX"
	}
	
	relativePath := filepath.Join(sanitizedFolder, "attachments", fmt.Sprintf("%d", messageUID))
	return security.SecurePath(fs.basePath, relativePath)
}

func sanitizeFilename(filename string) string {
	// Remove or replace characters that are problematic in filenames
	replacements := map[string]string{
		"/":  "_",
		"\\": "_",
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
	}

	result := filename
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Truncate if too long
	if len(result) > 255 {
		ext := filepath.Ext(result)
		name := strings.TrimSuffix(result, ext)
		result = name[:255-len(ext)] + ext
	}

	return result
}

// generateMessageFilename creates a unique filename using sender and timestamp
// Format: <Sender Name>_YYYY-MM-DD_HH_MM_SS
func generateMessageFilename(msg *imap.Message) string {
	// Extract sender name from From field
	senderName := extractSenderName(msg.From)
	
	// Format timestamp
	timestamp := msg.Date.Format("2006-01-02_15_04_05")
	
	// Create base filename
	baseFilename := fmt.Sprintf("%s_%s", senderName, timestamp)
	
	// Sanitize the filename
	return sanitizeFilename(baseFilename)
}

// extractSenderName extracts a clean sender name from email address
func extractSenderName(from string) string {
	if from == "" {
		return "Unknown"
	}
	
	// Try to extract name from "Name <email@domain.com>" format
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		nameEnd := strings.Index(from, "<")
		if nameEnd > 0 {
			name := strings.TrimSpace(from[:nameEnd])
			// Remove quotes if present
			name = strings.Trim(name, "\"")
			if name != "" {
				return sanitizeForFilename(name)
			}
		}
	}
	
	// Try to extract name from just email address
	if strings.Contains(from, "@") {
		atIndex := strings.Index(from, "@")
		localPart := from[:atIndex]
		// Remove common prefixes and clean up
		localPart = strings.ReplaceAll(localPart, ".", "_")
		return sanitizeForFilename(localPart)
	}
	
	// If all else fails, use the whole string
	return sanitizeForFilename(from)
}

// sanitizeForFilename removes problematic characters for filenames
func sanitizeForFilename(name string) string {
	// Replace spaces and problematic characters
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, ",", "_")
	name = strings.ReplaceAll(name, ";", "_")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "\"", "")
	
	// Apply the general sanitizer
	name = sanitizeFilename(name)
	
	// Limit length for filename component
	if len(name) > 50 {
		name = name[:50]
	}
	
	// Ensure it's not empty
	if name == "" || name == "_" {
		name = "Unknown"
	}
	
	return name
}

// Note: sanitizeFolderName removed - now using security.SanitizeFolderName