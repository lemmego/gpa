//go:build ignore

// Package main demonstrates how to use the GPA framework with GORM adapter
package main

import (
	"context"
	"fmt"
	"github.com/lemmego/gpa/gpagorm"
	"log"
	"time"

	"github.com/lemmego/gpa"
)

// =====================================
// Domain Models
// =====================================

// User represents a user entity
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Age       int       `gorm:"not null" json:"age"`
	Status    string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Orders []Order `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

// Order represents an order entity
type Order struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	ProductName string    `gorm:"size:255;not null" json:"product_name"`
	Amount      float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status      string    `gorm:"size:20;default:'pending'" json:"status"`
	OrderDate   time.Time `gorm:"not null" json:"order_date"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Product represents a product entity
type Product struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null;index" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock       int       `gorm:"not null;default:0" json:"stock"`
	CategoryID  uint      `gorm:"not null;index" json:"category_id"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Category represents a product category
type Category struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"size:100;not null;uniqueIndex" json:"name"`
	ParentID *uint  `gorm:"index" json:"parent_id,omitempty"`

	// Self-referencing relationship
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// =====================================
// Service Layer
// =====================================

// UserService provides business logic for user operations
type UserService struct {
	userRepo  gpa.Repository
	orderRepo gpa.Repository
	provider  gpa.Provider
}

