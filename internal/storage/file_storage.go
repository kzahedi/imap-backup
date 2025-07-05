package storage

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"imap-backup/internal/errors"
	"imap-backup/internal/filesystem"
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
		return nil, errors.Wrap(err, "get folder path")
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
		return errors.WrapWithMessage(err, "invalid folder name")
	}
	
	folderPath, err := fs.getFolderPathWithDelimiter(folderName, delimiter)
	if err != nil {
		return errors.Wrap(err, "get secure folder path")
	}
	
	// Use secure directory permissions
	if err := filesystem.EnsureSecureDir(folderPath); err != nil {
		return err
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
		return errors.Wrap(err, "marshal metadata")
	}

	if err := filesystem.WriteSecureFile(metadataPath, metadataData); err != nil {
		return err
	}

	// Save raw message
	rawPath := filepath.Join(folderPath, fmt.Sprintf("%s.eml", baseFilename))
	if err := filesystem.WriteSecureFile(rawPath, msg.Raw); err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage) SaveAttachment(folderName string, msg *imap.Message, attachment imap.Attachment) error {
	return fs.SaveAttachmentWithDelimiter(folderName, "", msg, attachment)
}

func (fs *FileStorage) SaveAttachmentWithDelimiter(folderName, delimiter string, msg *imap.Message, attachment imap.Attachment) error {
	// Validate folder name for security
	if err := security.ValidateFolderName(folderName); err != nil {
		return errors.WrapWithMessage(err, "invalid folder name")
	}
	
	attachmentDir, err := fs.getAttachmentDirWithDelimiter(folderName, delimiter, msg)
	if err != nil {
		return errors.Wrap(err, "get secure attachment directory")
	}
	
	if err := filesystem.EnsureSecureDir(attachmentDir); err != nil {
		return err
	}

	// Sanitize filename
	filename := security.SanitizeFilename(attachment.Filename)
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

	if err := filesystem.WriteSecureFile(attachmentPath, attachment.Data); err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage) getFolderPath(folderName string) (string, error) {
	return fs.getFolderPathWithDelimiter(folderName, "")
}

func (fs *FileStorage) getFolderPathWithDelimiter(folderName, delimiter string) (string, error) {
	// Validate folder name first
	if err := security.ValidateFolderName(folderName); err != nil {
		return "", errors.WrapWithMessage(err, "invalid folder name")
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

func (fs *FileStorage) getAttachmentDir(folderName string, msg *imap.Message) (string, error) {
	return fs.getAttachmentDirWithDelimiter(folderName, "", msg)
}

func (fs *FileStorage) getAttachmentDirWithDelimiter(folderName, delimiter string, msg *imap.Message) (string, error) {
	// Build relative path for attachments using same naming as message files
	sanitizedFolder := security.SanitizeFolderName(folderName)
	if sanitizedFolder == "" {
		sanitizedFolder = "INBOX"
	}
	
	// Use the same filename format as messages for the attachment directory
	messageFilename := generateMessageFilename(msg)
	relativePath := filepath.Join(sanitizedFolder, "attachments", messageFilename)
	return security.SecurePath(fs.basePath, relativePath)
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
	return security.SanitizeFilename(baseFilename)
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
				return security.SanitizeForEmailName(name, 50)
			}
		}
	}
	
	// Try to extract name from just email address
	if strings.Contains(from, "@") {
		atIndex := strings.Index(from, "@")
		localPart := from[:atIndex]
		// Remove common prefixes and clean up
		localPart = strings.ReplaceAll(localPart, ".", "_")
		return security.SanitizeForEmailName(localPart, 50)
	}
	
	// If all else fails, use the whole string
	return security.SanitizeForEmailName(from, 50)
}

