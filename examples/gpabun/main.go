package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpabun"
)

// User represents a user entity with Bun-specific tags
type User struct {
	ID        int64     `bun:"id,pk,autoincrement"`
	Name      string    `bun:"name,notnull"`
	Email     string    `bun:"email,unique,notnull"`
	Age       int       `bun:"age,notnull"`
	IsActive  bool      `bun:"is_active,default:true"`
	Salary    *float64  `bun:"salary,nullzero"` // Nullable field
	CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`
	Profile   *Profile  `bun:"rel:has-one,join:id=user_id"`
	Posts     []*Post   `bun:"rel:has-many,join:id=user_id"`
}

// Profile represents a user profile with a foreign key relationship
type Profile struct {
	ID       int64  `bun:"id,pk,autoincrement"`
	UserID   int64  `bun:"user_id,notnull"`
	Bio      string `bun:"bio,type:text"`
	Website  string `bun:"website"`
	Location string `bun:"location"`
	Skills   string `bun:"skills,type:text"` // JSON-like storage
	User     *User  `bun:"rel:belongs-to,join:user_id=id"`
}

// Post represents a blog post
type Post struct {
	ID        int64     `bun:"id,pk,autoincrement"`
	UserID    int64     `bun:"user_id,notnull"`
	Title     string    `bun:"title,notnull"`
	Content   string    `bun:"content,type:text"`
	Published bool      `bun:"published,default:false"`
	Views     int       `bun:"views,default:0"`
	Tags      string    `bun:"tags"` // Comma-separated tags
	CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`
	User      *User     `bun:"rel:belongs-to,join:user_id=id"`
}

// Comment represents a comment on a post
type Comment struct {
	ID        int64     `bun:"id,pk,autoincrement"`
	PostID    int64     `bun:"post_id,notnull"`
	UserID    int64     `bun:"user_id,notnull"`
	Content   string    `bun:"content,type:text,notnull"`
	CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
	Post      *Post     `bun:"rel:belongs-to,join:post_id=id"`
	User      *User     `bun:"rel:belongs-to,join:user_id=id"`
}