// NewUserService creates a new user service
func NewUserService(provider gpa.Provider) *UserService {
	return &UserService{
		userRepo:  provider.RepositoryFor(&User{}),
		orderRepo: provider.RepositoryFor(&Order{}),
		provider:  provider,
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

	return s.userRepo.Create(ctx, user)
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*User, error) {
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

// SearchUsers searches users by name or email
func (s *UserService) SearchUsers(ctx context.Context, query string) ([]User, error) {
	var users []User

	// Create composite condition for searching - Method 1: Using OrOption
	err := s.userRepo.Query(ctx, &users,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%"+query+"%"),
			gpa.WhereCondition("email", gpa.OpLike, "%"+query+"%"),
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

// UpdateUserStatus updates a user's status
func (s *UserService) UpdateUserStatus(ctx context.Context, userID uint, status string) error {
	return s.userRepo.UpdatePartial(ctx, userID, map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})
}

// GetUserWithOrders retrieves a user with their orders using preloading
func (s *UserService) GetUserWithOrders(ctx context.Context, userID uint) (*User, error) {
	var user User

	// Method 1: Using preloading (recommended for GORM)
	err := s.userRepo.Query(ctx, &user,
		gpa.Where("id", gpa.OpEqual, userID),
		gpa.Preload("Orders"), // Preload the Orders relationship
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with orders: %w", err)
	}

	return &user, nil
}

// GetUserWithOrdersAndDetails retrieves a user with orders and nested relationships
func (s *UserService) GetUserWithOrdersAndDetails(ctx context.Context, userID uint) (*User, error) {
	var user User

	// Preload multiple levels of relationships
	err := s.userRepo.Query(ctx, &user,
		gpa.Where("id", gpa.OpEqual, userID),
		gpa.Preload("Orders"),      // Load user's orders
		gpa.Preload("Orders.User"), // Load user info in each order (if needed)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with detailed orders: %w", err)
	}

	return &user, nil
}

// GetUsersWithRecentOrders gets users with their recent orders using joins
func (s *UserService) GetUsersWithRecentOrders(ctx context.Context, days int) ([]User, error) {
	var users []User

	// Method 2: Using joins for filtering
	cutoffDate := time.Now().AddDate(0, 0, -days)

	err := s.userRepo.Query(ctx, &users,
		gpa.Join(gpa.JoinInner, "orders", "orders.user_id = users.id"),
		gpa.Where("orders.order_date", gpa.OpGreaterThan, cutoffDate),
		gpa.Where("users.status", gpa.OpEqual, "active"),
		gpa.Preload("Orders"), // Still preload to get all orders for each user
		gpa.OrderBy("users.name", gpa.OrderAsc),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with recent orders: %w", err)
	}

	return users, nil
}

// ManageUserOrders demonstrates association management
func (s *UserService) ManageUserOrders(ctx context.Context, userID uint) error {
	// Get the user first
	var user User
	err := s.userRepo.FindByID(ctx, userID, &user)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Get GORM repository to access association methods
	if gormRepo, ok := s.userRepo.(*gpagorm.Repository); ok {
		// Get association manager for Orders
		ordersAssoc := gormRepo.Association(ctx, &user, "Orders")

		// Count existing orders
		count, err := ordersAssoc.Count()
		if err != nil {
			return fmt.Errorf("failed to count orders: %w", err)
		}
		fmt.Printf("User %d has %d orders\n", userID, count)

		// Add a new order
		newOrder := &Order{
			ProductName: "New Product",
			Amount:      199.99,
			Status:      "pending",
			OrderDate:   time.Now(),
		}

		err = ordersAssoc.Append(newOrder)
		if err != nil {
			return fmt.Errorf("failed to add order: %w", err)
		}

		fmt.Printf("Added new order with ID: %d\n", newOrder.ID)
	}

	return nil
}

// =====================================
// Relationship Examples
// =====================================

// runRelationshipExamples demonstrates relationship usage
func runRelationshipExamples(ctx context.Context, app *App) error {
	userRepo := app.provider.RepositoryFor(&User{})
	orderRepo := app.provider.RepositoryFor(&Order{})

	fmt.Println("1. Preloading relationships...")

	// Find users with their orders preloaded
	var usersWithOrders []User
	err := userRepo.Query(ctx, &usersWithOrders,
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.Preload("Orders"), // Preload Orders relationship
		gpa.Limit(3),
	)
	if err != nil {
		return fmt.Errorf("failed to preload orders: %w", err)
	}

	fmt.Printf("Found %d users with preloaded orders:\n", len(usersWithOrders))
	for _, user := range usersWithOrders {
		fmt.Printf("  - %s has %d orders\n", user.Name, len(user.Orders))
		for _, order := range user.Orders {
			fmt.Printf("    * Order #%d: %s ($%.2f)\n", order.ID, order.ProductName, order.Amount)
		}
	}

	// Find specific user with orders
	fmt.Println("\n2. Getting specific user with orders...")
	if len(usersWithOrders) > 0 {
		userWithOrders, err := app.userService.GetUserWithOrders(ctx, usersWithOrders[0].ID)
		if err != nil {
			return fmt.Errorf("failed to get user with orders: %w", err)
		}
		fmt.Printf("User %s has %d orders loaded via preloading\n", userWithOrders.Name, len(userWithOrders.Orders))
	}

	fmt.Println("\n3. Using joins for filtering...")

	// Get users who have made orders in the last 30 days using joins
	recentUsers, err := app.userService.GetUsersWithRecentOrders(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to get users with recent orders: %w", err)
	}
	fmt.Printf("Found %d users with recent orders\n", len(recentUsers))

	fmt.Println("\n4. Association management...")

	// Demonstrate association management
	if len(usersWithOrders) > 0 {
		err := app.userService.ManageUserOrders(ctx, usersWithOrders[0].ID)
		if err != nil {
			return fmt.Errorf("failed to manage user orders: %w", err)
		}
	}

	fmt.Println("\n5. Complex relationship queries...")

	// Orders with user information
	var ordersWithUsers []Order
	err = orderRepo.Query(ctx, &ordersWithUsers,
		gpa.Where("amount", gpa.OpGreaterThan, 500),
		gpa.Preload("User"), // Preload User relationship
		gpa.OrderBy("amount", gpa.OrderDesc),
		gpa.Limit(5),
	)
	if err != nil {
		return fmt.Errorf("failed to get orders with users: %w", err)
	}

	fmt.Printf("High-value orders:\n")
	for _, order := range ordersWithUsers {
		fmt.Printf("  - Order #%d: %s ($%.2f) by %s\n",
			order.ID, order.ProductName, order.Amount, order.User.Name)
	}

	fmt.Println("\n6. Conditional preloading...")

	// Preload orders but only active ones (requires GORM-specific syntax)
	var usersWithActiveOrders []User
	err = userRepo.Query(ctx, &usersWithActiveOrders,
		gpa.Where("status", gpa.OpEqual, "active"),
		// Note: Conditional preloading would need to be implemented in GORM adapter
		gpa.Preload("Orders"),
		gpa.Limit(3),
	)
	if err != nil {
		return fmt.Errorf("failed conditional preloading: %w", err)
	}

	fmt.Printf("Users with conditionally loaded orders: %d\n", len(usersWithActiveOrders))

	fmt.Println("\n7. Nested relationship queries...")

	// Create some categories and products for demonstration
	categoryRepo := app.provider.RepositoryFor(&Category{})
	productRepo := app.provider.RepositoryFor(&Product{})

	// Create a category
	category := &Category{
		Name: "Electronics",
	}
	err = categoryRepo.Create(ctx, category)
	if err != nil {
		log.Printf("Warning: Failed to create category: %v", err)
	} else {
		// Create products in the category
		products := []*Product{
			{Name: "Laptop", Price: 999.99, Stock: 10, CategoryID: category.ID, IsActive: true},
			{Name: "Mouse", Price: 29.99, Stock: 50, CategoryID: category.ID, IsActive: true},
		}

		for _, product := range products {
			err = productRepo.Create(ctx, product)
			if err != nil {
				log.Printf("Warning: Failed to create product %s: %v", product.Name, err)
			}
		}

		// Query products with category information
		var productsWithCategory []Product
		err = productRepo.Query(ctx, &productsWithCategory,
			gpa.Where("category_id", gpa.OpEqual, category.ID),
			gpa.Preload("Category"), // This would need the relationship defined in Product model
		)
		if err == nil {
			fmt.Printf("Products in category: %d\n", len(productsWithCategory))
		}
	}

	return nil
}

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

		// Create order
		if err := tx.Create(ctx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		return nil
	})
}

// GetUserStats returns user statistics using aggregation
func (s *UserService) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	sqlRepo, ok := s.userRepo.(gpa.SQLRepository)
	if !ok {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeUnsupported,
			Message: "SQL operations not supported",
		}
	}

	// Use a struct to properly scan the results
	var stats struct {
		TotalUsers    int64     `json:"total_users"`
		ActiveUsers   int64     `json:"active_users"`
		InactiveUsers int64     `json:"inactive_users"`
		AverageAge    float64   `json:"average_age"`
		FirstUserDate time.Time `json:"first_user_date"`
		LastUserDate  time.Time `json:"last_user_date"`
	}

	err := sqlRepo.FindBySQL(ctx,
		`SELECT
			COUNT(*) as total_users,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_users,
			COUNT(CASE WHEN status = 'inactive' THEN 1 END) as inactive_users,
			AVG(age) as average_age,
			MIN(created_at) as first_user_date,
			MAX(created_at) as last_user_date
		FROM users`,
		[]interface{}{},
		&stats,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Convert struct to map for easier handling
	result := map[string]interface{}{
		"total_users":     stats.TotalUsers,
		"active_users":    stats.ActiveUsers,
		"inactive_users":  stats.InactiveUsers,
		"average_age":     stats.AverageAge,
		"first_user_date": stats.FirstUserDate,
		"last_user_date":  stats.LastUserDate,
	}

	return result, nil
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
	fmt.Println("=== Running Basic CRUD Examples ===")
	if err := runBasicCRUDExamples(ctx, app); err != nil {
		log.Printf("Basic CRUD examples failed: %v", err)
	}

	// Advanced query examples
	fmt.Println("\n=== Running Advanced Query Examples ===")
	if err := runAdvancedQueryExamples(ctx, app); err != nil {
		log.Printf("Advanced query examples failed: %v", err)
	}

	// Transaction examples
	fmt.Println("\n=== Running Transaction Examples ===")
	if err := runTransactionExamples(ctx, app); err != nil {
		log.Printf("Transaction examples failed: %v", err)
	}

	// Schema management examples
	fmt.Println("\n=== Running Schema Management Examples ===")
	if err := runSchemaExamples(ctx, app); err != nil {
		log.Printf("Schema examples failed: %v", err)
	}

	// Relationship examples
	fmt.Println("\n=== Running Relationship Examples ===")
	if err := runRelationshipExamples(ctx, app); err != nil {
		log.Printf("Relationship examples failed: %v", err)
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
	// Database configuration
	config := gpa.Config{
		Driver:   "postgres", // or "mysql", "sqlite", "sqlserver"
		Host:     "localhost",
		Port:     5432,
		Database: "gpa_example",
		Username: "postgres",
		Password: "password",

		// Connection pool settings
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,

		// GORM-specific options
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level":      "info",
				"singular_table": false,
			},
		},

		// SSL configuration
		SSL: gpa.SSLConfig{
			Enabled: false, // Set to true for production
			Mode:    "disable",
		},
	}

	// For SQLite (simpler setup for examples)
	if true { // Change to false to use PostgreSQL
		config = gpa.Config{
			Driver:   "sqlite",
			Database: "example.db",
			Options: map[string]interface{}{
				"gorm": map[string]interface{}{
					"log_level": "info",
				},
			},
		}
	}

	// Create provider
	provider, err := gpa.NewProvider("gorm", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Test connection
	if err := provider.Health(); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	// Auto-migrate tables
	if err := setupDatabase(provider); err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	// Initialize services
	userService := NewUserService(provider)

	return &App{
		provider:    provider,
		userService: userService,
	}, nil
}

