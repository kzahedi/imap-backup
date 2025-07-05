package errors

import (
	"errors"
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
			name:      "nil error returns nil",
			err:       nil,
			operation: "test operation",
			expected:  "",
		},
		{
			name:      "wraps error with operation",
			err:       errors.New("original error"),
			operation: "test operation",
			expected:  "failed to test operation: original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.err, tt.operation)
			if tt.expected == "" {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Error("expected error, got nil")
				} else if result.Error() != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result.Error())
				}
			}
		})
	}
}

func TestWrapStore(t *testing.T) {
	err := errors.New("connection failed")
	result := WrapStore(err, "create")
	expected := "failed to create account store: connection failed"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestWrapKeychain(t *testing.T) {
	err := errors.New("access denied")
	result := WrapKeychain(err, "retrieve")
	expected := "failed to retrieve password from keychain: access denied"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestWrapAccount(t *testing.T) {
	err := errors.New("not found")
	result := WrapAccount(err, "find", "gmail-account")
	expected := "failed to find account 'gmail-account': not found"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestWrapFile(t *testing.T) {
	err := errors.New("permission denied")
	result := WrapFile(err, "read", "config.json")
	expected := "failed to read file 'config.json': permission denied"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestWrapConnection(t *testing.T) {
	err := errors.New("timeout")
	result := WrapConnection(err, "establish", "imap.gmail.com")
	expected := "failed to establish connection to imap.gmail.com: timeout"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestWrapBackup(t *testing.T) {
	err := errors.New("disk full")
	result := WrapBackup(err, "save message")
	expected := "backup failed: save message: disk full"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestNewValidation(t *testing.T) {
	result := NewValidation("email", "invalid format")
	expected := "validation failed for email: invalid format"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestNewConfiguration(t *testing.T) {
	result := NewConfiguration("port", "out of range")
	expected := "configuration error for port: out of range"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

// Benchmark tests for performance
func BenchmarkWrap(b *testing.B) {
	err := errors.New("test error")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = Wrap(err, "test operation")
	}
}

func BenchmarkWrapStore(b *testing.B) {
	err := errors.New("test error")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = WrapStore(err, "create")
	}
}