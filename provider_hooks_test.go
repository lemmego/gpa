package gpa

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Test entity with hooks for provider integration testing
type TestUser struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate implements BeforeCreateHook
func (u *TestUser) BeforeCreate(ctx context.Context) error {
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	return nil
}

// AfterCreate implements AfterCreateHook
func (u *TestUser) AfterCreate(ctx context.Context) error {
	// This could log, send notifications, update cache, etc.
	return nil
}

// BeforeUpdate implements BeforeUpdateHook
func (u *TestUser) BeforeUpdate(ctx context.Context) error {
	u.UpdatedAt = time.Now()
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	return nil
}

// AfterUpdate implements AfterUpdateHook
func (u *TestUser) AfterUpdate(ctx context.Context) error {
	// This could log, send notifications, update cache, etc.
	return nil
}

// BeforeDelete implements BeforeDeleteHook
func (u *TestUser) BeforeDelete(ctx context.Context) error {
	// This could check permissions, log, etc.
	return nil
}

// AfterDelete implements AfterDeleteHook
func (u *TestUser) AfterDelete(ctx context.Context) error {
	// This could clean up related data, send notifications, etc.
	return nil
}

// AfterFind implements AfterFindHook
func (u *TestUser) AfterFind(ctx context.Context) error {
	// This could log access, update last accessed time, etc.
	return nil
}

// Validate implements ValidationHook
func (u *TestUser) Validate(ctx context.Context) error {
	if u.Email == "" {
		return NewError(ErrorTypeValidation, "email is required")
	}
	if !strings.Contains(u.Email, "@") {
		return NewError(ErrorTypeValidation, "email must contain @")
	}
	if u.Name == "" {
		return NewError(ErrorTypeValidation, "name is required")
	}
	return nil
}

// TestEntityHookInterfaces verifies that TestUser implements all hook interfaces
func TestEntityHookInterfaces(t *testing.T) {
	var user interface{} = &TestUser{}
	
	// Test that TestUser implements all hook interfaces
	if _, ok := user.(BeforeCreateHook); !ok {
		t.Error("TestUser should implement BeforeCreateHook")
	}
	if _, ok := user.(AfterCreateHook); !ok {
		t.Error("TestUser should implement AfterCreateHook")
	}
	if _, ok := user.(BeforeUpdateHook); !ok {
		t.Error("TestUser should implement BeforeUpdateHook")
	}
	if _, ok := user.(AfterUpdateHook); !ok {
		t.Error("TestUser should implement AfterUpdateHook")
	}
	if _, ok := user.(BeforeDeleteHook); !ok {
		t.Error("TestUser should implement BeforeDeleteHook")
	}
	if _, ok := user.(AfterDeleteHook); !ok {
		t.Error("TestUser should implement AfterDeleteHook")
	}
	if _, ok := user.(AfterFindHook); !ok {
		t.Error("TestUser should implement AfterFindHook")
	}
	if _, ok := user.(ValidationHook); !ok {
		t.Error("TestUser should implement ValidationHook")
	}
}

// TestEntityHooksExecution tests that hooks are executed correctly
func TestEntityHooksExecution(t *testing.T) {
	ctx := context.Background()
	user := &TestUser{
		Email: "  JOHN@EXAMPLE.COM  ",
		Name:  "John Doe",
	}
	
	// Test BeforeCreate hook
	if hook, ok := any(user).(BeforeCreateHook); ok {
		err := hook.BeforeCreate(ctx)
		if err != nil {
			t.Errorf("BeforeCreate hook failed: %v", err)
		}
		
		// Check that email was normalized
		if user.Email != "john@example.com" {
			t.Errorf("Expected email to be normalized to 'john@example.com', got: %s", user.Email)
		}
		
		// Check that timestamps were set
		if user.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set by BeforeCreate hook")
		}
		if user.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be set by BeforeCreate hook")
		}
	}
	
	// Test AfterCreate hook
	if hook, ok := any(user).(AfterCreateHook); ok {
		err := hook.AfterCreate(ctx)
		if err != nil {
			t.Errorf("AfterCreate hook failed: %v", err)
		}
	}
	
	// Test validation
	if hook, ok := any(user).(ValidationHook); ok {
		err := hook.Validate(ctx)
		if err != nil {
			t.Errorf("Validation failed: %v", err)
		}
	}
	
	// Test validation failure
	invalidUser := &TestUser{Name: "No Email"}
	if hook, ok := any(invalidUser).(ValidationHook); ok {
		err := hook.Validate(ctx)
		if err == nil {
			t.Error("Expected validation to fail for user without email")
		}
	}
}

