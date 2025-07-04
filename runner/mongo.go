//go:build ignore

// Package main demonstrates how to use the GPA framework with MongoDB adapter
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpamongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// =====================================
// Domain Models with MongoDB Tags
// =====================================

// User represents a user entity for MongoDB
type User struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Email     string                 `bson:"email" json:"email"`
	Name      string                 `bson:"name" json:"name"`
	Age       int                    `bson:"age" json:"age"`
	Status    string                 `bson:"status" json:"status"`
	Profile   UserProfile            `bson:"profile" json:"profile"`
	Tags      []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`

	// Relationships (embedded or referenced)
	Orders []Order `bson:"orders,omitempty" json:"orders,omitempty"`
}

func (u User) CollectionName() string { return "users" }

// UserProfile represents nested user profile data
type UserProfile struct {
	Bio       string   `bson:"bio,omitempty" json:"bio,omitempty"`
	Avatar    string   `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Interests []string `bson:"interests,omitempty" json:"interests,omitempty"`
	Location  Location `bson:"location,omitempty" json:"location,omitempty"`
	Verified  bool     `bson:"verified" json:"verified"`
}

// Location represents geographical location
type Location struct {
	Country   string  `bson:"country,omitempty" json:"country,omitempty"`
	City      string  `bson:"city,omitempty" json:"city,omitempty"`
	Latitude  float64 `bson:"latitude,omitempty" json:"latitude,omitempty"`
	Longitude float64 `bson:"longitude,omitempty" json:"longitude,omitempty"`
}

// Order represents an order entity for MongoDB
type Order struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID `bson:"user_id" json:"user_id"`
	ProductName     string             `bson:"product_name" json:"product_name"`
	Amount          float64            `bson:"amount" json:"amount"`
	Currency        string             `bson:"currency" json:"currency"`
	Status          string             `bson:"status" json:"status"`
	Items           []OrderItem        `bson:"items" json:"items"`
	ShippingAddress Address            `bson:"shipping_address" json:"shipping_address"`
	OrderDate       time.Time          `bson:"order_date" json:"order_date"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

func (o Order) CollectionName() string { return "orders" }

// OrderItem represents an item within an order
type OrderItem struct {
	ProductID primitive.ObjectID `bson:"product_id" json:"product_id"`
	Name      string             `bson:"name" json:"name"`
	Quantity  int                `bson:"quantity" json:"quantity"`
	Price     float64            `bson:"price" json:"price"`
}

// Address represents a shipping address
type Address struct {
	Street  string `bson:"street" json:"street"`
	City    string `bson:"city" json:"city"`
	State   string `bson:"state" json:"state"`
	ZipCode string `bson:"zip_code" json:"zip_code"`
	Country string `bson:"country" json:"country"`
}

// Product represents a product entity for MongoDB
type Product struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	Description string                 `bson:"description" json:"description"`
	Price       float64                `bson:"price" json:"price"`
	Currency    string                 `bson:"currency" json:"currency"`
	Stock       int                    `bson:"stock" json:"stock"`
	Category    string                 `bson:"category" json:"category"`
	Tags        []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	Images      []string               `bson:"images,omitempty" json:"images,omitempty"`
	Attributes  map[string]interface{} `bson:"attributes,omitempty" json:"attributes,omitempty"`
	IsActive    bool                   `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

func (p Product) CollectionName() string { return "products" }

// Category represents a product category for MongoDB
type Category struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	Description string                 `bson:"description,omitempty" json:"description,omitempty"`
	ParentID    *primitive.ObjectID    `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
	Path        []string               `bson:"path" json:"path"` // For hierarchical categories
	Level       int                    `bson:"level" json:"level"`
	IsActive    bool                   `bson:"is_active" json:"is_active"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

func (c Category) CollectionName() string { return "categories" }

// =====================================
// Service Layer for MongoDB
// =====================================

