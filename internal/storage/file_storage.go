package storage

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"imap-backup/internal/imap"
	"os"
	"path/filepath"
	"strconv"
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
	folderPath := fs.getFolderPathWithDelimiter(folderName, delimiter)
	uids := make(map[uint32]bool)

	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return uids, nil
	}

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		// Extract UID from filename
		filename := strings.TrimSuffix(info.Name(), ".json")
		uid, err := strconv.ParseUint(filename, 10, 32)
		if err != nil {
			return nil // Skip files that don't follow our naming convention
		}

		uids[uint32(uid)] = true
		return nil
	})

	return uids, err
}

func (fs *FileStorage) SaveMessage(folderName string, msg *imap.Message) error {
	return fs.SaveMessageWithDelimiter(folderName, "", msg)
}

func (fs *FileStorage) SaveMessageWithDelimiter(folderName, delimiter string, msg *imap.Message) error {
	folderPath := fs.getFolderPathWithDelimiter(folderName, delimiter)
	if err := os.MkdirAll(folderPath, 0755); err != nil {
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

	// Save metadata
	metadataPath := filepath.Join(folderPath, fmt.Sprintf("%d.json", msg.UID))
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Save raw message
	rawPath := filepath.Join(folderPath, fmt.Sprintf("%d.eml", msg.UID))
	if err := os.WriteFile(rawPath, msg.Raw, 0644); err != nil {
		return fmt.Errorf("failed to write raw message: %w", err)
	}

	return nil
}

func (fs *FileStorage) SaveAttachment(folderName string, messageUID uint32, attachment imap.Attachment) error {
	return fs.SaveAttachmentWithDelimiter(folderName, "", messageUID, attachment)
}

func (fs *FileStorage) SaveAttachmentWithDelimiter(folderName, delimiter string, messageUID uint32, attachment imap.Attachment) error {
	attachmentDir := fs.getAttachmentDirWithDelimiter(folderName, delimiter, messageUID)
	if err := os.MkdirAll(attachmentDir, 0755); err != nil {
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

	if err := os.WriteFile(attachmentPath, attachment.Data, 0644); err != nil {
		return fmt.Errorf("failed to write attachment: %w", err)
	}

	return nil
}

func (fs *FileStorage) getFolderPath(folderName string) string {
	return fs.getFolderPathWithDelimiter(folderName, "")
}

func (fs *FileStorage) getFolderPathWithDelimiter(folderName, delimiter string) string {
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
			sanitized := sanitizeFolderName(folderName)
			if sanitized == "" {
				return filepath.Join(fs.basePath, "INBOX")
			}
			return filepath.Join(fs.basePath, sanitized)
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
		sanitized := sanitizeFolderName(component)
		if sanitized != "" {
			sanitizedComponents = append(sanitizedComponents, sanitized)
		}
	}
	
	// Build the final path
	if len(sanitizedComponents) == 0 {
		return filepath.Join(fs.basePath, "INBOX") // Default to INBOX if no valid components
	}
	
	return filepath.Join(fs.basePath, filepath.Join(sanitizedComponents...))
}

func (fs *FileStorage) getAttachmentDir(folderName string, messageUID uint32) string {
	return fs.getAttachmentDirWithDelimiter(folderName, "", messageUID)
}

func (fs *FileStorage) getAttachmentDirWithDelimiter(folderName, delimiter string, messageUID uint32) string {
	return filepath.Join(fs.getFolderPathWithDelimiter(folderName, delimiter), "attachments", fmt.Sprintf("%d", messageUID))
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

func sanitizeFolderName(folderName string) string {
	// Remove or replace characters that are problematic in folder names
	// More permissive than filename sanitization since we're creating directories
	replacements := map[string]string{
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
		// Don't replace / and \ here as they're handled in getFolderPath
	}

	result := strings.TrimSpace(folderName)
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Handle special folder names that might conflict with system folders
	if result == "." || result == ".." {
		result = "_" + result
	}

	// Truncate if too long (filesystem limit)
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}