// TestHookTypeAssertions tests the type assertion pattern used in providers
func TestHookTypeAssertions(t *testing.T) {
	ctx := context.Background()
	
	// Test with entity that has hooks
	userWithHooks := &TestUser{Email: "test@example.com", Name: "Test User"}
	
	// Test BeforeCreate assertion
	if hook, ok := any(userWithHooks).(BeforeCreateHook); ok {
		err := hook.BeforeCreate(ctx)
		if err != nil {
			t.Errorf("BeforeCreate hook failed: %v", err)
		}
	} else {
		t.Error("Expected TestUser to implement BeforeCreateHook")
	}
	
	// Test with entity that doesn't have hooks
	type SimpleEntity struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	
	simpleEntity := &SimpleEntity{ID: 1, Name: "Simple"}
	
	// Test BeforeCreate assertion (should not implement)
	if hook, ok := any(simpleEntity).(BeforeCreateHook); ok {
		t.Error("SimpleEntity should not implement BeforeCreateHook")
		_ = hook // avoid unused variable
	}
	
	// Test AfterFind assertion (should not implement)
	if hook, ok := any(simpleEntity).(AfterFindHook); ok {
		t.Error("SimpleEntity should not implement AfterFindHook")
		_ = hook // avoid unused variable
	}
}

// TestHookErrorHandling tests error handling in hooks
func TestHookErrorHandling(t *testing.T) {
	ctx := context.Background()
	
	// Test validation error
	invalidUser := &TestUser{
		Email: "invalid-email", // no @ symbol
		Name:  "Test User",
	}
	
	if hook, ok := any(invalidUser).(ValidationHook); ok {
		err := hook.Validate(ctx)
		if err == nil {
			t.Error("Expected validation to fail for invalid email")
		}
		if !strings.Contains(err.Error(), "email must contain @") {
			t.Errorf("Expected email validation error, got: %v", err)
		}
	}
	
	// Test empty name validation
	emptyNameUser := &TestUser{
		Email: "test@example.com",
		Name:  "", // empty name
	}
	
	if hook, ok := any(emptyNameUser).(ValidationHook); ok {
		err := hook.Validate(ctx)
		if err == nil {
			t.Error("Expected validation to fail for empty name")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Errorf("Expected name validation error, got: %v", err)
		}
	}
}

// TestHookChaining tests that multiple hooks can be chained together
func TestHookChaining(t *testing.T) {
	ctx := context.Background()
	user := &TestUser{
		Email: "  MIXED@CASE.COM  ",
		Name:  "Test User",
	}
	
	// Simulate the provider calling hooks in sequence
	// 1. BeforeCreate hook
	if hook, ok := any(user).(BeforeCreateHook); ok {
		err := hook.BeforeCreate(ctx)
		if err != nil {
			t.Errorf("BeforeCreate hook failed: %v", err)
		}
	}
	
	// 2. Validation hook
	if hook, ok := any(user).(ValidationHook); ok {
		err := hook.Validate(ctx)
		if err != nil {
			t.Errorf("Validation hook failed: %v", err)
		}
	}
	
	// 3. AfterCreate hook
	if hook, ok := any(user).(AfterCreateHook); ok {
		err := hook.AfterCreate(ctx)
		if err != nil {
			t.Errorf("AfterCreate hook failed: %v", err)
		}
	}
	
	// Verify that the BeforeCreate hook normalized the email
	if user.Email != "mixed@case.com" {
		t.Errorf("Expected email to be normalized to 'mixed@case.com', got: %s", user.Email)
	}
	
	// Verify that timestamps were set
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Error("Timestamps should be set by BeforeCreate hook")
	}
}