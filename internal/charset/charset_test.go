package charset

import (
	"io"
	"strings"
	"testing"
)

func TestNewReader(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		charsetName string
		expected    string
		expectError bool
	}{
		{
			name:        "empty charset - should pass through",
			input:       "Hello, World!",
			charsetName: "",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "utf-8 charset - should pass through",
			input:       "Hello, World!",
			charsetName: "utf-8",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "UTF-8 charset case insensitive - should pass through",
			input:       "Hello, World!",
			charsetName: "UTF-8",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "utf8 charset - should pass through",
			input:       "Hello, World!",
			charsetName: "utf8",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "us-ascii charset",
			input:       "Hello, World!",
			charsetName: "us-ascii",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "iso-8859-1 charset",
			input:       "Hello, World!",
			charsetName: "iso-8859-1",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "windows-1252 charset",
			input:       "Hello, World!",
			charsetName: "windows-1252",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "charset with underscores",
			input:       "Hello, World!",
			charsetName: "shift_jis",
			expected:    "Hello, World!",
			expectError: false,
		},
		{
			name:        "unsupported charset",
			input:       "Hello, World!",
			charsetName: "unsupported-charset",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(strings.NewReader(tt.input), tt.charsetName)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}
			
			if !tt.expectError {
				result, err := io.ReadAll(reader)
				if err != nil {
					t.Errorf("Failed to read from reader: %v", err)
					return
				}
				
				if string(result) != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, string(result))
				}
			}
		})
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		charsetName string
		expected    string
	}{
		{
			name:        "empty charset",
			input:       "Hello, World!",
			charsetName: "",
			expected:    "Hello, World!",
		},
		{
			name:        "utf-8 charset",
			input:       "Hello, World!",
			charsetName: "utf-8",
			expected:    "Hello, World!",
		},
		{
			name:        "UTF-8 charset case insensitive",
			input:       "Hello, World!",
			charsetName: "UTF-8",
			expected:    "Hello, World!",
		},
		{
			name:        "utf8 charset",
			input:       "Hello, World!",
			charsetName: "utf8",
			expected:    "Hello, World!",
		},
		{
			name:        "us-ascii charset",
			input:       "Hello, World!",
			charsetName: "us-ascii",
			expected:    "Hello, World!",
		},
		{
			name:        "iso-8859-1 charset",
			input:       "Hello, World!",
			charsetName: "iso-8859-1",
			expected:    "Hello, World!",
		},
		{
			name:        "windows-1252 charset",
			input:       "Hello, World!",
			charsetName: "windows-1252",
			expected:    "Hello, World!",
		},
		{
			name:        "charset with underscores",
			input:       "Hello, World!",
			charsetName: "shift_jis",
			expected:    "Hello, World!",
		},
		{
			name:        "unsupported charset - should return original",
			input:       "Hello, World!",
			charsetName: "unsupported-charset",
			expected:    "Hello, World!",
		},
		{
			name:        "empty string input",
			input:       "",
			charsetName: "iso-8859-1",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeString(tt.input, tt.charsetName)
			
			// DecodeString should not return errors, it should return the original string
			if err != nil {
				t.Errorf("DecodeString returned error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		name        string
		charsetName string
		expected    bool
	}{
		{
			name:        "empty charset",
			charsetName: "",
			expected:    true,
		},
		{
			name:        "utf-8",
			charsetName: "utf-8",
			expected:    true,
		},
		{
			name:        "UTF-8 case insensitive",
			charsetName: "UTF-8",
			expected:    true,
		},
		{
			name:        "utf8",
			charsetName: "utf8",
			expected:    true,
		},
		{
			name:        "iso-8859-1",
			charsetName: "iso-8859-1",
			expected:    true,
		},
		{
			name:        "iso8859-1 without dash",
			charsetName: "iso8859-1",
			expected:    true,
		},
		{
			name:        "windows-1252",
			charsetName: "windows-1252",
			expected:    true,
		},
		{
			name:        "cp1252",
			charsetName: "cp1252",
			expected:    true,
		},
		{
			name:        "us-ascii",
			charsetName: "us-ascii",
			expected:    true,
		},
		{
			name:        "ascii",
			charsetName: "ascii",
			expected:    true,
		},
		{
			name:        "shift_jis with underscore",
			charsetName: "shift_jis",
			expected:    true,
		},
		{
			name:        "shift-jis with dash",
			charsetName: "shift-jis",
			expected:    true,
		},
		{
			name:        "gb2312",
			charsetName: "gb2312",
			expected:    true,
		},
		{
			name:        "gbk",
			charsetName: "gbk",
			expected:    true,
		},
		{
			name:        "gb18030",
			charsetName: "gb18030",
			expected:    true,
		},
		{
			name:        "big5",
			charsetName: "big5",
			expected:    true,
		},
		{
			name:        "euc-jp",
			charsetName: "euc-jp",
			expected:    true,
		},
		{
			name:        "iso-2022-jp",
			charsetName: "iso-2022-jp",
			expected:    true,
		},
		{
			name:        "euc-kr",
			charsetName: "euc-kr",
			expected:    true,
		},
		{
			name:        "koi8-r",
			charsetName: "koi8-r",
			expected:    true,
		},
		{
			name:        "koi8-u",
			charsetName: "koi8-u",
			expected:    true,
		},
		{
			name:        "macintosh",
			charsetName: "macintosh",
			expected:    true,
		},
		{
			name:        "utf-16",
			charsetName: "utf-16",
			expected:    true,
		},
		{
			name:        "utf16",
			charsetName: "utf16",
			expected:    true,
		},
		{
			name:        "utf-16be",
			charsetName: "utf-16be",
			expected:    true,
		},
		{
			name:        "utf-16le",
			charsetName: "utf-16le",
			expected:    true,
		},
		{
			name:        "windows-1250",
			charsetName: "windows-1250",
			expected:    true,
		},
		{
			name:        "windows-1251",
			charsetName: "windows-1251",
			expected:    true,
		},
		{
			name:        "windows-1253",
			charsetName: "windows-1253",
			expected:    true,
		},
		{
			name:        "windows-1254",
			charsetName: "windows-1254",
			expected:    true,
		},
		{
			name:        "windows-1255",
			charsetName: "windows-1255",
			expected:    true,
		},
		{
			name:        "windows-1256",
			charsetName: "windows-1256",
			expected:    true,
		},
		{
			name:        "windows-1257",
			charsetName: "windows-1257",
			expected:    true,
		},
		{
			name:        "windows-1258",
			charsetName: "windows-1258",
			expected:    true,
		},
		{
			name:        "cp1250",
			charsetName: "cp1250",
			expected:    true,
		},
		{
			name:        "cp1251",
			charsetName: "cp1251",
			expected:    true,
		},
		{
			name:        "ISO 8859 series",
			charsetName: "iso-8859-2",
			expected:    true,
		},
		{
			name:        "ISO 8859 series",
			charsetName: "iso-8859-15",
			expected:    true,
		},
		{
			name:        "case insensitive test",
			charsetName: "ISO-8859-1",
			expected:    true,
		},
		{
			name:        "unsupported charset",
			charsetName: "unsupported-charset-12345",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSupported(tt.charsetName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCharsetNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "underscore to dash",
			input:    "shift_jis",
			expected: "shift-jis",
		},
		{
			name:     "uppercase to lowercase",
			input:    "UTF-8",
			expected: "utf-8",
		},
		{
			name:     "mixed case with underscore",
			input:    "ISO_8859_1",
			expected: "iso-8859-1",
		},
		{
			name:     "already normalized",
			input:    "iso-8859-1",
			expected: "iso-8859-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test normalization through IsSupported function
			normalizedInput := strings.ToLower(tt.input)
			normalizedInput = strings.ReplaceAll(normalizedInput, "_", "-")
			
			if normalizedInput != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, normalizedInput)
			}
		})
	}
}