// Close closes the application and its resources
func (app *App) Close() error {
	return app.provider.Close()
}

// setupDatabase auto-migrates all tables
func setupDatabase(provider gpa.Provider) error {
	// Get SQL repository for schema operations
	sqlRepo := provider.RepositoryFor(&User{}).(gpa.SQLRepository)

	ctx := context.Background()

	// Auto-migrate all tables
	entities := []interface{}{
		&User{},
		&Order{},
		&Product{},
		&Category{},
	}

	for _, entity := range entities {
		if err := sqlRepo.MigrateTable(ctx, entity); err != nil {
			return fmt.Errorf("failed to migrate table for %T: %w", entity, err)
		}
	}

	// Create additional indexes (check if they exist first)
	indexConfigs := []struct {
		entity interface{}
		fields []string
		unique bool
		name   string
	}{
		{&User{}, []string{"email"}, true, "email index"},
		{&Order{}, []string{"user_id", "order_date"}, false, "order index"},
		{&Product{}, []string{"name"}, false, "product name index"},
		{&Product{}, []string{"category_id"}, false, "product category index"},
		{&Category{}, []string{"name"}, true, "category name index"},
	}

	for _, config := range indexConfigs {
		if err := sqlRepo.CreateIndex(ctx, config.entity, config.fields, config.unique); err != nil {
			// Only log warnings for duplicate indexes, not errors
			if gpaErr, ok := err.(gpa.GPAError); ok && gpaErr.Type == gpa.ErrorTypeDuplicate {
				log.Printf("Info: %s already exists", config.name)
			} else {
				log.Printf("Warning: Failed to create %s: %v", config.name, err)
			}
		} else {
			log.Printf("Created %s successfully", config.name)
		}
	}

	return nil
}

