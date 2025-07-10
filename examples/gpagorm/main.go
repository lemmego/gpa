package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpagorm"
)

// User represents a user entity with GORM-specific tags
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:255;not null"`
	Email     string    `gorm:"uniqueIndex;size:255;not null"`
	Age       int       `gorm:"not null"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	Profile   *Profile  `gorm:"foreignKey:UserID"`
}

// Profile represents a user profile with a foreign key relationship
type Profile struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"not null;index"`
	Bio      string `gorm:"type:text"`
	Website  string `gorm:"size:255"`
	Location string `gorm:"size:100"`
}

// Post represents a blog post
type Post struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	Title     string    `gorm:"size:255;not null"`
	Content   string    `gorm:"type:text"`
	Published bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	User      *User     `gorm:"foreignKey:UserID"`
}

func main() {
	fmt.Println("🔧 GORM Provider Example")
	fmt.Println("Demonstrating GORM-specific features and SQL operations")

	// Configure GORM with SQLite in-memory database
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level":      "info",
				"singular_table": false,
			},
		},
	}

	// Create a single provider using the new unified API
	provider, err := gpagorm.NewProvider(config)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create multiple repositories from the same provider using the new unified API
	userRepo := gpagorm.GetRepository[User](provider)
	profileRepo := gpagorm.GetRepository[Profile](provider)
	postRepo := gpagorm.GetRepository[Post](provider)

	ctx := context.Background()

	// ============================================
	// Schema Migration
	// ============================================
	fmt.Println("\n=== Schema Migration ===")

	if migratableUserRepo, ok := userRepo.(gpa.MigratableRepository[User]); ok {
		err = migratableUserRepo.MigrateTable(ctx)
		if err != nil {
			log.Fatalf("Failed to migrate users table: %v", err)
		}
		fmt.Println("✓ Users table migrated")

		// Get migration status
		status, err := migratableUserRepo.GetMigrationStatus(ctx)
		if err != nil {
			log.Printf("Failed to get migration status: %v", err)
		} else {
			fmt.Printf("✓ Migration status - Table exists: %t, Needs migration: %t\n",
				status.TableExists, status.NeedsMigration)
		}

		// Get table info
		tableInfo, err := migratableUserRepo.GetTableInfo(ctx)
		if err != nil {
			log.Printf("Failed to get table info: %v", err)
		} else {
			fmt.Printf("✓ Table info - Name: %s, Columns: %d\n",
				tableInfo.Name, len(tableInfo.Columns))
		}
	}

	// Migrate other tables
	if migratableProfileRepo, ok := profileRepo.(gpa.MigratableRepository[Profile]); ok {
		migratableProfileRepo.MigrateTable(ctx)
		fmt.Println("✓ Profiles table migrated")
	}

	if migratablePostRepo, ok := postRepo.(gpa.MigratableRepository[Post]); ok {
		migratablePostRepo.MigrateTable(ctx)
		fmt.Println("✓ Posts table migrated")
	}

	// ============================================
	// Index Management
	// ============================================
	fmt.Println("\n=== Index Management ===")

	if sqlUserRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Create a composite index
		err = sqlUserRepo.CreateIndex(ctx, []string{"age", "is_active"}, false)
		if err != nil {
			log.Printf("Failed to create index: %v", err)
		} else {
			fmt.Println("✓ Created composite index on age and is_active")
		}

		// Create a unique index
		err = sqlUserRepo.CreateIndex(ctx, []string{"email"}, true)
		if err != nil {
			log.Printf("Index might already exist: %v", err)
		} else {
			fmt.Println("✓ Created unique index on email")
		}
	}

	// ============================================
	// Basic CRUD Operations
	// ============================================
	fmt.Println("\n=== Basic CRUD Operations ===")

	// Create users
	users := []*User{
		{Name: "John Doe", Email: "john@example.com", Age: 30, IsActive: true},
		{Name: "Jane Smith", Email: "jane@example.com", Age: 25, IsActive: true},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, IsActive: false},
		{Name: "Alice Brown", Email: "alice@example.com", Age: 28, IsActive: true},
		{Name: "Charlie Wilson", Email: "charlie@example.com", Age: 45, IsActive: true},
	}

	err = userRepo.CreateBatch(ctx, users)
	if err != nil {
		log.Printf("Failed to create users: %v", err)
	} else {
		fmt.Printf("✓ Created %d users\n", len(users))
	}

	// Create profiles for users
	profiles := []*Profile{
		{UserID: users[0].ID, Bio: "Software developer", Website: "https://johndoe.dev", Location: "San Francisco"},
		{UserID: users[1].ID, Bio: "Product designer", Website: "https://janesmith.design", Location: "New York"},
		{UserID: users[3].ID, Bio: "Data scientist", Website: "https://alicebrown.ai", Location: "Boston"},
	}

	err = profileRepo.CreateBatch(ctx, profiles)
	if err != nil {
		log.Printf("Failed to create profiles: %v", err)
	} else {
		fmt.Printf("✓ Created %d profiles\n", len(profiles))
	}

	// Create posts
	posts := []*Post{
		{UserID: users[0].ID, Title: "Getting Started with Go", Content: "Go is an amazing language...", Published: true},
		{UserID: users[0].ID, Title: "Advanced Go Patterns", Content: "In this post, we'll explore...", Published: false},
		{UserID: users[1].ID, Title: "Design Principles", Content: "Good design is about...", Published: true},
		{UserID: users[3].ID, Title: "Machine Learning Basics", Content: "Let's dive into ML...", Published: true},
	}

	err = postRepo.CreateBatch(ctx, posts)
	if err != nil {
		log.Printf("Failed to create posts: %v", err)
	} else {
		fmt.Printf("✓ Created %d posts\n", len(posts))
	}

	// ============================================
	// SQL-Specific Queries
	// ============================================
	fmt.Println("\n=== SQL-Specific Queries ===")

	if sqlUserRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Raw SQL query
		activeUsers, err := sqlUserRepo.FindBySQL(ctx,
			"SELECT * FROM users WHERE is_active = ? AND age > ? ORDER BY created_at DESC",
			[]interface{}{true, 25})
		if err != nil {
			log.Printf("Failed to execute raw SQL: %v", err)
		} else {
			fmt.Printf("✓ Raw SQL found %d active users over 25\n", len(activeUsers))
		}

		// Complex join query
		usersWithProfiles, err := sqlUserRepo.FindBySQL(ctx,
			`SELECT u.* FROM users u
			 INNER JOIN profiles p ON u.id = p.user_id
			 WHERE u.is_active = ?
			 ORDER BY u.name`,
			[]interface{}{true})
		if err != nil {
			log.Printf("Failed to execute join query: %v", err)
		} else {
			fmt.Printf("✓ Found %d users with profiles\n", len(usersWithProfiles))
		}

		// Execute raw SQL command
		result, err := sqlUserRepo.ExecSQL(ctx,
			"UPDATE users SET is_active = ? WHERE age > ?",
			false, 40)
		if err != nil {
			log.Printf("Failed to execute raw SQL command: %v", err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			fmt.Printf("✓ Updated %d users (set inactive for age > 40)\n", rowsAffected)
		}
	}

	// ============================================
	// Relationship Queries
	// ============================================
	fmt.Println("\n=== Relationship Queries ===")

	if sqlUserRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Find users with their profiles preloaded
		usersWithRelations, err := sqlUserRepo.FindWithRelations(ctx,
			[]string{"Profile"},
			gpa.Where("is_active", gpa.OpEqual, true))
		if err != nil {
			log.Printf("Failed to find users with relations: %v", err)
		} else {
			fmt.Printf("✓ Found %d users with preloaded profiles\n", len(usersWithRelations))
			for _, user := range usersWithRelations {
				if user.Profile != nil {
					bio := user.Profile.Bio
					if len(bio) > 20 {
						bio = bio[:20] + "..."
					}
					fmt.Printf("  %s has profile: %s\n", user.Name, bio)
				}
			}
		}

		// Find specific user with relations
		userWithProfile, err := sqlUserRepo.FindByIDWithRelations(ctx, users[0].ID, []string{"Profile"})
		if err != nil {
			log.Printf("Failed to find user with profile: %v", err)
		} else {
			fmt.Printf("✓ Found user %s with profile\n", userWithProfile.Name)
			if userWithProfile.Profile != nil {
				fmt.Printf("  Profile: %s\n", userWithProfile.Profile.Bio)
			}
		}
	}

	// ============================================
	// Complex Query Operations
	// ============================================
	fmt.Println("\n=== Complex Query Operations ===")

	// Multi-condition queries
	youngActiveUsers, err := userRepo.Query(ctx,
		gpa.Where("age", gpa.OpLessThan, 30),
		gpa.Where("is_active", gpa.OpEqual, true),
		gpa.OrderBy("age", gpa.OrderDesc),
	)
	if err != nil {
		log.Printf("Failed to query young active users: %v", err)
	} else {
		fmt.Printf("✓ Found %d young active users\n", len(youngActiveUsers))
	}

	// Query with LIKE and IN operators
	specificUsers, err := userRepo.Query(ctx,
		gpa.WhereLike("name", "J%"),
		gpa.WhereIn("age", []interface{}{25, 30, 35}),
	)
	if err != nil {
		log.Printf("Failed to query specific users: %v", err)
	} else {
		fmt.Printf("✓ Found %d users named J* with specific ages\n", len(specificUsers))
	}

	// Pagination example
	page1Users, err := userRepo.Query(ctx,
		gpa.OrderBy("created_at", gpa.OrderDesc),
		gpa.Limit(3),
		gpa.Offset(0),
	)
	if err != nil {
		log.Printf("Failed to get page 1: %v", err)
	} else {
		fmt.Printf("✓ Page 1: %d users\n", len(page1Users))
	}

	// ============================================
	// Aggregation and Statistics
	// ============================================
	fmt.Println("\n=== Aggregation and Statistics ===")

	// Count users by status
	activeCount, _ := userRepo.Count(ctx, gpa.Where("is_active", gpa.OpEqual, true))
	inactiveCount, _ := userRepo.Count(ctx, gpa.Where("is_active", gpa.OpEqual, false))
	totalUsers, _ := userRepo.Count(ctx)

	fmt.Printf("✓ User statistics:\n")
	fmt.Printf("  Total: %d\n", totalUsers)
	fmt.Printf("  Active: %d\n", activeCount)
	fmt.Printf("  Inactive: %d\n", inactiveCount)

	// Count posts by user
	for _, user := range users[:3] { // Check first 3 users
		postCount, err := postRepo.Count(ctx, gpa.Where("user_id", gpa.OpEqual, user.ID))
		if err != nil {
			log.Printf("Failed to count posts for user %s: %v", user.Name, err)
		} else {
			fmt.Printf("  %s has %d posts\n", user.Name, postCount)
		}
	}

	// ============================================
	// Transaction Examples
	// ============================================
	fmt.Println("\n=== Transaction Examples ===")

	// Successful transaction
	err = userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
		// Create a new user
		newUser := &User{
			Name:     "Transaction User",
			Email:    "tx@example.com",
			Age:      32,
			IsActive: true,
		}
		if err := tx.Create(ctx, newUser); err != nil {
			return err
		}

		// Update another user
		if err := tx.UpdatePartial(ctx, users[0].ID, map[string]interface{}{
			"age": 31,
		}); err != nil {
			return err
		}

		fmt.Println("✓ Transaction operations completed")
		return nil
	})
	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("✓ Transaction committed successfully")
	}

	// Transaction with rollback
	err = userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
		// Create a user
		tempUser := &User{
			Name:     "Temp User",
			Email:    "temp@example.com",
			Age:      25,
			IsActive: true,
		}
		if err := tx.Create(ctx, tempUser); err != nil {
			return err
		}

		// Force a rollback
		return gpa.NewError(gpa.ErrorTypeValidation, "intentional rollback")
	})
	if err != nil {
		fmt.Println("✓ Transaction rolled back as expected")
	}

	// ============================================
	// Advanced GORM Features
	// ============================================
	fmt.Println("\n=== Advanced GORM Features ===")

	// Get entity information
	entityInfo, err := userRepo.GetEntityInfo()
	if err != nil {
		log.Printf("Failed to get entity info: %v", err)
	} else {
		fmt.Printf("✓ Entity info - Name: %s, Table: %s, Fields: %d\n",
			entityInfo.Name, entityInfo.TableName, len(entityInfo.Fields))

		// Show primary key fields
		fmt.Printf("  Primary keys: %v\n", entityInfo.PrimaryKey)

		// Show first few fields
		for i, field := range entityInfo.Fields[:3] {
			fmt.Printf("  Field %d: %s (%s) - PK: %t, Nullable: %t\n",
				i+1, field.Name, field.DatabaseType, field.IsPrimaryKey, field.IsNullable)
		}
	}

	// Table operations
	if _, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Note: Creating/dropping tables should be done carefully in production
		fmt.Println("✓ Table operations available (create/drop tables)")
	}

	// ============================================
	// Performance Examples
	// ============================================
	fmt.Println("\n=== Performance Examples ===")

	// Existence check (more efficient than count)
	hasYoungUsers, err := userRepo.Exists(ctx, gpa.Where("age", gpa.OpLessThan, 25))
	if err != nil {
		log.Printf("Failed to check existence: %v", err)
	} else {
		fmt.Printf("✓ Has users under 25: %t\n", hasYoungUsers)
	}

	// Batch operations
	batchUsers := []*User{
		{Name: "Batch User 1", Email: "batch1@example.com", Age: 20, IsActive: true},
		{Name: "Batch User 2", Email: "batch2@example.com", Age: 22, IsActive: true},
	}
	err = userRepo.CreateBatch(ctx, batchUsers)
	if err != nil {
		log.Printf("Failed to create batch: %v", err)
	} else {
		fmt.Printf("✓ Created %d users in batch operation\n", len(batchUsers))
	}

	// ============================================
	// Cleanup and Final Stats
	// ============================================
	fmt.Println("\n=== Final Statistics ===")

	finalUserCount, _ := userRepo.Count(ctx)
	finalProfileCount, _ := profileRepo.Count(ctx)
	finalPostCount, _ := postRepo.Count(ctx)

	fmt.Printf("✓ Final counts:\n")
	fmt.Printf("  Users: %d\n", finalUserCount)
	fmt.Printf("  Profiles: %d\n", finalProfileCount)
	fmt.Printf("  Posts: %d\n", finalPostCount)

	fmt.Println("\n🎉 GORM provider example completed!")
}