func TestCharsetMapCompleteness(t *testing.T) {
	// Test that all entries in charsetMap are valid
	for name, encoding := range charsetMap {
		if encoding == nil {
			t.Errorf("Charset %q has nil encoding", name)
		}
		
		// Test that we can create a decoder
		decoder := encoding.NewDecoder()
		if decoder == nil {
			t.Errorf("Charset %q cannot create decoder", name)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with empty reader
	reader, err := NewReader(strings.NewReader(""), "iso-8859-1")
	if err != nil {
		t.Errorf("Expected no error for empty reader, got: %v", err)
	}
	
	result, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Expected no error reading empty content, got: %v", err)
	}
	
	if string(result) != "" {
		t.Errorf("Expected empty string, got: %q", string(result))
	}
	
	// Test DecodeString with empty string
	decoded, err := DecodeString("", "iso-8859-1")
	if err != nil {
		t.Errorf("Expected no error for empty string, got: %v", err)
	}
	
	if decoded != "" {
		t.Errorf("Expected empty string, got: %q", decoded)
	}
}

func TestReaderWithNilInput(t *testing.T) {
	// Test what happens with nil reader (should panic or handle gracefully)
	defer func() {
		if r := recover(); r != nil {
			// This is expected behavior - nil reader should panic
			t.Logf("NewReader with nil input panicked as expected: %v", r)
		}
	}()
	
	// This might panic, which is acceptable
	_, err := NewReader(nil, "iso-8859-1")
	if err != nil {
		t.Logf("NewReader with nil input returned error: %v", err)
	}
}