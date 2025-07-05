package security

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Common character replacement maps for different sanitization needs
var (
	// FilenameCharMap defines characters that need to be replaced in filenames
	FilenameCharMap = map[string]string{
		"/":    "_",
		"\\":   "_",
		":":    "_",
		"*":    "_",
		"?":    "_",
		"\"":   "_",
		"<":    "_",
		">":    "_",
		"|":    "_",
		"\x00": "_", // null byte
		"\r":   "_", // carriage return
		"\n":   "_", // newline
	}

	// FolderCharMap defines characters that need to be replaced in folder names
	FolderCharMap = map[string]string{
		":":    "_",
		"*":    "_",
		"?":    "_",
		"\"":   "_",
		"<":    "_",
		">":    "_",
		"|":    "_",
		"\x00": "_", // null byte
	}

	// EmailCharMap defines characters that need to be replaced in email-derived names
	EmailCharMap = map[string]string{
		" ":  "_",
		".":  "_",
		",":  "_",
		";":  "_",
		"'":  "",
		"\"": "",
	}
)

// SanitizeString applies character replacements from the provided map
func SanitizeString(input string, charMap map[string]string) string {
	result := input
	for old, new := range charMap {
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

// SanitizeUTF8 converts invalid UTF-8 sequences to replacement characters
func SanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// Convert invalid UTF-8 to valid UTF-8 by replacing invalid sequences
	var result strings.Builder
	result.Grow(len(s))

	for i, w := 0, 0; i < len(s); i += w {
		r, width := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && width == 1 {
			// Invalid UTF-8 sequence, replace with underscore
			result.WriteRune('_')
			w = 1
		} else {
			result.WriteRune(r)
			w = width
		}
	}

	return result.String()
}

// SanitizeUnicodeChars removes or replaces problematic Unicode characters
func SanitizeUnicodeChars(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		switch {
		case r == 0xFEFF || r == 0x200B || r == 0x2060:
			// Remove BOM and invisible/zero-width characters
			continue
		case unicode.IsControl(r) && r != '\t':
			// Replace control characters (except tab) with underscore
			result.WriteRune('_')
		case unicode.Is(unicode.Cf, r):
			// Replace format characters with underscore (except BOM which we already handled)
			result.WriteRune('_')
		case unicode.Is(unicode.Cs, r):
			// Replace surrogate characters with underscore
			result.WriteRune('_')
		case unicode.Is(unicode.Co, r):
			// Replace private use characters with underscore
			result.WriteRune('_')
		default:
			// Keep the character
			result.WriteRune(r)
		}
	}

	return result.String()
}

// SanitizeFilename provides comprehensive filename sanitization
func SanitizeFilename(filename string) string {
	// First, ensure the filename is valid UTF-8
	filename = SanitizeUTF8(filename)

	// Handle empty filename
	if filename == "" {
		return "untitled"
	}

	// Apply character replacements
	result := SanitizeString(filename, FilenameCharMap)

	// Remove control characters and other problematic Unicode characters
	result = SanitizeUnicodeChars(result)

	// Remove leading/trailing dots and spaces (problematic on Windows)
	result = strings.Trim(result, ". ")

	// Ensure it's not empty or only underscores after sanitization
	if result == "" || strings.Trim(result, "_") == "" {
		result = "untitled"
	}

	// Truncate if too long (filesystem limit)
	if len(result) > 255 {
		ext := filepath.Ext(result)
		name := strings.TrimSuffix(result, ext)
		maxNameLen := 255 - len(ext)
		if maxNameLen < 1 {
			maxNameLen = 1
		}
		result = name[:maxNameLen] + ext
	}

	return result
}

// SanitizeForEmailName removes problematic characters for email-derived names
func SanitizeForEmailName(name string, maxLength int) string {
	// First sanitize encoding issues
	name = SanitizeUTF8(name)

	// Apply email-specific character replacements
	name = SanitizeString(name, EmailCharMap)

	// Check for empty before applying general sanitizer
	if name == "" || name == "_" || strings.Trim(name, "_") == "" {
		name = "Unknown"
	} else {
		// Apply the general filename sanitizer for additional safety
		name = SanitizeFilename(name)
		
		// Check again after sanitization in case it became "untitled"
		if name == "untitled" {
			name = "Unknown"
		}
	}

	// Limit length for filename component
	if maxLength > 0 && len(name) > maxLength {
		name = name[:maxLength]
	}

	return name
}