// UserService provides business logic for user operations
type UserService struct {
	userRepo     gpa.Repository
	orderRepo    gpa.Repository
	productRepo  gpa.Repository
	categoryRepo gpa.Repository
	provider     gpa.Provider
}

// NewUserService creates a new user service
func NewUserService(provider gpa.Provider) *UserService {
	return &UserService{
		userRepo:     provider.RepositoryFor(&User{}),
		orderRepo:    provider.RepositoryFor(&Order{}),
		productRepo:  provider.RepositoryFor(&Product{}),
		categoryRepo: provider.RepositoryFor(&Category{}),
		provider:     provider,
	}
}

// CreateUser creates a new user with validation
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	// Business logic validation
	if user.Age < 18 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "user must be at least 18 years old",
		}
	}

	// Check if email already exists
	exists, err := s.userRepo.Exists(ctx,
		gpa.Where("email", gpa.OpEqual, user.Email))
	if err != nil {
		return fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: "email already exists",
		}
	}

	// Set default values
	user.Status = "active"
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Tags == nil {
		user.Tags = []string{}
	}
	if user.Metadata == nil {
		user.Metadata = make(map[string]interface{})
	}

	return s.userRepo.Create(ctx, user)
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID primitive.ObjectID) (*User, error) {
	var user User
	err := s.userRepo.FindByID(ctx, userID, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// GetActiveUsers retrieves all active users with pagination
func (s *UserService) GetActiveUsers(ctx context.Context, limit, offset int) ([]User, error) {
	var users []User
	err := s.userRepo.Query(ctx, &users,
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrderBy("created_at", gpa.OrderDesc),
		gpa.Limit(limit),
		gpa.Offset(offset),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}
	return users, nil
}

// SearchUsers searches users by name, email, or tags
func (s *UserService) SearchUsers(ctx context.Context, query string) ([]User, error) {
	var users []User

	// MongoDB supports more flexible text search
	err := s.userRepo.Query(ctx, &users,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpRegex, "(?i)"+query), // Case-insensitive regex
			gpa.WhereCondition("email", gpa.OpRegex, "(?i)"+query),
			gpa.WhereCondition("tags", gpa.OpIn, []string{query}), // Search in tags array
		),
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrderBy("name", gpa.OrderAsc),
		gpa.Limit(50),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

// UpdateUserProfile updates a user's profile using MongoDB's nested document updates
func (s *UserService) UpdateUserProfile(ctx context.Context, userID primitive.ObjectID, profile UserProfile) error {
	updates := map[string]interface{}{
		"profile":    profile,
		"updated_at": time.Now(),
	}
	return s.userRepo.UpdatePartial(ctx, userID, updates)
}

// AddUserTag adds a tag to a user's tags array
func (s *UserService) AddUserTag(ctx context.Context, userID primitive.ObjectID, tag string) error {
	// For now, use standard update operations
	// Fetch current user first
	var user User
	err := s.userRepo.FindByID(ctx, userID, &user)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Check if tag already exists
	for _, existingTag := range user.Tags {
		if existingTag == tag {
			return nil // Tag already exists
		}
	}

	// Add tag to the slice
	user.Tags = append(user.Tags, tag)
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, &user)
}

// GetUsersByLocation finds users near a location using MongoDB geospatial queries
func (s *UserService) GetUsersByLocation(ctx context.Context, lat, lon float64, maxDistance float64) ([]User, error) {
	if mongoRepo, ok := s.userRepo.(*gpamongo.Repository); ok {
		// Use MongoDB aggregation for geospatial search
		pipeline := []map[string]interface{}{
			{
				"$match": map[string]interface{}{
					"profile.location": map[string]interface{}{
						"$near": map[string]interface{}{
							"$geometry": map[string]interface{}{
								"type":        "Point",
								"coordinates": []float64{lon, lat},
							},
							"$maxDistance": maxDistance,
						},
					},
					"status": "active",
				},
			},
			{"$limit": 50},
		}

		var users []User
		err := mongoRepo.Aggregate(ctx, pipeline, &users)
		if err != nil {
			return nil, fmt.Errorf("failed geospatial search: %w", err)
		}
		return users, nil
	}

	return nil, gpa.GPAError{
		Type:    gpa.ErrorTypeUnsupported,
		Message: "geospatial queries not supported",
	}
}

// =====================================
// MongoDB-Specific Aggregation Examples
// =====================================

// GetUserStatistics returns user statistics using MongoDB aggregation
func (s *UserService) GetUserStatistics(ctx context.Context) (map[string]interface{}, error) {
	if mongoRepo, ok := s.userRepo.(*gpamongo.Repository); ok {
		pipeline := []map[string]interface{}{
			{
				"$group": map[string]interface{}{
					"_id": nil,
					"total_users": map[string]interface{}{
						"$sum": 1,
					},
					"active_users": map[string]interface{}{
						"$sum": map[string]interface{}{
							"$cond": []interface{}{
								map[string]interface{}{"$eq": []string{"$status", "active"}},
								1,
								0,
							},
						},
					},
					"average_age": map[string]interface{}{
						"$avg": "$age",
					},
					"age_distribution": map[string]interface{}{
						"$push": "$age",
					},
				},
			},
		}

		var results []map[string]interface{}
		err := mongoRepo.Aggregate(ctx, pipeline, &results)
		if err != nil {
			return nil, fmt.Errorf("failed to get statistics: %w", err)
		}

		if len(results) > 0 {
			return results[0], nil
		}
		return map[string]interface{}{}, nil
	}

	return nil, gpa.GPAError{
		Type:    gpa.ErrorTypeUnsupported,
		Message: "aggregation not supported",
	}
}

// GetUsersByAgeGroup groups users by age ranges
func (s *UserService) GetUsersByAgeGroup(ctx context.Context) ([]map[string]interface{}, error) {
	if mongoRepo, ok := s.userRepo.(*gpamongo.Repository); ok {
		pipeline := []map[string]interface{}{
			{
				"$match": map[string]interface{}{
					"status": "active",
				},
			},
			{
				"$group": map[string]interface{}{
					"_id": map[string]interface{}{
						"$switch": map[string]interface{}{
							"branches": []map[string]interface{}{
								{
									"case": map[string]interface{}{"$lt": []interface{}{"$age", 25}},
									"then": "18-24",
								},
								{
									"case": map[string]interface{}{"$lt": []interface{}{"$age", 35}},
									"then": "25-34",
								},
								{
									"case": map[string]interface{}{"$lt": []interface{}{"$age", 45}},
									"then": "35-44",
								},
								{
									"case": map[string]interface{}{"$lt": []interface{}{"$age", 55}},
									"then": "45-54",
								},
							},
							"default": "55+",
						},
					},
					"count": map[string]interface{}{"$sum": 1},
					"users": map[string]interface{}{
						"$push": map[string]interface{}{
							"name":  "$name",
							"email": "$email",
							"age":   "$age",
						},
					},
				},
			},
			{
				"$sort": map[string]interface{}{"_id": 1},
			},
		}

		var results []map[string]interface{}
		err := mongoRepo.Aggregate(ctx, pipeline, &results)
		if err != nil {
			return nil, fmt.Errorf("failed to group by age: %w", err)
		}
		return results, nil
	}

	return nil, gpa.GPAError{
		Type:    gpa.ErrorTypeUnsupported,
		Message: "aggregation not supported",
	}
}

// =====================================
// MongoDB Transaction Examples
// =====================================

// CreateUserWithFirstOrder creates a user and their first order in a transaction
func (s *UserService) CreateUserWithFirstOrder(ctx context.Context, user *User, order *Order) error {
	return s.userRepo.Transaction(ctx, func(tx gpa.Transaction) error {
		// Create user first
		if err := tx.Create(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Set the user ID for the order
		order.UserID = user.ID
		order.OrderDate = time.Now()
		order.CreatedAt = time.Now()
		order.UpdatedAt = time.Now()

		// Create order
		if err := tx.Create(ctx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		return nil
	})
}

// =====================================
// Main Application
// =====================================

func main() {
	// Initialize the application
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer app.Close()

	// Run examples
	ctx := context.Background()

	// Basic CRUD operations
	fmt.Println("=== Running Basic MongoDB CRUD Examples ===")
	if err := runBasicCRUDExamples(ctx, app); err != nil {
		log.Printf("Basic CRUD examples failed: %v", err)
	}

	// Document operations
	fmt.Println("\n=== Running MongoDB Document Operations ===")
	if err := runDocumentOperations(ctx, app); err != nil {
		log.Printf("Document operations failed: %v", err)
	}

	// Array and nested document operations
	fmt.Println("\n=== Running MongoDB Array & Nested Document Operations ===")
	if err := runArrayAndNestedOperations(ctx, app); err != nil {
		log.Printf("Array operations failed: %v", err)
	}

	// Aggregation examples
	fmt.Println("\n=== Running MongoDB Aggregation Examples ===")
	if err := runAggregationExamples(ctx, app); err != nil {
		log.Printf("Aggregation examples failed: %v", err)
	}

	// Transaction examples
	fmt.Println("\n=== Running MongoDB Transaction Examples ===")
	if err := runTransactionExamples(ctx, app); err != nil {
		log.Printf("Transaction examples failed: %v", err)
	}

	// Index and performance examples
	fmt.Println("\n=== Running MongoDB Index Examples ===")
	if err := runIndexExamples(ctx, app); err != nil {
		log.Printf("Index examples failed: %v", err)
	}
}

// =====================================
// Application Setup
// =====================================

// App represents the main application
type App struct {
	provider    gpa.Provider
	userService *UserService
}

// NewApp creates and initializes the application
func NewApp() (*App, error) {
	// MongoDB configuration
	config := gpa.Config{
		Driver:   "mongodb",
		Host:     "localhost",
		Port:     27017,
		Database: "gpa_mongodb_example",

		// MongoDB-specific options
		Options: map[string]interface{}{
			"mongo": map[string]interface{}{
				"max_pool_size": 10,
				"min_pool_size": 1,
				"max_idle_time": time.Minute * 30,
			},
		},

		// For authenticated MongoDB
		// Username: "username",
		// Password: "password",

		// SSL configuration for MongoDB Atlas or secure deployments
		// SSL: gpa.SSLConfig{
		// 	Enabled: true,
		// 	Mode:    "require",
		// },
	}

	// For MongoDB Atlas (cloud)
	if false { // Set to true to use MongoDB Atlas
		config = gpa.Config{
			Driver:        "mongodb",
			ConnectionURL: "mongodb+srv://username:password@cluster.mongodb.net/database?retryWrites=true&w=majority",
			Database:      "gpa_mongodb_example",
		}
	}

	// Create provider
	provider, err := gpa.NewProvider("mongodb", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB provider: %w", err)
	}

	// Test connection
	if err := provider.Health(); err != nil {
		return nil, fmt.Errorf("MongoDB health check failed: %w", err)
	}

	fmt.Println("✅ Connected to MongoDB successfully!")

	// Initialize services
	userService := NewUserService(provider)

	return &App{
		provider:    provider,
		userService: userService,
	}, nil
}

// Close closes the application and its resources
func (app *App) Close() error {
	fmt.Println("Closing MongoDB connection...")
	return app.provider.Close()
}

// =====================================
// Example Functions
// =====================================

// runBasicCRUDExamples demonstrates basic CRUD operations with MongoDB
func runBasicCRUDExamples(ctx context.Context, app *App) error {
	fmt.Println("1. Creating users with nested documents...")

	// Create users with nested profile data
	users := []*User{
		{
			Name:  "Alice Johnson",
			Email: "alice@example.com",
			Age:   28,
			Tags:  []string{"developer", "mongodb", "go"},
			Profile: UserProfile{
				Bio:       "Software developer passionate about databases",
				Avatar:    "https://example.com/alice.jpg",
				Interests: []string{"coding", "databases", "travel"},
				Location: Location{
					Country:   "USA",
					City:      "San Francisco",
					Latitude:  37.7749,
					Longitude: -122.4194,
				},
				Verified: true,
			},
			Metadata: map[string]interface{}{
				"source":           "signup",
				"marketing_opt_in": true,
				"referral_code":    "REF123",
			},
		},
		{
			Name:  "Bob Smith",
			Email: "bob@example.com",
			Age:   35,
			Tags:  []string{"manager", "analytics"},
			Profile: UserProfile{
				Bio:       "Data analytics manager",
				Interests: []string{"data", "analytics", "leadership"},
				Location: Location{
					Country: "Canada",
					City:    "Toronto",
				},
				Verified: false,
			},
			Metadata: map[string]interface{}{
				"source":      "referral",
				"department":  "analytics",
				"employee_id": "EMP001",
			},
		},
	}

	for _, user := range users {
		if err := app.userService.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user %s: %w", user.Name, err)
		}
		fmt.Printf("✅ Created user: %s (ID: %s)\n", user.Name, user.ID.Hex())
	}

	// Read user with nested data
	fmt.Println("\n2. Reading user with nested documents...")
	user, err := app.userService.GetUserByID(ctx, users[0].ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	fmt.Printf("📖 Retrieved user: %s\n", user.Name)
	fmt.Printf("   Bio: %s\n", user.Profile.Bio)
	fmt.Printf("   Location: %s, %s\n", user.Profile.Location.City, user.Profile.Location.Country)
	fmt.Printf("   Tags: %v\n", user.Tags)
	fmt.Printf("   Metadata: %v\n", user.Metadata)

	// Update nested document
	fmt.Println("\n3. Updating nested profile...")
	newProfile := UserProfile{
		Bio:       "Senior Software Developer",
		Avatar:    "https://example.com/alice-new.jpg",
		Interests: []string{"coding", "databases", "travel", "photography"},
		Location: Location{
			Country:   "USA",
			City:      "Seattle",
			Latitude:  47.6062,
			Longitude: -122.3321,
		},
		Verified: true,
	}
	if err := app.userService.UpdateUserProfile(ctx, user.ID, newProfile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}
	fmt.Printf("✅ Updated profile for user %s\n", user.Name)

	// List users with filtering
	fmt.Println("\n4. Listing active users...")
	activeUsers, err := app.userService.GetActiveUsers(ctx, 10, 0)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}
	fmt.Printf("📋 Found %d active users:\n", len(activeUsers))
	for _, u := range activeUsers {
		fmt.Printf("   - %s (%s) from %s\n", u.Name, u.Email, u.Profile.Location.City)
	}

	return nil
}

// runDocumentOperations demonstrates MongoDB document-specific operations
func runDocumentOperations(ctx context.Context, app *App) error {
	fmt.Println("1. Document-based queries...")

	userRepo := app.userService.userRepo
	if mongoRepo, ok := userRepo.(*gpamongo.Repository); ok {
		// Find by document (native MongoDB operation)
		document := map[string]interface{}{
			"profile.verified": true,
			"age":              map[string]interface{}{"$gte": 25},
		}

		var users []User
		err := mongoRepo.FindByDocument(ctx, document, &users)
		if err != nil {
			return fmt.Errorf("failed document query: %w", err)
		}
		fmt.Printf("📋 Found %d verified users over 25\n", len(users))

		// Update using document
		if len(users) > 0 {
			updateDoc := map[string]interface{}{
				"metadata.last_verified": time.Now(),
				"updated_at":             time.Now(),
			}
			err = mongoRepo.UpdateDocument(ctx, users[0].ID, updateDoc)
			if err != nil {
				return fmt.Errorf("failed document update: %w", err)
			}
			fmt.Printf("✅ Updated verification timestamp for %s\n", users[0].Name)
		}
	}

	fmt.Println("\n2. Complex nested queries...")

	// Query users by nested location
	var usersInUSA []User
	err := userRepo.Query(ctx, &usersInUSA,
		gpa.Where("profile.location.country", gpa.OpEqual, "USA"),
		gpa.Where("status", gpa.OpEqual, "active"),
	)
	if err != nil {
		return fmt.Errorf("failed nested query: %w", err)
	}
	fmt.Printf("🇺🇸 Found %d users in USA\n", len(usersInUSA))

	// Query by array content
	var developersUsers []User
	err = userRepo.Query(ctx, &developersUsers,
		gpa.Where("tags", gpa.OpIn, []string{"developer"}),
	)
	if err != nil {
		return fmt.Errorf("failed array query: %w", err)
	}
	fmt.Printf("👩‍💻 Found %d developers\n", len(developersUsers))

	return nil
}

// runArrayAndNestedOperations demonstrates MongoDB array and nested operations
func runArrayAndNestedOperations(ctx context.Context, app *App) error {
	fmt.Println("1. Array operations...")

	// Get first user to work with
	users, err := app.userService.GetActiveUsers(ctx, 1, 0)
	if err != nil || len(users) == 0 {
		return fmt.Errorf("no users found for array operations")
	}
	user := users[0]

	// Add tags to user
	fmt.Printf("Adding tags to user %s...\n", user.Name)
	tags := []string{"expert", "team-lead", "mentor"}
	for _, tag := range tags {
		if err := app.userService.AddUserTag(ctx, user.ID, tag); err != nil {
			log.Printf("Warning: Failed to add tag %s: %v", tag, err)
		}
	}

	// Verify tags were added
	updatedUser, err := app.userService.GetUserByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get updated user: %w", err)
	}
	fmt.Printf("✅ User now has tags: %v\n", updatedUser.Tags)

	fmt.Println("\n2. Creating products with arrays...")

	productRepo := app.userService.productRepo
	products := []*Product{
		{
			Name:        "MacBook Pro",
			Description: "Apple laptop for professionals",
			Price:       2499.99,
			Currency:    "USD",
			Stock:       50,
			Category:    "electronics",
			Tags:        []string{"laptop", "apple", "professional", "high-end"},
			Images:      []string{"macbook1.jpg", "macbook2.jpg", "macbook3.jpg"},
			Attributes: map[string]interface{}{
				"brand":       "Apple",
				"model":       "MacBook Pro 16-inch",
				"processor":   "M3 Pro",
				"memory":      "32GB",
				"storage":     "1TB SSD",
				"screen_size": 16.2,
				"color":       "Space Gray",
				"warranty":    "1 year",
			},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "Wireless Headphones",
			Description: "Premium noise-canceling headphones",
			Price:       349.99,
			Currency:    "USD",
			Stock:       100,
			Category:    "electronics",
			Tags:        []string{"headphones", "wireless", "noise-canceling", "premium"},
			Images:      []string{"headphones1.jpg", "headphones2.jpg"},
			Attributes: map[string]interface{}{
				"brand":           "Sony",
				"model":           "WH-1000XM4",
				"battery_life":    "30 hours",
				"noise_canceling": true,
				"wireless":        true,
				"color":           "Black",
			},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, product := range products {
		if err := productRepo.Create(ctx, product); err != nil {
			return fmt.Errorf("failed to create product %s: %w", product.Name, err)
		}
		fmt.Printf("✅ Created product: %s (ID: %s)\n", product.Name, product.ID.Hex())
	}

	// Query products by tags
	var laptops []Product
	err = productRepo.Query(ctx, &laptops,
		gpa.Where("tags", gpa.OpIn, []string{"laptop"}),
		gpa.Where("is_active", gpa.OpEqual, true),
	)
	if err != nil {
		return fmt.Errorf("failed to query laptops: %w", err)
	}
	fmt.Printf("💻 Found %d laptops\n", len(laptops))

	return nil
}

// runAggregationExamples demonstrates MongoDB aggregation pipeline
func runAggregationExamples(ctx context.Context, app *App) error {
	fmt.Println("1. User statistics aggregation...")

	stats, err := app.userService.GetUserStatistics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}
	fmt.Printf("📊 User Statistics:\n")
	fmt.Printf("   Total Users: %.0f\n", stats["total_users"])
	fmt.Printf("   Active Users: %.0f\n", stats["active_users"])
	fmt.Printf("   Average Age: %.1f\n", stats["average_age"])

	fmt.Println("\n2. Age group distribution...")
	ageGroups, err := app.userService.GetUsersByAgeGroup(ctx)
	if err != nil {
		return fmt.Errorf("failed to get age groups: %w", err)
	}
	fmt.Printf("📊 Age Distribution:\n")
	for _, group := range ageGroups {
		fmt.Printf("   %s: %.0f users\n", group["_id"], group["count"])
	}

	fmt.Println("\n3. Product aggregation...")
	productRepo := app.userService.productRepo
	if mongoRepo, ok := productRepo.(*gpamongo.Repository); ok {
		// Aggregate products by category
		pipeline := []map[string]interface{}{
			{
				"$match": map[string]interface{}{
					"is_active": true,
				},
			},
			{
				"$group": map[string]interface{}{
					"_id":            "$category",
					"total_products": map[string]interface{}{"$sum": 1},
					"total_value": map[string]interface{}{"$sum": map[string]interface{}{
						"$multiply": []string{"$price", "$stock"},
					}},
					"avg_price": map[string]interface{}{"$avg": "$price"},
					"max_price": map[string]interface{}{"$max": "$price"},
					"min_price": map[string]interface{}{"$min": "$price"},
				},
			},
			{
				"$sort": map[string]interface{}{"total_value": -1},
			},
		}

		var categoryStats []map[string]interface{}
		err = mongoRepo.Aggregate(ctx, pipeline, &categoryStats)
		if err != nil {
			return fmt.Errorf("failed product aggregation: %w", err)
		}

		fmt.Printf("📦 Product Statistics by Category:\n")
		for _, stat := range categoryStats {
			fmt.Printf("   %s: %v products, avg price: $%.2f, total value: $%.2f\n",
				stat["_id"], stat["total_products"], stat["avg_price"], stat["total_value"])
		}
	}

	return nil
}

// runTransactionExamples demonstrates MongoDB transactions
func runTransactionExamples(ctx context.Context, app *App) error {
	fmt.Println("1. Creating user with first order in transaction...")

	user := &User{
		Name:  "Transaction User",
		Email: "transaction@example.com",
		Age:   30,
		Profile: UserProfile{
			Bio:      "User created in transaction",
			Verified: false,
		},
		Tags:      []string{"new", "transaction"},
		Metadata:  map[string]interface{}{"created_in": "transaction"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	order := &Order{
		ProductName: "Laptop",
		Amount:      999.99,
		Currency:    "USD",
		Status:      "pending",
		Items: []OrderItem{
			{
				Name:     "MacBook Air",
				Quantity: 1,
				Price:    999.99,
			},
		},
		ShippingAddress: Address{
			Street:  "123 Main St",
			City:    "New York",
			State:   "NY",
			ZipCode: "10001",
			Country: "USA",
		},
	}

	if err := app.userService.CreateUserWithFirstOrder(ctx, user, order); err != nil {
		log.Printf("Transaction failed (may not be supported in standalone MongoDB): %v", err)
		fmt.Println("⚠️  Transactions may require MongoDB replica set")
		return nil
	}

	fmt.Printf("✅ Created user %s with order %s in transaction\n", user.Name, order.ID.Hex())

	return nil
}

// runIndexExamples demonstrates MongoDB indexing
func runIndexExamples(ctx context.Context, app *App) error {
	fmt.Println("1. Creating indexes...")
	userRepo := app.userService.userRepo
	if mongoRepo, ok := userRepo.(*gpamongo.Repository); ok {
		// Create text index for search
		textIndex := bson.D{
			{Key: "name", Value: "text"},
			{Key: "email", Value: "text"},
			{Key: "profile.bio", Value: "text"},
		}
		if err := mongoRepo.CreateIndex(ctx, textIndex, false); err != nil {
			log.Printf("Warning: Failed to create text index: %v", err)
		} else {
			fmt.Println("✅ Created text search index")
		}

		// Create compound index
		compoundIndex := bson.D{
			{Key: "status", Value: 1},
			{Key: "age", Value: 1},
			{Key: "created_at", Value: -1},
		}
		if err := mongoRepo.CreateIndex(ctx, compoundIndex, false); err != nil {
			log.Printf("Warning: Failed to create compound index: %v", err)
		} else {
			fmt.Println("✅ Created compound index")
		}

		// Create geospatial index
		geoIndex := bson.D{
			{Key: "profile.location", Value: "2dsphere"},
		}
		if err := mongoRepo.CreateIndex(ctx, geoIndex, false); err != nil {
			log.Printf("Warning: Failed to create geo index: %v", err)
		} else {
			fmt.Println("✅ Created geospatial index")
		}

		// List all indexes
		indexes, err := mongoRepo.ListIndexes(ctx)
		if err != nil {
			log.Printf("Warning: Failed to list indexes: %v", err)
		} else {
			fmt.Printf("📋 Total indexes: %d\n", len(indexes))
			for _, index := range indexes {
				if name, ok := index["name"]; ok {
					fmt.Printf("   - %s\n", name)
				}
			}
		}
	}

	return nil
}

// =====================================
// Configuration Examples
// =====================================

// ExampleWithMongoDBAtlas shows how to configure MongoDB Atlas
func ExampleWithMongoDBAtlas() gpa.Config {
	return gpa.Config{
		Driver:        "mongodb",
		ConnectionURL: "mongodb+srv://username:password@cluster.mongodb.net/database?retryWrites=true&w=majority",
		Database:      "production_db",

		Options: map[string]interface{}{
			"mongo": map[string]interface{}{
				"max_pool_size": 20,
				"min_pool_size": 5,
				"max_idle_time": time.Hour,
			},
		},
	}
}

// ExampleWithLocalMongoDB shows how to configure local MongoDB
func ExampleWithLocalMongoDB() gpa.Config {
	return gpa.Config{
		Driver:   "mongodb",
		Host:     "localhost",
		Port:     27017,
		Database: "local_dev_db",
		Username: "dev_user",
		Password: "dev_password",

		Options: map[string]interface{}{
			"mongo": map[string]interface{}{
				"max_pool_size": 10,
				"min_pool_size": 2,
			},
		},
	}
}

// ExampleWithMongoDBReplicaSet shows how to configure MongoDB replica set
func ExampleWithMongoDBReplicaSet() gpa.Config {
	return gpa.Config{
		Driver:        "mongodb",
		ConnectionURL: "mongodb://user:password@host1:27017,host2:27017,host3:27017/database?replicaSet=myReplicaSet",
		Database:      "replica_db",

		SSL: gpa.SSLConfig{
			Enabled: true,
			Mode:    "require",
		},

		Options: map[string]interface{}{
			"mongo": map[string]interface{}{
				"max_pool_size": 25,
				"min_pool_size": 5,
				"max_idle_time": time.Minute * 30,
			},
		},
	}
}
