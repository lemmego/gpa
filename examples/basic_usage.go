package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpagorm"
)

// User represents a user in our application
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"size:255;not null"`
	Email    string `gorm:"uniqueIndex;size:255;not null"`
	Age      int    `gorm:"not null"`
	IsActive bool   `gorm:"default:true"`
}

func RunBasicUsage() {
	// Configure the database connection
	config := gpa.Config{
		Driver:   "sqlite",
		Database: "example.db",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "info",
			},
		},
	}

	// Create a type-safe provider
	provider, err := gpagorm.NewTypeSafeProvider[User](config)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Get a type-safe repository
	repo := provider.Repository()

	// Create the table (auto-migration)
	if sqlRepo, ok := repo.(gpa.MigratableRepository[User]); ok {
		err = sqlRepo.MigrateTable(context.Background())
		if err != nil {
			log.Fatalf("Failed to migrate table: %v", err)
		}
		fmt.Println("✓ Table migrated successfully")
	}

	ctx := context.Background()

	// Example 1: Create a new user
	fmt.Println("\n=== Creating Users ===")
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      30,
		IsActive: true,
	}

	err = repo.Create(ctx, user)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
	} else {
		fmt.Printf("✓ Created user: %+v\n", user)
	}

	// Example 2: Create multiple users
	users := []*User{
		{Name: "Alice Smith", Email: "alice@example.com", Age: 25, IsActive: true},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, IsActive: false},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 28, IsActive: true},
	}

	err = repo.CreateBatch(ctx, users)
	if err != nil {
		log.Printf("Failed to create users batch: %v", err)
	} else {
		fmt.Printf("✓ Created %d users in batch\n", len(users))
	}

	// Example 3: Find user by ID
	fmt.Println("\n=== Finding Users ===")
	foundUser, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to find user: %v", err)
	} else {
		fmt.Printf("✓ Found user by ID: %+v\n", foundUser)
	}

	// Example 4: Find all users
	allUsers, err := repo.FindAll(ctx)
	if err != nil {
		log.Printf("Failed to find all users: %v", err)
	} else {
		fmt.Printf("✓ Found %d total users\n", len(allUsers))
	}

	// Example 5: Query with conditions
	fmt.Println("\n=== Querying with Conditions ===")
	activeUsers, err := repo.Query(ctx,
		gpa.Where("is_active", gpa.OpEqual, true),
		gpa.Where("age", gpa.OpGreaterThan, 25),
		gpa.OrderBy("name", gpa.OrderAsc),
	)
	if err != nil {
		log.Printf("Failed to query active users: %v", err)
	} else {
		fmt.Printf("✓ Found %d active users over 25:\n", len(activeUsers))
		for _, u := range activeUsers {
			fmt.Printf("  - %s (age %d)\n", u.Name, u.Age)
		}
	}

	// Example 6: Count users
	fmt.Println("\n=== Counting and Aggregation ===")
	totalCount, err := repo.Count(ctx)
	if err != nil {
		log.Printf("Failed to count users: %v", err)
	} else {
		fmt.Printf("✓ Total users: %d\n", totalCount)
	}

	activeCount, err := repo.Count(ctx, gpa.Where("is_active", gpa.OpEqual, true))
	if err != nil {
		log.Printf("Failed to count active users: %v", err)
	} else {
		fmt.Printf("✓ Active users: %d\n", activeCount)
	}

	// Example 7: Update user
	fmt.Println("\n=== Updating Users ===")
	user.Age = 31
	user.Name = "John Smith"
	err = repo.Update(ctx, user)
	if err != nil {
		log.Printf("Failed to update user: %v", err)
	} else {
		fmt.Printf("✓ Updated user: %+v\n", user)
	}

	// Example 8: Partial update
	err = repo.UpdatePartial(ctx, user.ID, map[string]interface{}{
		"is_active": false,
	})
	if err != nil {
		log.Printf("Failed to partial update: %v", err)
	} else {
		fmt.Println("✓ Partially updated user (set inactive)")
	}

	// Example 9: Transaction
	fmt.Println("\n=== Transaction Example ===")
	err = repo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
		// Create a new user
		newUser := &User{
			Name:     "Transaction User",
			Email:    "tx@example.com",
			Age:      40,
			IsActive: true,
		}
		if err := tx.Create(ctx, newUser); err != nil {
			return err
		}

		// Update another user
		if err := tx.UpdatePartial(ctx, user.ID, map[string]interface{}{
			"age": 32,
		}); err != nil {
			return err
		}

		fmt.Println("✓ Transaction completed successfully")
		return nil
	})
	if err != nil {
		log.Printf("Transaction failed: %v", err)
	}

	// Example 10: Check if user exists
	fmt.Println("\n=== Existence Checks ===")
	exists, err := repo.Exists(ctx, gpa.Where("email", gpa.OpEqual, "john@example.com"))
	if err != nil {
		log.Printf("Failed to check existence: %v", err)
	} else {
		fmt.Printf("✓ User with email john@example.com exists: %t\n", exists)
	}

	// Example 11: Raw SQL (if supported)
	if sqlRepo, ok := repo.(gpa.SQLRepository[User]); ok {
		fmt.Println("\n=== Raw SQL Query ===")
		rawUsers, err := sqlRepo.FindBySQL(ctx, "SELECT * FROM users WHERE age > ? ORDER BY name", []interface{}{25})
		if err != nil {
			log.Printf("Failed to execute raw SQL: %v", err)
		} else {
			fmt.Printf("✓ Raw SQL found %d users\n", len(rawUsers))
		}
	}

	// Example 12: Clean up - delete a user
	fmt.Println("\n=== Cleanup ===")
	err = repo.Delete(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to delete user: %v", err)
	} else {
		fmt.Printf("✓ Deleted user with ID %d\n", user.ID)
	}

	// Final count
	finalCount, _ := repo.Count(ctx)
	fmt.Printf("✓ Final user count: %d\n", finalCount)

	fmt.Println("\n🎉 Basic usage example completed!")
}