// =====================================
// Example Functions
// =====================================

// runBasicCRUDExamples demonstrates basic CRUD operations
func runBasicCRUDExamples(ctx context.Context, app *App) error {
	fmt.Println("1. Creating users...")

	// Create users
	users := []*User{
		{Name: "Alice Johnson", Email: "alice@example.com", Age: 28},
		{Name: "Bob Smith", Email: "bob@example.com", Age: 35},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 22},
	}

	for _, user := range users {
		if err := app.userService.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user %s: %w", user.Name, err)
		}
		fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	}

	// Read user
	fmt.Println("\n2. Reading user...")
	user, err := app.userService.GetUserByID(ctx, users[0].ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	fmt.Printf("Retrieved user: %+v\n", user)

	// Update user
	fmt.Println("\n3. Updating user...")
	if err := app.userService.UpdateUserStatus(ctx, user.ID, "premium"); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	fmt.Printf("Updated user %d status to premium\n", user.ID)

	// List users
	fmt.Println("\n4. Listing active users...")
	activeUsers, err := app.userService.GetActiveUsers(ctx, 10, 0)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}
	fmt.Printf("Found %d active users\n", len(activeUsers))
	for _, u := range activeUsers {
		fmt.Printf("  - %s (%s)\n", u.Name, u.Email)
	}

	return nil
}