func main() {
	fmt.Println("🏗️  Bun Provider Example")
	fmt.Println("Demonstrating Bun SQL toolkit features and advanced query patterns")

	// Configure Bun with SQLite database file (shared between all repositories)
	// Note: Using file-based DB instead of :memory: to share between multiple providers
	config := gpa.Config{
		Driver:   "sqlite",
		Database: "/tmp/bun_example_shared.db",
		Options: map[string]interface{}{
			"bun": map[string]interface{}{
				"debug":           true,
				"log_slow_query":  true,
				"slow_query_time": "100ms",
			},
		},
	}

	// Clean up any existing database file
	os.Remove("/tmp/bun_example_shared.db")
	defer os.Remove("/tmp/bun_example_shared.db") // Clean up after example

	// Create type-safe providers (they'll share the same database file)
	userProvider, err := gpabun.NewTypeSafeProvider[User](config)
	if err != nil {
		log.Fatalf("Failed to create user provider: %v", err)
	}
	defer userProvider.Close()

	profileProvider, err := gpabun.NewTypeSafeProvider[Profile](config)
	if err != nil {
		log.Fatalf("Failed to create profile provider: %v", err)
	}
	defer profileProvider.Close()

	postProvider, err := gpabun.NewTypeSafeProvider[Post](config)
	if err != nil {
		log.Fatalf("Failed to create post provider: %v", err)
	}
	defer postProvider.Close()

	commentProvider, err := gpabun.NewTypeSafeProvider[Comment](config)
	if err != nil {
		log.Fatalf("Failed to create comment provider: %v", err)
	}
	defer commentProvider.Close()

	// Get repositories
	userRepo := userProvider.Repository()
	profileRepo := profileProvider.Repository()
	postRepo := postProvider.Repository()
	commentRepo := commentProvider.Repository()

	ctx := context.Background()

	// ============================================
	// Schema Migration and Setup
	// ============================================
	fmt.Println("\n=== Schema Migration and Setup ===")

	// Create tables manually since Bun provider doesn't implement MigratableRepository yet
	// In a production environment, you would use Bun's migration features
	// Note: All providers share the same database connection in this example
	bunDB := userProvider.(*gpabun.TypeSafeProvider[User]).Repository().(*gpabun.Repository[User])
	
	// Create users table
	_, err = bunDB.RawExec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			age INTEGER NOT NULL,
			is_active BOOLEAN DEFAULT true,
			salary REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}
	fmt.Println("✓ Users table created")

	// Create profiles table
	_, err = bunDB.RawExec(ctx, `
		CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			bio TEXT,
			website TEXT,
			location TEXT,
			skills TEXT,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`, nil)
	if err != nil {
		log.Fatalf("Failed to create profiles table: %v", err)
	}
	fmt.Println("✓ Profiles table created")

	// Create posts table
	_, err = bunDB.RawExec(ctx, `
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT false,
			views INTEGER DEFAULT 0,
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`, nil)
	if err != nil {
		log.Fatalf("Failed to create posts table: %v", err)
	}
	fmt.Println("✓ Posts table created")

	// Create comments table
	_, err = bunDB.RawExec(ctx, `
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`, nil)
	if err != nil {
		log.Fatalf("Failed to create comments table: %v", err)
	}
	fmt.Println("✓ Comments table created")

	// ============================================
	// Index Creation for Performance
	// ============================================
	fmt.Println("\n=== Index Creation ===")

	if sqlUserRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Create indexes for better query performance
		indexes := []struct {
			name    string
			columns []string
			unique  bool
		}{
			{"idx_users_email", []string{"email"}, true},
			{"idx_users_active_age", []string{"is_active", "age"}, false},
			{"idx_users_created_at", []string{"created_at"}, false},
		}

		for _, idx := range indexes {
			err = sqlUserRepo.CreateIndex(ctx, idx.columns, idx.unique)
			if err != nil {
				log.Printf("Index %s might already exist: %v", idx.name, err)
			} else {
				fmt.Printf("✓ Created index: %s\n", idx.name)
			}
		}
	}

	if sqlPostRepo, ok := postRepo.(gpa.SQLRepository[Post]); ok {
		err = sqlPostRepo.CreateIndex(ctx, []string{"user_id", "published"}, false)
		if err != nil {
			log.Printf("Post index might already exist: %v", err)
		} else {
			fmt.Println("✓ Created post index on user_id and published")
		}
	}

	// ============================================
	// Data Creation with Relationships
	// ============================================
	fmt.Println("\n=== Data Creation ===")

	// Create users with diverse data
	salary1 := 75000.0
	salary2 := 95000.0
	users := []*User{
		{
			Name:     "Alice Johnson",
			Email:    "alice@example.com",
			Age:      28,
			IsActive: true,
			Salary:   &salary1,
		},
		{
			Name:     "Bob Smith",
			Email:    "bob@example.com", 
			Age:      32,
			IsActive: true,
			Salary:   &salary2,
		},
		{
			Name:     "Charlie Brown",
			Email:    "charlie@example.com",
			Age:      45,
			IsActive: false,
			Salary:   nil, // NULL salary
		},
		{
			Name:     "Diana Prince",
			Email:    "diana@example.com",
			Age:      30,
			IsActive: true,
			Salary:   &salary2,
		},
		{
			Name:     "Eve Adams",
			Email:    "eve@example.com",
			Age:      26,
			IsActive: true,
			Salary:   &salary1,
		},
	}

	err = userRepo.CreateBatch(ctx, users)
	if err != nil {
		log.Printf("Failed to create users: %v", err)
	} else {
		fmt.Printf("✓ Created %d users\n", len(users))
	}

	// Create profiles for users
	profiles := []*Profile{
		{
			UserID:   users[0].ID,
			Bio:      "Full-stack developer passionate about clean code and user experience",
			Website:  "https://alice-dev.com",
			Location: "San Francisco, CA",
			Skills:   `["Go", "JavaScript", "React", "PostgreSQL", "Docker"]`,
		},
		{
			UserID:   users[1].ID,
			Bio:      "Backend engineer specializing in distributed systems and microservices",
			Website:  "https://bobsmith.tech",
			Location: "Seattle, WA",
			Skills:   `["Go", "Kubernetes", "gRPC", "MongoDB", "Redis"]`,
		},
		{
			UserID:   users[2].ID,
			Bio:      "Senior engineering manager with 15+ years of experience",
			Website:  "",
			Location: "Austin, TX",
			Skills:   `["Leadership", "Go", "Python", "Architecture", "Team Management"]`,
		},
		{
			UserID:   users[3].ID,
			Bio:      "DevOps engineer focused on infrastructure automation and CI/CD",
			Website:  "https://diana-ops.com",
			Location: "New York, NY",
			Skills:   `["Terraform", "AWS", "Docker", "Jenkins", "Go"]`,
		},
	}

	err = profileRepo.CreateBatch(ctx, profiles)
	if err != nil {
		log.Printf("Failed to create profiles: %v", err)
	} else {
		fmt.Printf("✓ Created %d profiles\n", len(profiles))
	}

	// Create blog posts
	posts := []*Post{
		{
			UserID:    users[0].ID,
			Title:     "Building Scalable APIs with Go and Bun",
			Content:   "In this comprehensive guide, we'll explore how to build high-performance APIs using Go and the Bun SQL toolkit...",
			Published: true,
			Views:     245,
			Tags:      "go,api,bun,sql,performance",
		},
		{
			UserID:    users[0].ID,
			Title:     "Advanced Query Patterns with Bun",
			Content:   "Bun provides powerful query building capabilities. Let's dive into advanced patterns including joins, subqueries...",
			Published: false,
			Views:     0,
			Tags:      "go,bun,database,queries,advanced",
		},
		{
			UserID:    users[1].ID,
			Title:     "Microservices Communication Patterns",
			Content:   "When building distributed systems, choosing the right communication pattern is crucial...",
			Published: true,
			Views:     189,
			Tags:      "microservices,grpc,distributed,architecture",
		},
		{
			UserID:    users[1].ID,
			Title:     "Database Sharding Strategies",
			Content:   "As your application grows, you may need to consider database sharding. Here's how to approach it...",
			Published: true,
			Views:     312,
			Tags:      "database,sharding,scaling,performance",
		},
		{
			UserID:    users[3].ID,
			Title:     "Infrastructure as Code Best Practices",
			Content:   "Terraform has revolutionized infrastructure management. Here are the best practices I've learned...",
			Published: true,
			Views:     156,
			Tags:      "terraform,iac,devops,automation",
		},
	}

	err = postRepo.CreateBatch(ctx, posts)
	if err != nil {
		log.Printf("Failed to create posts: %v", err)
	} else {
		fmt.Printf("✓ Created %d posts\n", len(posts))
	}

	// Create comments
	comments := []*Comment{
		{PostID: posts[0].ID, UserID: users[1].ID, Content: "Great article! I've been using Bun for a few months now and love its simplicity."},
		{PostID: posts[0].ID, UserID: users[3].ID, Content: "Thanks for the detailed examples. The performance comparison section was particularly helpful."},
		{PostID: posts[2].ID, UserID: users[0].ID, Content: "Excellent overview of communication patterns. Have you considered event sourcing?"},
		{PostID: posts[3].ID, UserID: users[0].ID, Content: "This is exactly what I needed for my current project. The sharding strategies are well explained."},
		{PostID: posts[4].ID, UserID: users[1].ID, Content: "Terraform modules are indeed game-changers. Do you have any recommendations for state management?"},
	}

	err = commentRepo.CreateBatch(ctx, comments)
	if err != nil {
		log.Printf("Failed to create comments: %v", err)
	} else {
		fmt.Printf("✓ Created %d comments\n", len(comments))
	}

	// ============================================
	// Advanced SQL Queries with Bun
	// ============================================
	fmt.Println("\n=== Advanced SQL Queries ===")

	if sqlUserRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Complex raw SQL with joins
		userStats, err := sqlUserRepo.FindBySQL(ctx, `
			SELECT 
				u.id,
				u.name,
				u.email,
				u.age,
				u.salary,
				COUNT(p.id) as post_count,
				COALESCE(AVG(p.views), 0) as avg_views,
				MAX(p.created_at) as last_post_date
			FROM users u
			LEFT JOIN posts p ON u.id = p.user_id AND p.published = ?
			WHERE u.is_active = ?
			GROUP BY u.id, u.name, u.email, u.age, u.salary
			HAVING COUNT(p.id) > 0
			ORDER BY avg_views DESC
		`, []interface{}{true, true})
		
		if err != nil {
			log.Printf("Failed to execute user stats query: %v", err)
		} else {
			fmt.Printf("✓ User statistics query returned %d active users with posts\n", len(userStats))
		}

		// Subquery example
		topPosters, err := sqlUserRepo.FindBySQL(ctx, `
			SELECT u.* FROM users u
			WHERE u.id IN (
				SELECT p.user_id 
				FROM posts p 
				WHERE p.published = ? 
				GROUP BY p.user_id 
				HAVING COUNT(*) >= ?
			)
			ORDER BY u.name
		`, []interface{}{true, 2})
		
		if err != nil {
			log.Printf("Failed to execute subquery: %v", err)
		} else {
			fmt.Printf("✓ Found %d users with 2+ published posts\n", len(topPosters))
		}

		// Window function example (SQLite 3.25+)
		rankedUsers, err := sqlUserRepo.FindBySQL(ctx, `
			SELECT 
				name,
				age,
				salary,
				ROW_NUMBER() OVER (ORDER BY age DESC) as age_rank,
				RANK() OVER (ORDER BY COALESCE(salary, 0) DESC) as salary_rank
			FROM users 
			WHERE is_active = ?
			ORDER BY age_rank
		`, []interface{}{true})
		
		if err != nil {
			log.Printf("Window functions might not be supported: %v", err)
		} else {
			fmt.Printf("✓ Ranked %d active users by age and salary\n", len(rankedUsers))
		}
	}

	// ============================================
	// Relationship Queries
	// ============================================
	fmt.Println("\n=== Relationship Queries ===")

	if sqlUserRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
		// Find users with their profiles
		usersWithProfiles, err := sqlUserRepo.FindWithRelations(ctx, 
			[]string{"Profile"}, 
			gpa.Where("is_active", gpa.OpEqual, true))
		
		if err != nil {
			log.Printf("Failed to find users with profiles: %v", err)
		} else {
			fmt.Printf("✓ Found %d active users with profiles\n", len(usersWithProfiles))
			for _, user := range usersWithProfiles {
				if user.Profile != nil {
					fmt.Printf("  %s: %s\n", user.Name, user.Profile.Bio[:50]+"...")
				}
			}
		}

		// Find users with multiple relations
		usersWithAll, err := sqlUserRepo.FindWithRelations(ctx, 
			[]string{"Profile", "Posts"}, 
			gpa.Where("is_active", gpa.OpEqual, true))
		
		if err != nil {
			log.Printf("Failed to find users with all relations: %v", err)
		} else {
			fmt.Printf("✓ Found %d users with profiles and posts\n", len(usersWithAll))
			for _, user := range usersWithAll {
				fmt.Printf("  %s: %d posts\n", user.Name, len(user.Posts))
			}
		}
	}

	// ============================================
	// Aggregation and Analytics
	// ============================================
	fmt.Println("\n=== Aggregation and Analytics ===")

	if sqlPostRepo, ok := postRepo.(gpa.SQLRepository[Post]); ok {
		// Post analytics with aggregation
		postAnalytics, err := sqlPostRepo.FindBySQL(ctx, `
			SELECT 
				'total' as metric,
				COUNT(*) as count,
				0 as avg_value
			FROM posts
			UNION ALL
			SELECT 
				'published' as metric,
				COUNT(*) as count,
				0 as avg_value
			FROM posts WHERE published = ?
			UNION ALL
			SELECT 
				'avg_views' as metric,
				0 as count,
				AVG(views) as avg_value
			FROM posts WHERE published = ?
		`, []interface{}{true, true})
		
		if err != nil {
			log.Printf("Failed to get post analytics: %v", err)
		} else {
			fmt.Printf("✓ Post analytics: %d result(s)\n", len(postAnalytics))
			// Note: In a real implementation, you'd properly parse these results
			fmt.Printf("  Analytics data retrieved\n")
		}

		// Tag analysis
		tagStats, err := sqlPostRepo.FindBySQL(ctx, `
			SELECT 
				TRIM(tag_part) as tag,
				COUNT(*) as usage_count
			FROM (
				SELECT 
					CASE 
						WHEN INSTR(tags, ',') > 0 THEN
							SUBSTR(tags, 1, INSTR(tags, ',') - 1)
						ELSE tags
					END as tag_part
				FROM posts 
				WHERE tags IS NOT NULL AND tags != ''
				UNION ALL
				SELECT 
					CASE 
						WHEN INSTR(SUBSTR(tags, INSTR(tags, ',') + 1), ',') > 0 THEN
							SUBSTR(SUBSTR(tags, INSTR(tags, ',') + 1), 1, 
								INSTR(SUBSTR(tags, INSTR(tags, ',') + 1), ',') - 1)
						ELSE SUBSTR(tags, INSTR(tags, ',') + 1)
					END as tag_part
				FROM posts 
				WHERE tags IS NOT NULL AND INSTR(tags, ',') > 0
			) tag_split
			WHERE tag_part IS NOT NULL AND tag_part != ''
			GROUP BY TRIM(tag_part)
			ORDER BY usage_count DESC
			LIMIT 5
		`, []interface{}{})
		
		if err != nil {
			log.Printf("Failed to analyze tags: %v", err)
		} else {
			fmt.Printf("✓ Found top tags across %d posts\n", len(tagStats))
		}
	}

	// ============================================
	// Complex Query Patterns
	// ============================================
	fmt.Println("\n=== Complex Query Patterns ===")

	// Multi-condition queries with GPA syntax
	complexUsers, err := userRepo.Query(ctx,
		gpa.Where("age", gpa.OpGreaterThanOrEqual, 28),
		gpa.Where("is_active", gpa.OpEqual, true),
		gpa.WhereNotNull("salary"),
		gpa.OrderBy("salary", gpa.OrderDesc),
		gpa.Limit(3),
	)
	
	if err != nil {
		log.Printf("Failed to execute complex query: %v", err)
	} else {
		fmt.Printf("✓ Complex query found %d users (age >= 28, active, with salary)\n", len(complexUsers))
		for _, user := range complexUsers {
			salaryStr := "N/A"
			if user.Salary != nil {
				salaryStr = fmt.Sprintf("$%.0f", *user.Salary)
			}
			fmt.Printf("  %s (age %d): %s\n", user.Name, user.Age, salaryStr)
		}
	}

	// Pattern matching queries
	devUsers, err := userRepo.Query(ctx,
		gpa.WhereLike("name", "A%"), // Names starting with 'A'
		gpa.WhereIn("age", []interface{}{28, 30, 32}),
		gpa.OrderBy("name", gpa.OrderAsc),
	)
	
	if err != nil {
		log.Printf("Failed to execute pattern query: %v", err)
	} else {
		fmt.Printf("✓ Found %d users with names starting with 'A' and specific ages\n", len(devUsers))
	}

	// ============================================
	// Transaction Examples
	// ============================================
	fmt.Println("\n=== Transaction Examples ===")

	// Complex transaction with multiple operations
	err = userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
		// Create a new user
		newUser := &User{
			Name:     "Transaction User",
			Email:    "transaction@example.com",
			Age:      29,
			IsActive: true,
		}
		
		if err := tx.Create(ctx, newUser); err != nil {
			return err
		}

		// Update another user's age
		if err := tx.UpdatePartial(ctx, users[0].ID, map[string]interface{}{
			"age": users[0].Age + 1,
			"updated_at": time.Now(),
		}); err != nil {
			return err
		}

		// Conditional logic
		if newUser.Age > 25 {
			salary := 70000.0
			if err := tx.UpdatePartial(ctx, newUser.ID, map[string]interface{}{
				"salary": &salary,
			}); err != nil {
				return err
			}
		}

		fmt.Println("✓ Transaction operations completed")
		return nil
	})
	
	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("✓ Complex transaction committed successfully")
	}

	// Transaction with rollback demonstration
	err = userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
		tempUser := &User{
			Name:     "Temp User",
			Email:    "temp@example.com",
			Age:      25,
			IsActive: true,
		}
		
		if err := tx.Create(ctx, tempUser); err != nil {
			return err
		}

		// Simulate an error condition
		return fmt.Errorf("simulated error for rollback demonstration")
	})
	
	if err != nil {
		fmt.Println("✓ Transaction rolled back as expected")
	}

	// ============================================
	// Performance Examples
	// ============================================
	fmt.Println("\n=== Performance Examples ===")

	// Batch operations
	newUsers := []*User{
		{Name: "Batch User 1", Email: "batch1@example.com", Age: 24, IsActive: true},
		{Name: "Batch User 2", Email: "batch2@example.com", Age: 26, IsActive: true},
		{Name: "Batch User 3", Email: "batch3@example.com", Age: 28, IsActive: true},
	}
	
	start := time.Now()
	err = userRepo.CreateBatch(ctx, newUsers)
	batchDuration := time.Since(start)
	
	if err != nil {
		log.Printf("Failed to create batch: %v", err)
	} else {
		fmt.Printf("✓ Created %d users in batch (%v)\n", len(newUsers), batchDuration)
	}

	// Existence checks (more efficient than counting)
	hasYoungUsers, err := userRepo.Exists(ctx, 
		gpa.Where("age", gpa.OpLessThan, 30),
		gpa.Where("is_active", gpa.OpEqual, true))
	
	if err != nil {
		log.Printf("Failed to check existence: %v", err)
	} else {
		fmt.Printf("✓ Has active users under 30: %t\n", hasYoungUsers)
	}

	// Count with conditions
	activeUserCount, err := userRepo.Count(ctx, gpa.Where("is_active", gpa.OpEqual, true))
	highEarnerCount, err2 := userRepo.Count(ctx, 
		gpa.Where("salary", gpa.OpGreaterThan, 80000),
		gpa.WhereNotNull("salary"))
	
	if err != nil || err2 != nil {
		log.Printf("Failed to count users: %v, %v", err, err2)
	} else {
		fmt.Printf("✓ Active users: %d, High earners (>$80k): %d\n", activeUserCount, highEarnerCount)
	}

	// ============================================
	// Schema Information and Metadata
	// ============================================
	fmt.Println("\n=== Schema Information ===")

	// Get entity information
	userEntityInfo, err := userRepo.GetEntityInfo()
	if err != nil {
		log.Printf("Failed to get user entity info: %v", err)
	} else {
		fmt.Printf("✓ User entity info:\n")
		fmt.Printf("  Name: %s\n", userEntityInfo.Name)
		fmt.Printf("  Table: %s\n", userEntityInfo.TableName)
		fmt.Printf("  Fields: %d\n", len(userEntityInfo.Fields))
		fmt.Printf("  Primary Keys: %v\n", userEntityInfo.PrimaryKey)
		
		// Show a few field details (if any fields are available)
		if len(userEntityInfo.Fields) > 0 {
			limit := len(userEntityInfo.Fields)
			if limit > 3 {
				limit = 3
			}
			for i := 0; i < limit; i++ {
				field := userEntityInfo.Fields[i]
				fmt.Printf("  Field %d: %s (%s) - PK: %t, Nullable: %t\n",
					i+1, field.Name, field.DatabaseType, field.IsPrimaryKey, field.IsNullable)
			}
		} else {
			fmt.Printf("  (Note: Field details not available in Bun provider)\n")
		}
	}

	// Table information
	fmt.Printf("✓ Table information not available (Bun provider doesn't implement MigratableRepository yet)\n")

	// ============================================
	// Final Statistics
	// ============================================
	fmt.Println("\n=== Final Statistics ===")

	finalUserCount, _ := userRepo.Count(ctx)
	finalProfileCount, _ := profileRepo.Count(ctx)
	finalPostCount, _ := postRepo.Count(ctx)
	finalCommentCount, _ := commentRepo.Count(ctx)

	fmt.Printf("✓ Final database counts:\n")
	fmt.Printf("  Users: %d\n", finalUserCount)
	fmt.Printf("  Profiles: %d\n", finalProfileCount)
	fmt.Printf("  Posts: %d\n", finalPostCount)
	fmt.Printf("  Comments: %d\n", finalCommentCount)

	// Provider information
	fmt.Printf("\n✓ Provider information:\n")
	providerInfo := userProvider.ProviderInfo()
	fmt.Printf("  Name: %s\n", providerInfo.Name)
	fmt.Printf("  Version: %s\n", providerInfo.Version)
	fmt.Printf("  Database Type: %s\n", providerInfo.DatabaseType)
	fmt.Printf("  Features: %v\n", providerInfo.Features)

	fmt.Println("\n🎉 Bun provider example completed!")
}