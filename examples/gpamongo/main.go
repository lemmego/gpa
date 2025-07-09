package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpamongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user document in MongoDB
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Age       int                `bson:"age" json:"age"`
	IsActive  bool               `bson:"is_active" json:"is_active"`
	Tags      []string           `bson:"tags,omitempty" json:"tags,omitempty"`
	Profile   *UserProfile       `bson:"profile,omitempty" json:"profile,omitempty"`
	Location  *GeoLocation       `bson:"location,omitempty" json:"location,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserProfile represents embedded profile data
type UserProfile struct {
	Bio      string   `bson:"bio" json:"bio"`
	Website  string   `bson:"website" json:"website"`
	Skills   []string `bson:"skills" json:"skills"`
	Social   Social   `bson:"social" json:"social"`
}

// Social represents social media links
type Social struct {
	Twitter   string `bson:"twitter,omitempty" json:"twitter,omitempty"`
	LinkedIn  string `bson:"linkedin,omitempty" json:"linkedin,omitempty"`
	GitHub    string `bson:"github,omitempty" json:"github,omitempty"`
}

// GeoLocation represents geographic coordinates
type GeoLocation struct {
	Type        string    `bson:"type" json:"type"`
	Coordinates []float64 `bson:"coordinates" json:"coordinates"` // [longitude, latitude]
	City        string    `bson:"city" json:"city"`
	Country     string    `bson:"country" json:"country"`
}

// BlogPost represents a blog post document
type BlogPost struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Title     string             `bson:"title" json:"title"`
	Content   string             `bson:"content" json:"content"`
	Summary   string             `bson:"summary" json:"summary"`
	Tags      []string           `bson:"tags" json:"tags"`
	Category  string             `bson:"category" json:"category"`
	Published bool               `bson:"published" json:"published"`
	Views     int                `bson:"views" json:"views"`
	Likes     int                `bson:"likes" json:"likes"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

