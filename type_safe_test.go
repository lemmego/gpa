package gpa_test

import (
	"context"
	"testing"

	"github.com/lemmego/gpa"
)

// TestUser represents a test entity for generic testing
type TestUser struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (u TestUser) TableName() string { return "test_users" }

// TestGenericRepository demonstrates the type-safe generic repository interface
func TestGenericRepository(t *testing.T) {
	// Skip this test since it requires a database setup
	t.Skip("Skipping generic repository test - requires database setup")
	
	// This is how the generic interface would be used:
	_ = context.Background()
	
	// Example 1: Using generic repository directly (pseudocode)
	// In practice, you would create the generic repository like this:
	// userRepo := gpagorm.NewRepositoryG[TestUser](db, provider)
	// userRepo := gpabun.NewRepositoryG[TestUser](db, provider)  
	// userRepo := gpamongo.NewRepositoryG[TestUser](collection, provider)
	// userRepo := gparedis.NewRepositoryG[TestUser](provider, client, "user:")
	
	// Example 2: Demonstrating type-safe operations (pseudocode)
	// Create a user with compile-time type safety
	user := &TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	
	// With generics, all operations are type-safe:
	// err := userRepo.Create(ctx, user)                    // *TestUser in, no casting
	// foundUser, err := userRepo.FindByID(ctx, user.ID)   // *TestUser out, no casting
	// users, err := userRepo.Query(ctx, opts...)          // []*TestUser out, no casting
	// count, err := userRepo.Count(ctx, opts...)          // Always int64
	// exists, err := userRepo.Exists(ctx, opts...)        // Always bool
	
	// Example 2: Compare with traditional approach (pseudocode)
	// Traditional approach requires type assertions:
	// var provider gpa.Provider
	// traditionalRepo := provider.RepositoryFor(&TestUser{})
	// var foundUser TestUser
	// err := traditionalRepo.FindByID(ctx, user.ID, &foundUser)  // Must pass pointer
	// var users []TestUser  
	// err = traditionalRepo.Query(ctx, &users, opts...)          // Must pass pointer
	
	_ = user // Suppress unused variable warning
	_ = gpa.OpEqual // Reference gpa package to avoid import error
}

// TestGenericKeyValueRepository demonstrates the type-safe KV operations
func TestGenericKeyValueRepository(t *testing.T) {
	// This demonstrates how type-safe KV operations would work
	t.Run("BasicKeyValueRepositoryG usage example", func(t *testing.T) {
		// This is conceptual - shows the interface usage
		// var kvRepo gpa.BasicKeyValueRepositoryG[TestUser]
		// ctx := context.Background()
		
		// Type-safe operations:
		// user, err := kvRepo.Get(ctx, "user:123")  // Returns *TestUser
		// err = kvRepo.Set(ctx, "user:123", &user)  // Accepts *TestUser
		// users, err := kvRepo.MGet(ctx, []string{"user:1", "user:2"})  // Returns []*TestUser
		
		// No more runtime type assertions or interface{} parameters!
	})
	
	t.Run("AdvancedKeyValueRepositoryG usage example", func(t *testing.T) {
		// This demonstrates the full type-safe KV interface
		// var advRepo gpa.AdvancedKeyValueRepositoryG[TestUser]
		// ctx := context.Background()
		
		// All operations are type-safe:
		// user, err := advRepo.Get(ctx, "user:123")
		// err = advRepo.SetWithTTL(ctx, "session:abc", &user, 30*time.Minute)
		// users, err := advRepo.MGet(ctx, []string{"user:1", "user:2"})
		// count, err := advRepo.Increment(ctx, "user:count", 1)
		// keys, err := advRepo.Keys(ctx, "user:*")
	})
}

// TestGenericVsNonGeneric compares the old and new approaches
func TestGenericVsNonGeneric(t *testing.T) {
	t.Run("Traditional approach with interface{}", func(t *testing.T) {
		// Old way (still supported for backward compatibility):
		// var repo gpa.Repository
		// ctx := context.Background()
		// 
		// var users []TestUser
		// err := repo.FindAll(ctx, &users, gpa.Where("active", "=", true))
		// 
		// var user TestUser  
		// err = repo.FindByID(ctx, 123, &user)  // Need dest parameter
		//
		// Requires runtime type checking and potential panics
	})
	
	t.Run("New generic approach", func(t *testing.T) {
		// New way with compile-time type safety:
		// var repo gpa.RepositoryG[TestUser] 
		// ctx := context.Background()
		//
		// users, err := repo.FindAll(ctx, gpa.Where("active", "=", true))  // Returns []*TestUser
		// user, err := repo.FindByID(ctx, 123)  // Returns *TestUser directly
		//
		// Compile-time type safety, no runtime assertions, better performance
	})
}

// BenchmarkGenericVsTraditional shows performance comparison
func BenchmarkGenericVsTraditional(b *testing.B) {
	b.Skip("Benchmark skipped - requires database setup")
	
	// The generic approach should be faster because:
	// 1. No runtime type assertions
	// 2. No reflection for interface{} parameters  
	// 3. Direct type conversions
	// 4. Better compiler optimizations
}