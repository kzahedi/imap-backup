package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name    string
		path    string
		mode    os.FileMode
		wantErr bool
	}{
		{
			name:    "create new directory",
			path:    filepath.Join(tempDir, "new_dir"),
			mode:    0755,
			wantErr: false,
		},
		{
			name:    "create nested directory",
			path:    filepath.Join(tempDir, "nested", "deep", "dir"),
			mode:    0755,
			wantErr: false,
		},
		{
			name:    "existing directory",
			path:    tempDir, // Already exists
			mode:    0755,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureDir(tt.path, tt.mode)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Check that directory exists
				info, err := os.Stat(tt.path)
				if err != nil {
					t.Errorf("EnsureDir() created directory should exist: %v", err)
					return
				}
				
				if !info.IsDir() {
					t.Errorf("EnsureDir() should create a directory, got file")
				}
			}
		})
	}
}

func TestEnsureSecureDir(t *testing.T) {
	tempDir := t.TempDir()
	secureDir := filepath.Join(tempDir, "secure_dir")
	
	err := EnsureSecureDir(secureDir)
	if err != nil {
		t.Errorf("EnsureSecureDir() error = %v", err)
		return
	}
	
	// Check that directory exists
	info, err := os.Stat(secureDir)
	if err != nil {
		t.Errorf("EnsureSecureDir() created directory should exist: %v", err)
		return
	}
	
	if !info.IsDir() {
		t.Errorf("EnsureSecureDir() should create a directory, got file")
	}
	
	// Check permissions (on Unix-like systems)
	if info.Mode().Perm() != 0700 {
		t.Errorf("EnsureSecureDir() permissions = %o, want %o", info.Mode().Perm(), 0700)
	}
}

func TestSecureJoin(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name     string
		basePath string
		elements []string
		wantErr  bool
	}{
		{
			name:     "simple join",
			basePath: tempDir,
			elements: []string{"folder", "file.txt"},
			wantErr:  false,
		},
		{
			name:     "empty elements",
			basePath: tempDir,
			elements: []string{},
			wantErr:  false,
		},
		{
			name:     "single element",
			basePath: tempDir,
			elements: []string{"folder"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SecureJoin(tt.basePath, tt.elements...)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("SecureJoin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Should start with base path
				if !filepath.HasPrefix(result, tt.basePath) {
					t.Errorf("SecureJoin() result should start with base path")
				}
			}
		})
	}
}

func TestWriteSecureFile(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name     string
		filePath string
		data     []byte
		wantErr  bool
	}{
		{
			name:     "write to new file",
			filePath: filepath.Join(tempDir, "test.txt"),
			data:     []byte("test content"),
			wantErr:  false,
		},
		{
			name:     "write to nested path",
			filePath: filepath.Join(tempDir, "nested", "test.txt"),
			data:     []byte("nested content"),
			wantErr:  false,
		},
		{
			name:     "empty data",
			filePath: filepath.Join(tempDir, "empty.txt"),
			data:     []byte{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteSecureFile(tt.filePath, tt.data)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteSecureFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Check that file exists and has correct content
				content, err := os.ReadFile(tt.filePath)
				if err != nil {
					t.Errorf("WriteSecureFile() file should be readable: %v", err)
					return
				}
				
				if string(content) != string(tt.data) {
					t.Errorf("WriteSecureFile() content = %q, want %q", string(content), string(tt.data))
				}
				
				// Check permissions
				info, err := os.Stat(tt.filePath)
				if err != nil {
					t.Errorf("WriteSecureFile() stat error: %v", err)
					return
				}
				
				if info.Mode().Perm() != 0600 {
					t.Errorf("WriteSecureFile() permissions = %o, want %o", info.Mode().Perm(), 0600)
				}
			}
		})
	}
}

func TestCreateSecureFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "created.txt")
	
	file, err := CreateSecureFile(filePath)
	if err != nil {
		t.Errorf("CreateSecureFile() error = %v", err)
		return
	}
	defer file.Close()
	
	// Write some data
	_, err = file.WriteString("test content")
	if err != nil {
		t.Errorf("CreateSecureFile() write error = %v", err)
		return
	}
	
	file.Close()
	
	// Check that file exists and has correct permissions
	info, err := os.Stat(filePath)
	if err != nil {
		t.Errorf("CreateSecureFile() file should exist: %v", err)
		return
	}
	
	if info.Mode().Perm() != 0600 {
		t.Errorf("CreateSecureFile() permissions = %o, want %o", info.Mode().Perm(), 0600)
	}
}

func TestPathExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.txt")
	
	// Create a test file
	err := os.WriteFile(existingFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     existingFile,
			expected: true,
		},
		{
			name:     "existing directory",
			path:     tempDir,
			expected: true,
		},
		{
			name:     "non-existing path",
			path:     filepath.Join(tempDir, "nonexistent.txt"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PathExists(tt.path)
			if result != tt.expected {
				t.Errorf("PathExists() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	// Create a test file
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "directory",
			path:     tempDir,
			expected: true,
		},
		{
			name:     "file",
			path:     testFile,
			expected: false,
		},
		{
			name:     "non-existing path",
			path:     filepath.Join(tempDir, "nonexistent"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDirectory(tt.path)
			if result != tt.expected {
				t.Errorf("IsDirectory() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	// Create a test file
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "file",
			path:     testFile,
			expected: true,
		},
		{
			name:     "directory",
			path:     tempDir,
			expected: false,
		},
		{
			name:     "non-existing path",
			path:     filepath.Join(tempDir, "nonexistent.txt"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSafeFileOperation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	tests := []struct {
		name      string
		filePath  string
		operation func() error
		wantErr   bool
	}{
		{
			name:     "successful operation",
			filePath: testFile,
			operation: func() error {
				return os.WriteFile(testFile, []byte("test"), 0644)
			},
			wantErr: false,
		},
		{
			name:     "failing operation",
			filePath: testFile,
			operation: func() error {
				return os.ErrNotExist
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SafeFileOperation(tt.filePath, tt.operation)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeFileOperation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}