func main() {
	fmt.Println("🍃 MongoDB Provider Example")
	fmt.Println("Demonstrating MongoDB-specific features and document operations")

	// Check if MongoDB is available
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	// Configure MongoDB connection
	config := gpa.Config{
		Driver:        "mongodb",
		ConnectionURL: mongoURL,
		Database:      "gpa_mongo_example",
		Options: map[string]interface{}{
			"mongo": map[string]interface{}{
				"max_pool_size": uint64(50),
				"min_pool_size": uint64(5),
			},
		},
	}

	// Create type-safe providers
	userProvider, err := gpamongo.NewTypeSafeProvider[User](config)
	if err != nil {
		log.Fatalf("Failed to create user provider (ensure MongoDB is running): %v", err)
	}
	defer userProvider.Close()

	postProvider, err := gpamongo.NewTypeSafeProvider[BlogPost](config)
	if err != nil {
		log.Fatalf("Failed to create post provider: %v", err)
	}
	defer postProvider.Close()

	// Get repositories
	userRepo := userProvider.Repository()
	postRepo := postProvider.Repository()

	ctx := context.Background()

	// Check provider health
	err = userProvider.Health()
	if err != nil {
		log.Fatalf("MongoDB health check failed: %v", err)
	}
	fmt.Println("✓ Connected to MongoDB successfully")

	// ============================================
	// Cleanup any existing data from previous runs
	// ============================================
	fmt.Println("\n=== Cleanup Previous Data ===")
	
	// Clean up existing users and posts to avoid duplicate key errors
	existingUsers, _ := userRepo.FindAll(ctx)
	for _, user := range existingUsers {
		userRepo.Delete(ctx, user.ID)
	}
	
	existingPosts, _ := postRepo.FindAll(ctx)
	for _, post := range existingPosts {
		postRepo.Delete(ctx, post.ID)
	}
	
	fmt.Printf("✓ Cleaned up %d existing users and %d existing posts\n", len(existingUsers), len(existingPosts))

	// ============================================
	// Document Creation and Basic Operations
	// ============================================
	fmt.Println("\n=== Document Creation ===")

	// Create users with complex nested structures
	users := []*User{
		{
			Name:     "John Doe",
			Email:    "john@example.com",
			Age:      30,
			IsActive: true,
			Tags:     []string{"developer", "golang", "mongodb"},
			Profile: &UserProfile{
				Bio:    "Full-stack developer passionate about Go and MongoDB",
				Website: "https://johndoe.dev",
				Skills: []string{"Go", "MongoDB", "JavaScript", "Docker"},
				Social: Social{
					Twitter:  "@johndoe",
					LinkedIn: "linkedin.com/in/johndoe",
					GitHub:   "github.com/johndoe",
				},
			},
			Location: &GeoLocation{
				Type:        "Point",
				Coordinates: []float64{-122.4194, 37.7749}, // San Francisco
				City:        "San Francisco",
				Country:     "USA",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:     "Jane Smith",
			Email:    "jane@example.com",
			Age:      28,
			IsActive: true,
			Tags:     []string{"designer", "ui", "ux"},
			Profile: &UserProfile{
				Bio:    "UX/UI Designer creating beautiful and functional interfaces",
				Website: "https://janesmith.design",
				Skills: []string{"Figma", "Adobe Creative Suite", "HTML", "CSS"},
				Social: Social{
					Twitter:  "@janesmith",
					LinkedIn: "linkedin.com/in/janesmith",
				},
			},
			Location: &GeoLocation{
				Type:        "Point",
				Coordinates: []float64{-74.0060, 40.7128}, // New York
				City:        "New York",
				Country:     "USA",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:     "Bob Johnson",
			Email:    "bob@example.com",
			Age:      35,
			IsActive: false,
			Tags:     []string{"manager", "agile", "scrum"},
			Profile: &UserProfile{
				Bio:    "Engineering manager focused on team productivity",
				Skills: []string{"Leadership", "Agile", "Project Management"},
				Social: Social{
					LinkedIn: "linkedin.com/in/bobjohnson",
				},
			},
			Location: &GeoLocation{
				Type:        "Point",
				Coordinates: []float64{-87.6298, 41.8781}, // Chicago
				City:        "Chicago",
				Country:     "USA",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err = userRepo.CreateBatch(ctx, users)
	if err != nil {
		log.Printf("Failed to create users: %v", err)
	} else {
		fmt.Printf("✓ Created %d users with complex nested documents\n", len(users))
		for _, user := range users {
			fmt.Printf("  %s (ID: %s)\n", user.Name, user.ID.Hex())
		}
	}

	// Create blog posts
	posts := []*BlogPost{
		{
			UserID:    users[0].ID,
			Title:     "Getting Started with MongoDB and Go",
			Content:   "MongoDB is a powerful NoSQL database that works excellently with Go...",
			Summary:   "Learn how to use MongoDB with Go for modern applications",
			Tags:      []string{"mongodb", "golang", "nosql", "tutorial"},
			Category:  "Programming",
			Published: true,
			Views:     150,
			Likes:     23,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			UserID:    users[0].ID,
			Title:     "Advanced MongoDB Aggregation Pipelines",
			Content:   "Aggregation pipelines in MongoDB allow for complex data processing...",
			Summary:   "Master MongoDB aggregation for powerful data analysis",
			Tags:      []string{"mongodb", "aggregation", "data", "advanced"},
			Category:  "Programming",
			Published: false,
			Views:     45,
			Likes:     7,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			UserID:    users[1].ID,
			Title:     "Modern UI Design Principles",
			Content:   "Creating user interfaces that are both beautiful and functional...",
			Summary:   "Essential principles for modern UI design",
			Tags:      []string{"design", "ui", "ux", "principles"},
			Category:  "Design",
			Published: true,
			Views:     89,
			Likes:     15,
			CreatedAt: time.Now().Add(-3 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
	}

	err = postRepo.CreateBatch(ctx, posts)
	if err != nil {
		log.Printf("Failed to create posts: %v", err)
	} else {
		fmt.Printf("✓ Created %d blog posts\n", len(posts))
	}

	// ============================================
	// Document-Specific Queries
	// ============================================
	fmt.Println("\n=== Document-Specific Queries ===")

	if docRepo, ok := userRepo.(gpa.DocumentRepository[User]); ok {
		// Query using MongoDB query syntax
		activeDevs, err := docRepo.FindByDocument(ctx, map[string]interface{}{
			"is_active": true,
			"tags":      map[string]interface{}{"$in": []string{"developer", "golang"}},
			"age":       map[string]interface{}{"$gte": 25},
		})
		if err != nil {
			log.Printf("Failed to find active developers: %v", err)
		} else {
			fmt.Printf("✓ Found %d active developers aged 25+\n", len(activeDevs))
		}

		// Text search (requires text index)
		// First create a text index
		textIndexKeys := map[string]interface{}{
			"name":          "text",
			"profile.bio":   "text",
			"profile.skills": "text",
		}
		err = docRepo.CreateIndex(ctx, textIndexKeys, false)
		if err != nil {
			log.Printf("Text index might already exist: %v", err)
		} else {
			fmt.Println("✓ Created text search index")
		}

		// Perform text search
		searchResults, err := docRepo.TextSearch(ctx, "developer golang", gpa.Limit(5))
		if err != nil {
			log.Printf("Failed to perform text search: %v", err)
		} else {
			fmt.Printf("✓ Text search found %d users matching 'developer golang'\n", len(searchResults))
		}

		// Geospatial queries
		// Create geospatial index
		geoIndexKeys := map[string]interface{}{
			"location": "2dsphere",
		}
		err = docRepo.CreateIndex(ctx, geoIndexKeys, false)
		if err != nil {
			log.Printf("Geo index might already exist: %v", err)
		} else {
			fmt.Println("✓ Created geospatial index")
		}

		// Find users near San Francisco (within 100km)
		nearSF, err := docRepo.FindNear(ctx, "location", 
			[]float64{-122.4194, 37.7749}, 100000) // 100km in meters
		if err != nil {
			log.Printf("Failed to find users near SF: %v", err)
		} else {
			fmt.Printf("✓ Found %d users near San Francisco\n", len(nearSF))
		}
	}

	// ============================================
	// Complex MongoDB Aggregations
	// ============================================
	fmt.Println("\n=== MongoDB Aggregation Pipelines ===")

	if docUserRepo, ok := userRepo.(gpa.DocumentRepository[User]); ok {
		// User statistics by location
		userStatsPipeline := []map[string]interface{}{
			{
				"$match": map[string]interface{}{
					"is_active": true,
				},
			},
			{
				"$group": map[string]interface{}{
					"_id": "$location.country",
					"count": map[string]interface{}{"$sum": 1},
					"avgAge": map[string]interface{}{"$avg": "$age"},
					"cities": map[string]interface{}{"$addToSet": "$location.city"},
				},
			},
			{
				"$sort": map[string]interface{}{
					"count": -1,
				},
			},
		}

		userStats, err := docUserRepo.Aggregate(ctx, userStatsPipeline)
		if err != nil {
			log.Printf("Failed to aggregate user stats: %v", err)
		} else {
			fmt.Printf("✓ User statistics by country:\n")
			for _, stat := range userStats {
				country := stat["_id"]
				count := stat["count"]
				avgAge := stat["avgAge"]
				fmt.Printf("  %s: %v users, avg age %.1f\n", country, count, avgAge)
			}
		}

		// Skills analysis
		skillsPipeline := []map[string]interface{}{
			{
				"$match": map[string]interface{}{
					"profile.skills": map[string]interface{}{"$exists": true},
				},
			},
			{
				"$unwind": "$profile.skills",
			},
			{
				"$group": map[string]interface{}{
					"_id":   "$profile.skills",
					"count": map[string]interface{}{"$sum": 1},
					"users": map[string]interface{}{"$addToSet": "$name"},
				},
			},
			{
				"$sort": map[string]interface{}{
					"count": -1,
				},
			},
			{
				"$limit": 5,
			},
		}

		skillsStats, err := docUserRepo.Aggregate(ctx, skillsPipeline)
		if err != nil {
			log.Printf("Failed to aggregate skills: %v", err)
		} else {
			fmt.Printf("✓ Top 5 skills:\n")
			for i, skill := range skillsStats {
				skillName := skill["_id"]
				count := skill["count"]
				fmt.Printf("  %d. %s (%v users)\n", i+1, skillName, count)
			}
		}
	}

	if docPostRepo, ok := postRepo.(gpa.DocumentRepository[BlogPost]); ok {
		// Blog post analytics
		postAnalyticsPipeline := []map[string]interface{}{
			{
				"$group": map[string]interface{}{
					"_id": "$category",
					"totalPosts": map[string]interface{}{"$sum": 1},
					"publishedPosts": map[string]interface{}{
						"$sum": map[string]interface{}{
							"$cond": []interface{}{
								"$published", 1, 0,
							},
						},
					},
					"totalViews": map[string]interface{}{"$sum": "$views"},
					"totalLikes": map[string]interface{}{"$sum": "$likes"},
					"avgViews": map[string]interface{}{"$avg": "$views"},
				},
			},
			{
				"$sort": map[string]interface{}{
					"totalViews": -1,
				},
			},
		}

		postAnalytics, err := docPostRepo.Aggregate(ctx, postAnalyticsPipeline)
		if err != nil {
			log.Printf("Failed to aggregate post analytics: %v", err)
		} else {
			fmt.Printf("✓ Blog post analytics by category:\n")
			for _, analytics := range postAnalytics {
				category := analytics["_id"]
				totalPosts := analytics["totalPosts"]
				publishedPosts := analytics["publishedPosts"]
				totalViews := analytics["totalViews"]
				totalLikes := analytics["totalLikes"]
				fmt.Printf("  %s: %v posts (%v published), %v views, %v likes\n",
					category, totalPosts, publishedPosts, totalViews, totalLikes)
			}
		}
	}

	// ============================================
	// Document Updates
	// ============================================
	fmt.Println("\n=== Document Updates ===")

	if docRepo, ok := userRepo.(gpa.DocumentRepository[User]); ok {
		// Update using MongoDB update operators
		updateResult, err := docRepo.UpdateDocument(ctx, users[0].ID, map[string]interface{}{
			"$set": map[string]interface{}{
				"age": 31,
				"updated_at": time.Now(),
			},
			"$addToSet": map[string]interface{}{
				"tags": "senior-developer",
			},
		})
		if err != nil {
			log.Printf("Failed to update user document: %v", err)
		} else {
			fmt.Printf("✓ Updated %d user document with MongoDB operators\n", updateResult)
		}

		// Update multiple documents
		manyUpdateResult, err := docRepo.UpdateManyDocuments(ctx,
			map[string]interface{}{
				"location.country": "USA",
				"is_active": true,
			},
			map[string]interface{}{
				"$inc": map[string]interface{}{
					"age": 1, // Increment age by 1
				},
				"$set": map[string]interface{}{
					"updated_at": time.Now(),
				},
			},
		)
		if err != nil {
			log.Printf("Failed to update many documents: %v", err)
		} else {
			fmt.Printf("✓ Updated %d active USA users (incremented age)\n", manyUpdateResult)
		}
	}

	// ============================================
	// Advanced Queries and Operations
	// ============================================
	fmt.Println("\n=== Advanced Queries ===")

	// Query with complex conditions using GPA syntax
	complexUsers, err := userRepo.Query(ctx,
		gpa.Where("is_active", gpa.OpEqual, true),
		gpa.Where("age", gpa.OpGreaterThan, 25),
		gpa.OrderBy("age", gpa.OrderDesc),
		gpa.Limit(5),
	)
	if err != nil {
		log.Printf("Failed to execute complex query: %v", err)
	} else {
		fmt.Printf("✓ Complex query found %d active users over 25\n", len(complexUsers))
		for _, user := range complexUsers {
			location := "Unknown"
			if user.Location != nil {
				location = user.Location.City
			}
			fmt.Printf("  %s (age %d) - %s\n", user.Name, user.Age, location)
		}
	}

	// Distinct values (using concrete MongoDB repository type)
	if mongoRepo, ok := userRepo.(*gpamongo.Repository[User]); ok {
		distinctCountries, err := mongoRepo.Distinct(ctx, "location.country", map[string]interface{}{})
		if err != nil {
			log.Printf("Failed to get distinct countries: %v", err)
		} else {
			fmt.Printf("✓ Found users from %d distinct countries: %v\n", 
				len(distinctCountries), distinctCountries)
		}

		distinctTags, err := mongoRepo.Distinct(ctx, "tags", map[string]interface{}{
			"is_active": true,
		})
		if err != nil {
			log.Printf("Failed to get distinct tags: %v", err)
		} else {
			fmt.Printf("✓ Active users have %d distinct tags\n", len(distinctTags))
		}
	}

	// ============================================
	// Index Management
	// ============================================
	fmt.Println("\n=== Index Management ===")

	if docRepo, ok := userRepo.(gpa.DocumentRepository[User]); ok {
		// Create compound index
		compoundIndexKeys := map[string]interface{}{
			"is_active": 1,
			"age":       -1,
			"location.country": 1,
		}
		err = docRepo.CreateIndex(ctx, compoundIndexKeys, false)
		if err != nil {
			log.Printf("Compound index might already exist: %v", err)
		} else {
			fmt.Println("✓ Created compound index on is_active, age, country")
		}

		// Create unique index
		uniqueIndexKeys := map[string]interface{}{
			"email": 1,
		}
		err = docRepo.CreateIndex(ctx, uniqueIndexKeys, true)
		if err != nil {
			log.Printf("Unique index might already exist: %v", err)
		} else {
			fmt.Println("✓ Created unique index on email")
		}
	}

	// ============================================
	// Transactions (if supported)
	// ============================================
	fmt.Println("\n=== MongoDB Transactions ===")

	err = userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
		// Create a new user in transaction
		newUser := &User{
			Name:     "Transaction User",
			Email:    "tx@example.com",
			Age:      29,
			IsActive: true,
			Tags:     []string{"test"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(ctx, newUser); err != nil {
			return err
		}

		// Update another user
		if err := tx.UpdatePartial(ctx, users[0].ID, map[string]interface{}{
			"updated_at": time.Now(),
		}); err != nil {
			return err
		}

		fmt.Println("✓ Transaction operations completed")
		return nil
	})
	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("✓ MongoDB transaction completed successfully")
	}

	// ============================================
	// Performance and Statistics
	// ============================================
	fmt.Println("\n=== Performance and Statistics ===")

	// Count operations
	totalUsers, _ := userRepo.Count(ctx)
	activeUsers, _ := userRepo.Count(ctx, gpa.Where("is_active", gpa.OpEqual, true))
	usaUsers, _ := userRepo.Count(ctx, gpa.Where("location.country", gpa.OpEqual, "USA"))

	fmt.Printf("✓ User statistics:\n")
	fmt.Printf("  Total users: %d\n", totalUsers)
	fmt.Printf("  Active users: %d\n", activeUsers)
	fmt.Printf("  USA users: %d\n", usaUsers)

	// Blog post statistics
	totalPosts, _ := postRepo.Count(ctx)
	publishedPosts, _ := postRepo.Count(ctx, gpa.Where("published", gpa.OpEqual, true))
	programmingPosts, _ := postRepo.Count(ctx, gpa.Where("category", gpa.OpEqual, "Programming"))

	fmt.Printf("✓ Blog post statistics:\n")
	fmt.Printf("  Total posts: %d\n", totalPosts)
	fmt.Printf("  Published posts: %d\n", publishedPosts)
	fmt.Printf("  Programming posts: %d\n", programmingPosts)

	// Existence checks
	hasSkillfulUsers, _ := userRepo.Exists(ctx, gpa.Where("profile.skills.0", gpa.OpExists, true))
	hasHighViewPosts, _ := postRepo.Exists(ctx, gpa.Where("views", gpa.OpGreaterThan, 100))

	fmt.Printf("✓ Existence checks:\n")
	fmt.Printf("  Users with skills: %t\n", hasSkillfulUsers)
	fmt.Printf("  Posts with >100 views: %t\n", hasHighViewPosts)

	// ============================================
	// Entity Information
	// ============================================
	fmt.Println("\n=== Entity Information ===")

	userEntityInfo, err := userRepo.GetEntityInfo()
	if err != nil {
		log.Printf("Failed to get user entity info: %v", err)
	} else {
		fmt.Printf("✓ User entity info:\n")
		fmt.Printf("  Name: %s\n", userEntityInfo.Name)
		fmt.Printf("  Collection: %s\n", userEntityInfo.TableName)
		fmt.Printf("  Fields: %d\n", len(userEntityInfo.Fields))
	}

	fmt.Printf("\n✓ Provider information:\n")
	providerInfo := userProvider.ProviderInfo()
	fmt.Printf("  Name: %s\n", providerInfo.Name)
	fmt.Printf("  Version: %s\n", providerInfo.Version)
	fmt.Printf("  Database Type: %s\n", providerInfo.DatabaseType)
	fmt.Printf("  Features: %v\n", providerInfo.Features)

	fmt.Println("\n🎉 MongoDB provider example completed!")
}