// runAdvancedQueryExamples demonstrates advanced querying
func runAdvancedQueryExamples(ctx context.Context, app *App) error {
	userRepo := app.provider.RepositoryFor(&User{})

	fmt.Println("1. Complex condition queries...")

	// Complex query with multiple conditions
	var users []User
	err := userRepo.Query(ctx, &users,
		gpa.Where("age", gpa.OpGreaterThan, 25),
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrderBy("name", gpa.OrderAsc),
		gpa.Limit(5),
	)
	if err != nil {
		return fmt.Errorf("failed complex query: %w", err)
	}
	fmt.Printf("Found %d users over 25 and active\n", len(users))

	// Search with OR conditions - Method 1: Using OrOption
	fmt.Println("\n2. Search with OR conditions (Method 1)...")
	var searchUsers1 []User
	err = userRepo.Query(ctx, &searchUsers1,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%Alice%"),
			gpa.WhereCondition("email", gpa.OpLike, "%Alice%"),
		),
		gpa.Where("status", gpa.OpEqual, "active"),
	)
	if err != nil {
		return fmt.Errorf("failed OR search: %w", err)
	}
	fmt.Printf("OR search results: %d users\n", len(searchUsers1))

	// Search with complex AND/OR conditions - Method 2: Using AndOption with nested OrOption
	fmt.Println("\n3. Complex AND/OR conditions...")
	var searchUsers2 []User
	err = userRepo.Query(ctx, &searchUsers2,
		gpa.Where("age", gpa.OpGreaterThan, 20),
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%Alice%"),
			gpa.WhereCondition("name", gpa.OpLike, "%Bob%"),
		),
		gpa.OrderBy("name", gpa.OrderAsc),
	)
	if err != nil {
		return fmt.Errorf("failed complex AND/OR search: %w", err)
	}
	fmt.Printf("Complex AND/OR results: %d users\n", len(searchUsers2))

	// Count query
	fmt.Println("\n4. Count operations...")
	count, err := userRepo.Count(ctx,
		gpa.Where("age", gpa.OpBetween, []interface{}{20, 30}))
	if err != nil {
		return fmt.Errorf("failed count: %w", err)
	}
	fmt.Printf("Users between 20-30 years: %d\n", count)

	// Exists check
	exists, err := userRepo.Exists(ctx,
		gpa.Where("email", gpa.OpEqual, "alice@example.com"))
	if err != nil {
		return fmt.Errorf("failed exists check: %w", err)
	}
	fmt.Printf("Alice exists: %t\n", exists)

	return nil
}

// runTransactionExamples demonstrates transaction usage
func runTransactionExamples(ctx context.Context, app *App) error {
	fmt.Println("1. Creating user with first order in transaction...")

	user := &User{
		Name:  "David Wilson",
		Email: "david@example.com",
		Age:   30,
	}

	order := &Order{
		ProductName: "Laptop",
		Amount:      999.99,
		Status:      "pending",
	}

	if err := app.userService.CreateUserWithFirstOrder(ctx, user, order); err != nil {
		return fmt.Errorf("failed transaction: %w", err)
	}

	fmt.Printf("Created user %s with order %d in transaction\n", user.Name, order.ID)

	// Verify the transaction worked
	userWithOrders, err := app.userService.GetUserWithOrders(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to verify transaction: %w", err)
	}

	fmt.Printf("User %s has %d orders\n", userWithOrders.Name, len(userWithOrders.Orders))

	return nil
}

// runSchemaExamples demonstrates schema management
func runSchemaExamples(ctx context.Context, app *App) error {
	sqlRepo := app.provider.RepositoryFor(&Product{}).(gpa.SQLRepository)

	fmt.Println("1. Getting entity metadata...")

	// Get entity information
	entityInfo, err := sqlRepo.GetEntityInfo(&Product{})
	if err != nil {
		return fmt.Errorf("failed to get entity info: %w", err)
	}

	fmt.Printf("Product entity info:\n")
	fmt.Printf("  Name: %s\n", entityInfo.Name)
	fmt.Printf("  Table: %s\n", entityInfo.TableName)
	fmt.Printf("  Fields: %d\n", len(entityInfo.Fields))
	fmt.Printf("  Primary Keys: %v\n", entityInfo.PrimaryKey)

	// Show field details
	for _, field := range entityInfo.Fields {
		fmt.Printf("    %s (%s) - PK: %t, Nullable: %t\n",
			field.Name, field.DatabaseType, field.IsPrimaryKey, field.IsNullable)
	}

	fmt.Println("\n2. Raw SQL operations...")

	// Get user statistics
	stats, err := app.userService.GetUserStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("User statistics: %+v\n", stats)

	return nil
}

// =====================================
// Query Building Examples
// =====================================

