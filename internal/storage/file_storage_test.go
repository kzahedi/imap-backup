package storage

import (
	"imap-backup/internal/imap"
	"imap-backup/internal/security"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestNewFileStorage(t *testing.T) {
	basePath := "/test/path"
	fs := NewFileStorage(basePath)
	
	if fs.basePath != basePath {
		t.Errorf("NewFileStorage() basePath = %v, want %v", fs.basePath, basePath)
	}
}

func TestGetFolderPathWithDelimiter(t *testing.T) {
	fs := NewFileStorage("/base")
	
	tests := []struct {
		name      string
		folder    string
		delimiter string
		wantPath  bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "simple folder",
			folder:    "INBOX",
			delimiter: "",
			wantPath:  true,
			wantErr:   false,
		},
		{
			name:      "folder with slash delimiter",
			folder:    "Work/Projects",
			delimiter: "/",
			wantPath:  true,
			wantErr:   false,
		},
		{
			name:      "folder with dot delimiter",
			folder:    "Work.Projects",
			delimiter: ".",
			wantPath:  true,
			wantErr:   false,
		},
		{
			name:      "invalid folder with path traversal",
			folder:    "../etc/passwd",
			delimiter: "/",
			wantPath:  false,
			wantErr:   true,
			errMsg:    "path traversal",
		},
		{
			name:      "empty folder name",
			folder:    "",
			delimiter: "",
			wantPath:  false,
			wantErr:   true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "reserved folder name",
			folder:    "CON",
			delimiter: "",
			wantPath:  false,
			wantErr:   true,
			errMsg:    "reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := fs.getFolderPathWithDelimiter(tt.folder, tt.delimiter)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("getFolderPathWithDelimiter() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("getFolderPathWithDelimiter() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("getFolderPathWithDelimiter() unexpected error = %v", err)
					return
				}
				if !tt.wantPath && path != "" {
					t.Errorf("getFolderPathWithDelimiter() expected empty path, got %v", path)
				}
				if tt.wantPath && path == "" {
					t.Errorf("getFolderPathWithDelimiter() expected non-empty path, got empty")
				}
				if tt.wantPath && !strings.HasPrefix(path, fs.basePath) {
					t.Errorf("getFolderPathWithDelimiter() path %v should start with base path %v", path, fs.basePath)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with problematic chars",
			input:    "file:name*with?chars<>.txt",
			expected: "file_name_with_chars__.txt",
		},
		{
			name:     "very long filename",
			input:    strings.Repeat("a", 300) + ".txt",
			expected: strings.Repeat("a", 251) + ".txt", // 255 total
		},
		{
			name:     "filename with path separators",
			input:    "folder/file.txt",
			expected: "folder_file.txt",
		},
		{
			name:     "filename with backslashes",
			input:    "folder\\file.txt",
			expected: "folder_file.txt",
		},
		{
			name:     "filename with invalid UTF-8 sequences",
			input:    "Bestelleingangsbest\xE4tigung.pdf", // Invalid UTF-8
			expected: "Bestelleingangsbest_tigung.pdf",
		},
		{
			name:     "filename with mixed encoding issues",
			input:    "file\xFF\xFEname\x00.txt", // Invalid UTF-8 and null byte
			expected: "file__name_.txt",
		},
		{
			name:     "filename with control characters",
			input:    "file\nname\r.txt",
			expected: "file_name_.txt",
		},
		{
			name:     "filename with Unicode control chars",
			input:    "file\u0001\u001Fname.txt", // Control characters
			expected: "file__name.txt",
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "untitled",
		},
		{
			name:     "filename with only spaces and dots",
			input:    "  ...  ",
			expected: "untitled",
		},
		{
			name:     "filename with BOM",
			input:    "\uFEFFdocument.pdf",
			expected: "document.pdf",
		},
		{
			name:     "German umlaut filename (valid UTF-8)",
			input:    "Bestelleingangsbestätigung.pdf",
			expected: "Bestelleingangsbestätigung.pdf",
		},
		{
			name:     "filename becomes empty after sanitization",
			input:    "\x00\x01\x02",
			expected: "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename() = %q, want %q", result, tt.expected)
			}
			if len(result) > 255 {
				t.Errorf("sanitizeFilename() result too long: %d chars", len(result))
			}
			// Ensure result is valid UTF-8
			if !utf8.ValidString(result) {
				t.Errorf("sanitizeFilename() result is not valid UTF-8: %q", result)
			}
		})
	}
}

func TestFileStorageIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "backup")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	fs := NewFileStorage(baseDir)

	// Test saving a message
	msg := &imap.Message{
		UID:     123,
		Subject: "Test Message",
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Date:    time.Now(),
		Flags:   []string{"\\Seen"},
		Headers: map[string][]string{
			"Content-Type": {"text/plain"},
		},
		Raw: []byte("Test message content"),
	}

	folderName := "TestFolder"
	err = fs.SaveMessage(folderName, msg)
	if err != nil {
		t.Fatalf("SaveMessage() error = %v", err)
	}

	// Check that files were created with new naming convention
	folderPath, err := fs.getFolderPath(folderName)
	if err != nil {
		t.Fatalf("getFolderPath() error = %v", err)
	}

	// List files in the folder to find our created files
	files, err := os.ReadDir(folderPath)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	var emlFound, jsonFound bool
	var emlPath string
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".eml") {
			emlFound = true
			emlPath = filepath.Join(folderPath, file.Name())
		}
		if strings.HasSuffix(file.Name(), ".json") {
			jsonFound = true
		}
	}

	if !emlFound {
		t.Errorf("EML file not created in folder: %v", folderPath)
	}

	if !jsonFound {
		t.Errorf("JSON file not created in folder: %v", folderPath)
	}

	// Test getting existing UIDs
	uids, err := fs.GetExistingUIDs(folderName)
	if err != nil {
		t.Fatalf("GetExistingUIDs() error = %v", err)
	}

	if !uids[123] {
		t.Errorf("GetExistingUIDs() should contain UID 123")
	}

	// Test file permissions
	info, err := os.Stat(emlPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	expectedMode := os.FileMode(0600)
	actualMode := info.Mode().Perm()
	if actualMode != expectedMode {
		t.Errorf("EML file permissions = %v, want %v", actualMode, expectedMode)
	}

	// Test directory permissions
	info, err = os.Stat(folderPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	expectedDirMode := os.FileMode(0700)
	actualDirMode := info.Mode().Perm()
	if actualDirMode != expectedDirMode {
		t.Errorf("Directory permissions = %v, want %v", actualDirMode, expectedDirMode)
	}
}

func TestSaveAttachment(t *testing.T) {
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "backup")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	fs := NewFileStorage(baseDir)

	attachment := imap.Attachment{
		Filename:    "test.pdf",
		ContentType: "application/pdf",
		Data:        []byte("fake PDF data"),
	}

	folderName := "TestFolder"
	msg := &imap.Message{
		UID:     456,
		Subject: "Test Message",
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Date:    time.Now(),
		Flags:   []string{"\\Seen"},
	}

	err = fs.SaveAttachment(folderName, msg, attachment)
	if err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	// Check that attachment file was created
	attachmentDir, err := fs.getAttachmentDir(folderName, msg)
	if err != nil {
		t.Fatalf("getAttachmentDir() error = %v", err)
	}

	attachmentPath := filepath.Join(attachmentDir, "test.pdf")
	if _, err := os.Stat(attachmentPath); os.IsNotExist(err) {
		t.Errorf("Attachment file not created: %v", attachmentPath)
	}

	// Check file content
	content, err := os.ReadFile(attachmentPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !reflect.DeepEqual(content, attachment.Data) {
		t.Errorf("Attachment content mismatch: got %v, want %v", content, attachment.Data)
	}

	// Test file permissions
	info, err := os.Stat(attachmentPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	expectedMode := os.FileMode(0600)
	actualMode := info.Mode().Perm()
	if actualMode != expectedMode {
		t.Errorf("Attachment file permissions = %v, want %v", actualMode, expectedMode)
	}
}

func TestGetExistingUIDsEmptyFolder(t *testing.T) {
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "backup")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	fs := NewFileStorage(baseDir)

	uids, err := fs.GetExistingUIDs("NonExistentFolder")
	if err != nil {
		t.Fatalf("GetExistingUIDs() error = %v", err)
	}

	if len(uids) != 0 {
		t.Errorf("GetExistingUIDs() should return empty map for non-existent folder, got %v", uids)
	}
}

func TestDuplicateAttachmentFilenames(t *testing.T) {
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "backup")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	fs := NewFileStorage(baseDir)

	folderName := "TestFolder"
	msg := &imap.Message{
		UID:     789,
		Subject: "Test Message with Attachments",
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Date:    time.Now(),
		Flags:   []string{"\\Seen"},
	}

	// Save first attachment
	attachment1 := imap.Attachment{
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Data:        []byte("first document"),
	}
	err = fs.SaveAttachment(folderName, msg, attachment1)
	if err != nil {
		t.Fatalf("SaveAttachment() first error = %v", err)
	}

	// Save second attachment with same filename
	attachment2 := imap.Attachment{
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Data:        []byte("second document"),
	}
	err = fs.SaveAttachment(folderName, msg, attachment2)
	if err != nil {
		t.Fatalf("SaveAttachment() second error = %v", err)
	}

	// Check that both files exist with different names
	attachmentDir, err := fs.getAttachmentDir(folderName, msg)
	if err != nil {
		t.Fatalf("getAttachmentDir() error = %v", err)
	}

	originalPath := filepath.Join(attachmentDir, "document.pdf")
	duplicatePath := filepath.Join(attachmentDir, "document_1.pdf")

	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		t.Errorf("Original attachment file not found: %v", originalPath)
	}

	if _, err := os.Stat(duplicatePath); os.IsNotExist(err) {
		t.Errorf("Duplicate attachment file not found: %v", duplicatePath)
	}

	// Verify content is different
	content1, _ := os.ReadFile(originalPath)
	content2, _ := os.ReadFile(duplicatePath)

	if reflect.DeepEqual(content1, content2) {
		t.Error("Duplicate attachments should have different content")
	}
}

