package errors

import (
	"fmt"
)

// Wrap wraps an error with an operation description
func Wrap(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s: %w", operation, err)
}

// WrapWithContext wraps an error with operation and context information
func WrapWithContext(err error, operation, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s for %s: %w", operation, context, err)
}

// WrapWithMessage wraps an error with a custom message
func WrapWithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// New creates a new error with formatted message
func New(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// NewOperation creates a new error for a failed operation
func NewOperation(operation string, details ...interface{}) error {
	if len(details) > 0 {
		return fmt.Errorf("failed to %s: %v", operation, details[0])
	}
	return fmt.Errorf("failed to %s", operation)
}

// NewValidation creates a new validation error
func NewValidation(field, reason string) error {
	return fmt.Errorf("validation failed for %s: %s", field, reason)
}

// NewConfiguration creates a new configuration error
func NewConfiguration(setting, reason string) error {
	return fmt.Errorf("configuration error for %s: %s", setting, reason)
}