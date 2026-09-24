package gpa

import (
	"errors"
	"fmt"
)

// =====================================
// Error Handling
// =====================================

// GPAError represents a GPA-specific error
type GPAError struct {
	Type    ErrorType
	Message string
	Cause   error
	Code    string
}

// Error implements the error interface
func (e GPAError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e GPAError) Unwrap() error {
	return e.Cause
}

// Is checks if the error is of a specific type
func (e GPAError) Is(target error) bool {
	if targetGPAError, ok := target.(GPAError); ok {
		return e.Type == targetGPAError.Type
	}
	return false
}

// NewError creates a new GPAError
func NewError(errorType ErrorType, message string) GPAError {
	return GPAError{
		Type:    errorType,
		Message: message,
	}
}

// NewErrorWithCause creates a new GPAError with a cause
func NewErrorWithCause(errorType ErrorType, message string, cause error) GPAError {
	return GPAError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// NewErrorWithCode creates a new GPAError with a code
func NewErrorWithCode(errorType ErrorType, message string, code string) GPAError {
	return GPAError{
		Type:    errorType,
		Message: message,
		Code:    code,
	}
}

// IsNotFound checks if an error is a "not found" error
func IsNotFound(err error) bool {
	return IsErrorType(err, ErrorTypeNotFound)
}

// IsDuplicate checks if an error is a "duplicate" error
func IsDuplicate(err error) bool {
	return IsErrorType(err, ErrorTypeDuplicate)
}

// IsValidation checks if an error is a "validation" error
func IsValidation(err error) bool {
	return IsErrorType(err, ErrorTypeValidation)
}

// IsConnection checks if an error is a "connection" error
func IsConnection(err error) bool {
	return IsErrorType(err, ErrorTypeConnection)
}

// IsTransaction checks if an error is a "transaction" error
func IsTransaction(err error) bool {
	return IsErrorType(err, ErrorTypeTransaction)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	var value GPAError
	if errors.As(err, &value) {
		return value.Type == errorType
	}

	var pointer *GPAError
	return errors.As(err, &pointer) && pointer != nil && pointer.Type == errorType
}