func TestGenerateMessageFilename(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		date     time.Time
		expected string
	}{
		{
			name:     "sender with name and email",
			from:     "John Doe <john@example.com>",
			date:     time.Date(2024, 7, 4, 15, 30, 45, 0, time.UTC),
			expected: "John_Doe_2024-07-04_15_30_45",
		},
		{
			name:     "email only",
			from:     "jane.smith@example.com",
			date:     time.Date(2024, 7, 4, 15, 30, 45, 0, time.UTC),
			expected: "jane_smith_2024-07-04_15_30_45",
		},
		{
			name:     "sender with problematic characters",
			from:     "Bad/Name <bad*name@example.com>",
			date:     time.Date(2024, 7, 4, 15, 30, 45, 0, time.UTC),
			expected: "Bad_Name_2024-07-04_15_30_45",
		},
		{
			name:     "empty sender",
			from:     "",
			date:     time.Date(2024, 7, 4, 15, 30, 45, 0, time.UTC),
			expected: "Unknown_2024-07-04_15_30_45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &imap.Message{
				From: tt.from,
				Date: tt.date,
			}
			
			result := generateMessageFilename(msg)
			if result != tt.expected {
				t.Errorf("generateMessageFilename() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractSenderName(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		expected string
	}{
		{
			name:     "name with email",
			from:     "John Doe <john@example.com>",
			expected: "John_Doe",
		},
		{
			name:     "quoted name with email",
			from:     "\"Jane Smith\" <jane@example.com>",
			expected: "Jane_Smith",
		},
		{
			name:     "email only",
			from:     "user@example.com",
			expected: "user",
		},
		{
			name:     "email with dots",
			from:     "user.name@example.com",
			expected: "user_name",
		},
		{
			name:     "empty from",
			from:     "",
			expected: "Unknown",
		},
		{
			name:     "name with special chars",
			from:     "Bad/Name*Test <user@example.com>",
			expected: "Bad_Name_Test",
		},
		{
			name:     "name with encoding issues",
			from:     "Max M\xFCller <max@example.com>", // Invalid UTF-8
			expected: "Max_M_ller",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSenderName(tt.from)
			if result != tt.expected {
				t.Errorf("extractSenderName() = %q, want %q", result, tt.expected)
			}
			// Ensure result is valid UTF-8
			if !utf8.ValidString(result) {
				t.Errorf("extractSenderName() result is not valid UTF-8: %q", result)
			}
		})
	}
}

func TestSanitizeUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8",
			input:    "Hello, 世界! 🌍",
			expected: "Hello, 世界! 🌍",
		},
		{
			name:     "invalid UTF-8 byte sequences",
			input:    "Hello\xFF\xFEWorld",
			expected: "Hello__World",
		},
		{
			name:     "mixed valid and invalid",
			input:    "Café\x80\x81Münch\xFFen",
			expected: "Café__Münch_en",
		},
		{
			name:     "German umlaut with invalid bytes",
			input:    "M\xFCller", // Invalid UTF-8 for ü
			expected: "M_ller",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.SanitizeUTF8(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeUTF8() = %q, want %q", result, tt.expected)
			}
			if !utf8.ValidString(result) {
				t.Errorf("sanitizeUTF8() result is not valid UTF-8: %q", result)
			}
		})
	}
}

func TestSanitizeUnicodeChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "control characters",
			input:    "Hello\x01\x02World\x1F",
			expected: "Hello__World_",
		},
		{
			name:     "tab character preserved",
			input:    "Hello\tWorld",
			expected: "Hello\tWorld",
		},
		{
			name:     "BOM removed",
			input:    "\uFEFFHello",
			expected: "Hello",
		},
		{
			name:     "mixed Unicode categories",
			input:    "Normal\u200Btext\u2060here",
			expected: "Normaltexthere",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.SanitizeUnicodeChars(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeUnicodeChars() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizeFilename(b *testing.B) {
	filename := "complex:file*name?with<many>problematic|chars.txt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		security.SanitizeFilename(filename)
	}
}

func BenchmarkGenerateMessageFilename(b *testing.B) {
	msg := &imap.Message{
		From: "John Doe <john@example.com>",
		Date: time.Now(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateMessageFilename(msg)
	}
}

func BenchmarkGetFolderPath(b *testing.B) {
	fs := NewFileStorage("/base/path")
	folderName := "Work/Projects/MyProject/SubProject"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs.getFolderPath(folderName)
	}
}