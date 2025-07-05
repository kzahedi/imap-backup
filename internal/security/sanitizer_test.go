package security

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		charMap  map[string]string
		expected string
	}{
		{
			name:     "no replacements needed",
			input:    "normal_text",
			charMap:  FilenameCharMap,
			expected: "normal_text",
		},
		{
			name:     "filename characters",
			input:    "file:name*with?chars",
			charMap:  FilenameCharMap,
			expected: "file_name_with_chars",
		},
		{
			name:     "folder characters",
			input:    "folder:name*",
			charMap:  FolderCharMap,
			expected: "folder_name_",
		},
		{
			name:     "email characters",
			input:    "user name, test",
			charMap:  EmailCharMap,
			expected: "user_name__test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input, tt.charMap)
			if result != tt.expected {
				t.Errorf("SanitizeString() = %q, want %q", result, tt.expected)
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
		{
			name:     "attachment filename case",
			input:    "Bestelleingangsbest\xE4tigung.pdf",
			expected: "Bestelleingangsbest_tigung.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeUTF8(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeUTF8() = %q, want %q", result, tt.expected)
			}
			if !utf8.ValidString(result) {
				t.Errorf("SanitizeUTF8() result is not valid UTF-8: %q", result)
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
			name:     "invisible characters removed",
			input:    "Normal\u200Btext\u2060here",
			expected: "Normaltexthere",
		},
		{
			name:     "format characters replaced",
			input:    "Text\u00ADwith\u061Cformatting",
			expected: "Text_with_formatting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeUnicodeChars(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeUnicodeChars() = %q, want %q", result, tt.expected)
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
			name:     "filename with invalid UTF-8",
			input:    "Bestelleingangsbest\xE4tigung.pdf",
			expected: "Bestelleingangsbest_tigung.pdf",
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "untitled",
		},
		{
			name:     "filename with only dots and spaces",
			input:    "  ...  ",
			expected: "untitled",
		},
		{
			name:     "filename with BOM",
			input:    "\uFEFFdocument.pdf",
			expected: "document.pdf",
		},
		{
			name:     "valid German umlaut",
			input:    "Bestelleingangsbestätigung.pdf",
			expected: "Bestelleingangsbestätigung.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename() = %q, want %q", result, tt.expected)
			}
			if len(result) > 255 {
				t.Errorf("SanitizeFilename() result too long: %d chars", len(result))
			}
			if !utf8.ValidString(result) {
				t.Errorf("SanitizeFilename() result is not valid UTF-8: %q", result)
			}
		})
	}
}

func TestSanitizeForEmailName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "normal name",
			input:     "John Doe",
			maxLength: 50,
			expected:  "John_Doe",
		},
		{
			name:      "name with special chars",
			input:     "Max Müller",
			maxLength: 50,
			expected:  "Max_Müller",
		},
		{
			name:      "long name truncated",
			input:     strings.Repeat("a", 100),
			maxLength: 20,
			expected:  strings.Repeat("a", 20),
		},
		{
			name:      "name with encoding issues",
			input:     "Max M\xFCller",
			maxLength: 50,
			expected:  "Max_M_ller",
		},
		{
			name:      "empty name",
			input:     "",
			maxLength: 50,
			expected:  "Unknown",
		},
		{
			name:      "name becomes underscore only",
			input:     "_",
			maxLength: 50,
			expected:  "Unknown",
		},
		{
			name:      "no length limit",
			input:     "John Doe",
			maxLength: 0,
			expected:  "John_Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForEmailName(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("SanitizeForEmailName() = %q, want %q", result, tt.expected)
			}
			if tt.maxLength > 0 && len(result) > tt.maxLength {
				t.Errorf("SanitizeForEmailName() result too long: %d chars, max %d", len(result), tt.maxLength)
			}
			if !utf8.ValidString(result) {
				t.Errorf("SanitizeForEmailName() result is not valid UTF-8: %q", result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizeFilename(b *testing.B) {
	filename := "complex:file*name?with<many>problematic|chars.txt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeFilename(filename)
	}
}

func BenchmarkSanitizeUTF8(b *testing.B) {
	input := "Mixed valid and invalid \xFF\xFE UTF-8 \x80 content"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeUTF8(input)
	}
}

func BenchmarkSanitizeUnicodeChars(b *testing.B) {
	input := "Text with \u200B invisible \u2060 and \u001F control chars"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeUnicodeChars(input)
	}
}