package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		expected  string
	}{
		{
			name:      "wrap error",
			err:       errors.New("original error"),
			operation: "perform operation",
			expected:  "failed to perform operation: original error",
		},
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "perform operation",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.err, tt.operation)
			
			if tt.err == nil {
				if result != nil {
					t.Errorf("Wrap() with nil error should return nil, got %v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("Wrap() with non-nil error should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("Wrap() = %q, want %q", result.Error(), tt.expected)
			}
			
			// Test that the original error is wrapped
			if !errors.Is(result, tt.err) {
				t.Errorf("Wrap() should wrap the original error")
			}
		})
	}
}

func TestWrapWithContext(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		context   string
		expected  string
	}{
		{
			name:      "wrap with context",
			err:       errors.New("original error"),
			operation: "read file",
			context:   "/path/to/file",
			expected:  "failed to read file for /path/to/file: original error",
		},
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "read file",
			context:   "/path/to/file",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithContext(tt.err, tt.operation, tt.context)
			
			if tt.err == nil {
				if result != nil {
					t.Errorf("WrapWithContext() with nil error should return nil, got %v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("WrapWithContext() with non-nil error should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("WrapWithContext() = %q, want %q", result.Error(), tt.expected)
			}
			
			// Test that the original error is wrapped
			if !errors.Is(result, tt.err) {
				t.Errorf("WrapWithContext() should wrap the original error")
			}
		})
	}
}

func TestWrapWithMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		message  string
		expected string
	}{
		{
			name:     "wrap with custom message",
			err:      errors.New("original error"),
			message:  "custom message",
			expected: "custom message: original error",
		},
		{
			name:     "nil error returns nil",
			err:      nil,
			message:  "custom message",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithMessage(tt.err, tt.message)
			
			if tt.err == nil {
				if result != nil {
					t.Errorf("WrapWithMessage() with nil error should return nil, got %v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("WrapWithMessage() with non-nil error should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("WrapWithMessage() = %q, want %q", result.Error(), tt.expected)
			}
			
			// Test that the original error is wrapped
			if !errors.Is(result, tt.err) {
				t.Errorf("WrapWithMessage() should wrap the original error")
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple message",
			format:   "simple error",
			args:     nil,
			expected: "simple error",
		},
		{
			name:     "formatted message",
			format:   "error with %s and %d",
			args:     []interface{}{"string", 42},
			expected: "error with string and 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := New(tt.format, tt.args...)
			
			if result == nil {
				t.Errorf("New() should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("New() = %q, want %q", result.Error(), tt.expected)
			}
		})
	}
}

func TestNewOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		details   []interface{}
		expected  string
	}{
		{
			name:      "operation without details",
			operation: "connect to database",
			details:   nil,
			expected:  "failed to connect to database",
		},
		{
			name:      "operation with details",
			operation: "connect to database",
			details:   []interface{}{"connection timeout"},
			expected:  "failed to connect to database: connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewOperation(tt.operation, tt.details...)
			
			if result == nil {
				t.Errorf("NewOperation() should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("NewOperation() = %q, want %q", result.Error(), tt.expected)
			}
		})
	}
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		reason   string
		expected string
	}{
		{
			name:     "validation error",
			field:    "email",
			reason:   "invalid format",
			expected: "validation failed for email: invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewValidation(tt.field, tt.reason)
			
			if result == nil {
				t.Errorf("NewValidation() should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("NewValidation() = %q, want %q", result.Error(), tt.expected)
			}
		})
	}
}

func TestNewConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		setting  string
		reason   string
		expected string
	}{
		{
			name:     "configuration error",
			setting:  "database.host",
			reason:   "cannot be empty",
			expected: "configuration error for database.host: cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewConfiguration(tt.setting, tt.reason)
			
			if result == nil {
				t.Errorf("NewConfiguration() should not return nil")
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("NewConfiguration() = %q, want %q", result.Error(), tt.expected)
			}
		})
	}
}

// Test error unwrapping functionality
func TestErrorUnwrapping(t *testing.T) {
	originalErr := fmt.Errorf("database connection failed")
	wrappedErr := Wrap(originalErr, "initialize application")
	
	// Test that errors.Is works correctly
	if !errors.Is(wrappedErr, originalErr) {
		t.Errorf("Wrapped error should be detectable with errors.Is")
	}
	
	// Test that errors.Unwrap works correctly  
	if errors.Unwrap(wrappedErr) != originalErr {
		t.Errorf("Wrapped error should be unwrappable to original error")
	}
}