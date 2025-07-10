package gpa

import (
	"errors"
	"testing"
)

func TestGPAError(t *testing.T) {
	err := GPAError{
		Type:    ErrorTypeValidation,
		Message: "validation failed",
		Code:    "INVALID_EMAIL",
	}

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected error type validation, got %s", err.Type)
	}
	if err.Message != "validation failed" {
		t.Errorf("Expected message 'validation failed', got '%s'", err.Message)
	}
	if err.Code != "INVALID_EMAIL" {
		t.Errorf("Expected code 'INVALID_EMAIL', got '%s'", err.Code)
	}
}

func TestGPAErrorError(t *testing.T) {
	err := GPAError{
		Type:    ErrorTypeNotFound,
		Message: "user not found",
	}

	expected := "not_found: user not found"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestGPAErrorWithCause(t *testing.T) {
	cause := errors.New("database connection failed")
	err := GPAError{
		Type:    ErrorTypeConnection,
		Message: "failed to connect",
		Cause:   cause,
	}

	if err.Cause != cause {
		t.Error("Expected cause to be set")
	}

	expectedMsg := "connection: failed to connect (caused by: database connection failed)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestGPAErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	err := GPAError{
		Type:    ErrorTypeInternal,
		Message: "wrapped error",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Error("Expected unwrapped error to match original cause")
	}
}

func TestGPAErrorIs(t *testing.T) {
	err1 := GPAError{Type: ErrorTypeValidation, Message: "validation error"}
	err2 := GPAError{Type: ErrorTypeValidation, Message: "different validation error"}
	err3 := GPAError{Type: ErrorTypeNotFound, Message: "not found error"}

	if !errors.Is(err1, err2) {
		t.Error("Expected errors with same type to be equal")
	}

	if errors.Is(err1, err3) {
		t.Error("Expected errors with different types to not be equal")
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrorTypeValidation, "validation failed")

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected error type validation, got %s", err.Type)
	}
	if err.Message != "validation failed" {
		t.Errorf("Expected message 'validation failed', got '%s'", err.Message)
	}
	if err.Cause != nil {
		t.Error("Expected no cause for basic error")
	}
}

func TestNewErrorWithCause(t *testing.T) {
	cause := errors.New("original error")
	err := NewErrorWithCause(ErrorTypeInternal, "internal error", cause)

	if err.Type != ErrorTypeInternal {
		t.Errorf("Expected error type internal, got %s", err.Type)
	}
	if err.Message != "internal error" {
		t.Errorf("Expected message 'internal error', got '%s'", err.Message)
	}
	if err.Cause != cause {
		t.Error("Expected cause to be set")
	}
}

func TestNewErrorWithCode(t *testing.T) {
	err := NewErrorWithCode(ErrorTypeValidation, "validation failed", "INVALID_INPUT")

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected error type validation, got %s", err.Type)
	}
	if err.Message != "validation failed" {
		t.Errorf("Expected message 'validation failed', got '%s'", err.Message)
	}
	if err.Code != "INVALID_INPUT" {
		t.Errorf("Expected code 'INVALID_INPUT', got '%s'", err.Code)
	}
}

func TestIsErrorType(t *testing.T) {
	err := NewError(ErrorTypeValidation, "validation error")

	if !IsErrorType(err, ErrorTypeValidation) {
		t.Error("Expected IsErrorType to return true for matching type")
	}

	if IsErrorType(err, ErrorTypeNotFound) {
		t.Error("Expected IsErrorType to return false for non-matching type")
	}

	if IsErrorType(errors.New("regular error"), ErrorTypeValidation) {
		t.Error("Expected IsErrorType to return false for non-GPA error")
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := NewError(ErrorTypeNotFound, "not found")
	validationErr := NewError(ErrorTypeValidation, "validation error")
	regularErr := errors.New("regular error")

	if !IsNotFound(notFoundErr) {
		t.Error("Expected IsNotFound to return true for not found error")
	}

	if IsNotFound(validationErr) {
		t.Error("Expected IsNotFound to return false for validation error")
	}

	if IsNotFound(regularErr) {
		t.Error("Expected IsNotFound to return false for regular error")
	}
}

func TestIsDuplicate(t *testing.T) {
	duplicateErr := NewError(ErrorTypeDuplicate, "duplicate entry")
	validationErr := NewError(ErrorTypeValidation, "validation error")
	regularErr := errors.New("regular error")

	if !IsDuplicate(duplicateErr) {
		t.Error("Expected IsDuplicate to return true for duplicate error")
	}

	if IsDuplicate(validationErr) {
		t.Error("Expected IsDuplicate to return false for validation error")
	}

	if IsDuplicate(regularErr) {
		t.Error("Expected IsDuplicate to return false for regular error")
	}
}

func TestIsValidation(t *testing.T) {
	validationErr := NewError(ErrorTypeValidation, "validation error")
	notFoundErr := NewError(ErrorTypeNotFound, "not found")
	regularErr := errors.New("regular error")

	if !IsValidation(validationErr) {
		t.Error("Expected IsValidation to return true for validation error")
	}

	if IsValidation(notFoundErr) {
		t.Error("Expected IsValidation to return false for not found error")
	}

	if IsValidation(regularErr) {
		t.Error("Expected IsValidation to return false for regular error")
	}
}

func TestIsConnection(t *testing.T) {
	connectionErr := NewError(ErrorTypeConnection, "connection failed")
	validationErr := NewError(ErrorTypeValidation, "validation error")
	regularErr := errors.New("regular error")

	if !IsConnection(connectionErr) {
		t.Error("Expected IsConnection to return true for connection error")
	}

	if IsConnection(validationErr) {
		t.Error("Expected IsConnection to return false for validation error")
	}

	if IsConnection(regularErr) {
		t.Error("Expected IsConnection to return false for regular error")
	}
}

func TestIsTransaction(t *testing.T) {
	transactionErr := NewError(ErrorTypeTransaction, "transaction failed")
	validationErr := NewError(ErrorTypeValidation, "validation error")
	regularErr := errors.New("regular error")

	if !IsTransaction(transactionErr) {
		t.Error("Expected IsTransaction to return true for transaction error")
	}

	if IsTransaction(validationErr) {
		t.Error("Expected IsTransaction to return false for validation error")
	}

	if IsTransaction(regularErr) {
		t.Error("Expected IsTransaction to return false for regular error")
	}
}

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeValidation, "validation"},
		{ErrorTypeNotFound, "not_found"},
		{ErrorTypeDuplicate, "duplicate"},
		{ErrorTypeConnection, "connection"},
		{ErrorTypeTransaction, "transaction"},
		{ErrorTypeInternal, "internal"},
		{ErrorTypeConstraint, "constraint"},
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypePermission, "permission"},
		{ErrorTypeDatabase, "database"},
	}

	for _, tt := range tests {
		if string(tt.errorType) != tt.expected {
			t.Errorf("Expected %s to be '%s', got '%s'", tt.errorType, tt.expected, string(tt.errorType))
		}
	}
}

func TestChainedErrors(t *testing.T) {
	rootCause := errors.New("root cause")
	middleError := NewErrorWithCause(ErrorTypeConnection, "connection failed", rootCause)
	topError := NewErrorWithCause(ErrorTypeInternal, "internal error", middleError)

	// Test unwrap chain
	if !errors.Is(topError, middleError) {
		t.Error("Expected errors.Is to find middle error in chain")
	}

	if !errors.Is(topError, rootCause) {
		t.Error("Expected errors.Is to find root cause in chain")
	}

	// Test error message includes chain
	errorMsg := topError.Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message")
	}
}