// DemonstrateQueryBuilding shows various ways to build queries
func DemonstrateQueryBuilding(ctx context.Context, userRepo gpa.Repository) error {
	var users []User

	// Method 1: Simple conditions
	fmt.Println("=== Simple Conditions ===")
	err := userRepo.Query(ctx, &users,
		gpa.Where("age", gpa.OpGreaterThan, 25),
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrderBy("name", gpa.OrderAsc),
	)
	if err != nil {
		return err
	}

	// Method 2: OR conditions using OrOption
	fmt.Println("=== OR Conditions ===")
	err = userRepo.Query(ctx, &users,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%John%"),
			gpa.WhereCondition("email", gpa.OpLike, "%john%"),
		),
	)
	if err != nil {
		return err
	}

	// Method 3: Complex nested conditions
	fmt.Println("=== Complex Nested Conditions ===")
	// (name LIKE '%John%' OR email LIKE '%john%') AND age > 18 AND status = 'active'
	err = userRepo.Query(ctx, &users,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%John%"),
			gpa.WhereCondition("email", gpa.OpLike, "%john%"),
		),
		gpa.Where("age", gpa.OpGreaterThan, 18),
		gpa.Where("status", gpa.OpEqual, "active"),
	)
	if err != nil {
		return err
	}

	// Method 4: Using AndOption for explicit grouping
	fmt.Println("=== Explicit AND Grouping ===")
	err = userRepo.Query(ctx, &users,
		gpa.AndOption(
			gpa.WhereCondition("age", gpa.OpGreaterThan, 18),
			gpa.WhereCondition("age", gpa.OpLessThan, 65),
		),
		gpa.Where("status", gpa.OpEqual, "active"),
	)
	if err != nil {
		return err
	}

	// Method 5: Multiple OR groups with AND
	fmt.Println("=== Multiple OR Groups ===")
	// (name LIKE '%John%' OR name LIKE '%Jane%') AND (status = 'active' OR status = 'premium')
	err = userRepo.Query(ctx, &users,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%John%"),
			gpa.WhereCondition("name", gpa.OpLike, "%Jane%"),
		),
		gpa.OrOption(
			gpa.WhereCondition("status", gpa.OpEqual, "active"),
			gpa.WhereCondition("status", gpa.OpEqual, "premium"),
		),
	)
	if err != nil {
		return err
	}

	// Method 6: Range queries
	fmt.Println("=== Range Queries ===")
	err = userRepo.Query(ctx, &users,
		gpa.Where("age", gpa.OpBetween, []interface{}{25, 45}),
		gpa.Where("created_at", gpa.OpGreaterThan, time.Now().AddDate(-1, 0, 0)),
	)
	if err != nil {
		return err
	}

	// Method 7: IN queries
	fmt.Println("=== IN Queries ===")
	err = userRepo.Query(ctx, &users,
		gpa.Where("status", gpa.OpIn, []string{"active", "premium", "vip"}),
		gpa.Where("age", gpa.OpNotIn, []int{16, 17}), // Exclude minors
	)
	if err != nil {
		return err
	}

	// Method 8: NULL checks
	fmt.Println("=== NULL Checks ===")
	err = userRepo.Query(ctx, &users,
		gpa.Where("email", gpa.OpIsNotNull, nil),
		gpa.Where("deleted_at", gpa.OpIsNull, nil),
	)
	if err != nil {
		return err
	}

	// Method 9: Full query with all options
	fmt.Println("=== Complete Query ===")
	err = userRepo.Query(ctx, &users,
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%search%"),
			gpa.WhereCondition("email", gpa.OpLike, "%search%"),
		),
		gpa.Where("age", gpa.OpGreaterThan, 18),
		gpa.Select("id", "name", "email", "age"), // Only select specific fields
		gpa.OrderBy("created_at", gpa.OrderDesc),
		gpa.OrderBy("name", gpa.OrderAsc), // Secondary sort
		gpa.Limit(20),
		gpa.Offset(40), // Page 3 (20 per page)
	)
	if err != nil {
		return err
	}

	return nil
}

// ExampleWithMySQL shows how to configure MySQL
func ExampleWithMySQL() gpa.Config {
	return gpa.Config{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "gpa_example",
		Username: "root",
		Password: "password",

		MaxOpenConns:    20,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,

		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "warn",
			},
		},
	}
}

// ExampleWithPostgreSQL shows how to configure PostgreSQL with SSL
func ExampleWithPostgreSQL() gpa.Config {
	return gpa.Config{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "gpa_example",
		Username: "postgres",
		Password: "password",

		SSL: gpa.SSLConfig{
			Enabled:  true,
			Mode:     "require",
			CertFile: "/path/to/client-cert.pem",
			KeyFile:  "/path/to/client-key.pem",
			CAFile:   "/path/to/ca-cert.pem",
		},

		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level":      "error",
				"singular_table": true,
			},
		},
	}
}

// ExampleWithConnectionURL shows how to use connection URL
func ExampleWithConnectionURL() gpa.Config {
	return gpa.Config{
		Driver:        "postgres",
		ConnectionURL: "postgres://user:password@localhost:5432/dbname?sslmode=disable",

		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "info",
			},
		},
	}
}
