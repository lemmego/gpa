//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpabun"
	_ "github.com/lemmego/gpa/gpabun" // Register the Bun provider
)

// =====================================
// Models with Bun Tags
// =====================================

type User struct {
	ID        uint       `bun:"id,pk,autoincrement" json:"id"`
	Email     string     `bun:"email,type:varchar(255),unique,notnull" json:"email"`
	Name      string     `bun:"name,type:varchar(100),notnull" json:"name"`
	Age       int        `bun:"age,notnull" json:"age"`
	Status    string     `bun:"status,type:varchar(20),default:'active'" json:"status"`
	CreatedAt time.Time  `bun:"created_at,default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `bun:"updated_at,default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at,omitempty"`

	// Relationships
	Orders []Order `bun:"rel:has-many,join:id=user_id" json:"orders,omitempty"`
}

func (u User) TableName() string { return "users" }

type Order struct {
	ID          uint       `bun:"id,pk,autoincrement" json:"id"`
	UserID      uint       `bun:"user_id,notnull" json:"user_id"`
	ProductName string     `bun:"product_name,type:varchar(255),notnull" json:"product_name"`
	Amount      float64    `bun:"amount,type:real,notnull" json:"amount"`
	Status      string     `bun:"status,type:varchar(20),default:'pending'" json:"status"`
	OrderDate   time.Time  `bun:"order_date,notnull" json:"order_date"`
	CreatedAt   time.Time  `bun:"created_at,default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at,omitempty"`

	// Relationships
	User User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

func (o Order) TableName() string { return "orders" }

type Product struct {
	ID          uint                   `bun:"id,pk,autoincrement" json:"id"`
	Name        string                 `bun:"name,type:varchar(255),notnull" json:"name"`
	Description string                 `bun:"description,type:text" json:"description"`
	Price       float64                `bun:"price,type:real,notnull" json:"price"`
	Stock       int                    `bun:"stock,notnull,default:0" json:"stock"`
	IsActive    bool                   `bun:"is_active,default:true" json:"is_active"`
	Metadata    map[string]interface{} `bun:"metadata,type:text" json:"metadata,omitempty"`
	CreatedAt   time.Time              `bun:"created_at,default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time              `bun:"updated_at,default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   *time.Time             `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at,omitempty"`
}

func (p Product) TableName() string { return "products" }

// =====================================
// Service Layer
// =====================================

type UserService struct {
	provider gpa.Provider
	userRepo *gpabun.Repository
}

func NewUserService(provider gpa.Provider) *UserService {
	return &UserService{
		provider: provider,
		userRepo: provider.RepositoryFor(&User{}).(*gpabun.Repository),
	}
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	// Validation
	if user.Age < 18 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "user must be at least 18 years old",
		}
	}

	// Check if email exists
	exists, err := s.userRepo.ExistsWhere(ctx,
		gpa.Where("email", gpa.OpEqual, user.Email))
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: "email already exists",
		}
	}

	// Set timestamps
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	return s.userRepo.Create(ctx, user)
}

func (s *UserService) GetUserByID(ctx context.Context, id uint) (*User, error) {
	var user User
	err := s.userRepo.FindByID(ctx, id, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (s *UserService) GetUsersWithPagination(ctx context.Context, page, limit int) ([]User, int64, error) {
	var users []User
	totalCount, err := s.userRepo.FindWithPagination(ctx, &users, page, limit,
		gpa.Where("deleted_at", gpa.OpIsNull, nil),
		gpa.OrderBy("created_at", gpa.OrderDesc),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}
	return users, totalCount, nil
}

func (s *UserService) SearchUsers(ctx context.Context, query string) ([]User, error) {
	var users []User

	// Try full-text search first (PostgreSQL)
	if s.userRepo.IsPostgreSQL() {
		err := s.userRepo.FullTextSearch(ctx, &users, query,
			[]string{"name", "email"},
			gpa.OrOption(
				gpa.WhereCondition("status", gpa.OpEqual, "active"),
				gpa.WhereCondition("status", gpa.OpEqual, "verified"),
			),
			gpa.Where("deleted_at", gpa.OpIsNull, nil),
			gpa.Limit(50),
		)
		if err == nil {
			return users, nil
		}
	}

	// Fallback to LIKE search
	err := s.userRepo.Query(ctx, &users,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%"+query+"%"),
			gpa.WhereCondition("email", gpa.OpLike, "%"+query+"%"),
		),
		gpa.OrOption(
			gpa.WhereCondition("status", gpa.OpEqual, "active"),
			gpa.WhereCondition("status", gpa.OpEqual, "verified"),
		),
		gpa.Where("deleted_at", gpa.OpIsNull, nil),
		gpa.Limit(50),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	return users, nil
}

func (s *UserService) BulkCreateUsers(ctx context.Context, users []User) error {
	// Use bun's bulk insert with the slice directly
	return s.userRepo.BulkInsert(ctx, &users, 100)
}

func (s *UserService) UpdateUserStatus(ctx context.Context, userID uint, status string) error {
	return s.userRepo.UpdatePartial(ctx, userID, map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})
}

func (s *UserService) SoftDeleteUser(ctx context.Context, userID uint) error {
	return s.userRepo.SoftDelete(ctx, userID)
}

func (s *UserService) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalUsers    int64   `bun:"total_users"`
		ActiveUsers   int64   `bun:"active_users"`
		InactiveUsers int64   `bun:"inactive_users"`
		AverageAge    float64 `bun:"average_age"`
	}

	err := s.userRepo.GroupBy(ctx, &stats, []string{},
		map[string]string{
			"total_users":    "COUNT(*)",
			"active_users":   "COUNT(CASE WHEN status = 'active' THEN 1 END)",
			"inactive_users": "COUNT(CASE WHEN status = 'inactive' THEN 1 END)",
			"average_age":    "AVG(age)",
		},
		gpa.Where("deleted_at", gpa.OpIsNull, nil),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return map[string]interface{}{
		"total_users":    stats.TotalUsers,
		"active_users":   stats.ActiveUsers,
		"inactive_users": stats.InactiveUsers,
		"average_age":    stats.AverageAge,
	}, nil
}

func (s *UserService) GetUserWithOrders(ctx context.Context, userID uint) (*User, error) {
	var user User
	err := s.userRepo.FindByIDWithRelations(ctx, userID, &user, []string{"Orders"})
	if err != nil {
		return nil, fmt.Errorf("failed to get user with orders: %w", err)
	}
	return &user, nil
}

func (s *UserService) CreateUserWithOrder(ctx context.Context, user *User, order *Order) error {
	return s.userRepo.Transaction(ctx, func(tx gpa.Transaction) error {
		// Create user
		if err := tx.Create(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Create order with user ID
		order.UserID = user.ID
		order.OrderDate = time.Now()
		if err := tx.Create(ctx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		return nil
	})
}

// =====================================
// Event Hooks Example
// =====================================

type UserHooks struct{}

func (h *UserHooks) BeforeCreate(ctx context.Context, entity interface{}) error {
	if user, ok := entity.(*User); ok {
		if !strings.Contains(user.Email, "@") {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeValidation,
				Message: "invalid email format",
			}
		}
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		log.Printf("Creating user: %s", user.Email)
	}
	return nil
}

func (h *UserHooks) AfterCreate(ctx context.Context, entity interface{}) error {
	if user, ok := entity.(*User); ok {
		log.Printf("User created successfully: %s (ID: %d)", user.Name, user.ID)
	}
	return nil
}

func (h *UserHooks) BeforeUpdate(ctx context.Context, entity interface{}) error {
	if user, ok := entity.(*User); ok {
		user.UpdatedAt = time.Now()
	}
	return nil
}

func (h *UserHooks) AfterUpdate(ctx context.Context, entity interface{}) error {
	if user, ok := entity.(*User); ok {
		log.Printf("User updated: %s", user.Name)
	}
	return nil
}

func (h *UserHooks) BeforeDelete(ctx context.Context, entity interface{}) error {
	if user, ok := entity.(*User); ok {
		log.Printf("Deleting user: %s", user.Name)
	}
	return nil
}

func (h *UserHooks) AfterDelete(ctx context.Context, entity interface{}) error {
	if user, ok := entity.(*User); ok {
		log.Printf("User deleted: %s", user.Name)
	}
	return nil
}

// =====================================
// Main Application
// =====================================

func main() {
	// Initialize database
	provider, err := initializeDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Initialize services
	userService := NewUserService(provider)

	// Run examples
	if err := runExamples(ctx, userService, provider); err != nil {
		log.Fatalf("Examples failed: %v", err)
	}
}

func initializeDatabase() (gpa.Provider, error) {
	// Database configuration
	config := gpa.Config{
		Driver:   "sqlite",
		Database: "bun_example.db",
		Options: map[string]interface{}{
			"bun": map[string]interface{}{
				"log_level": "info",
			},
		},
	}

	// For PostgreSQL (uncomment to use)
	/*
		config = gpa.Config{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "bun_example",
			Username: "postgres",
			Password: "password",
			Options: map[string]interface{}{
				"bun": map[string]interface{}{
					"log_level": "info",
				},
			},
		}
	*/

	// Create provider
	provider, err := gpa.NewProvider("bun", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Test connection
	if err := provider.Health(); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	// Create tables
	if err := setupTables(provider); err != nil {
		return nil, fmt.Errorf("failed to setup tables: %w", err)
	}

	return provider, nil
}

func setupTables(provider gpa.Provider) error {
	ctx := context.Background()
	sqlRepo := provider.RepositoryFor(&User{}).(gpa.SQLRepository)

	// Create tables
	tables := []interface{}{&User{}, &Order{}, &Product{}}
	for _, table := range tables {
		if err := sqlRepo.CreateTable(ctx, table); err != nil {
			log.Printf("Table creation info: %v", err)
		}
	}

	// Create indexes
	indexes := []struct {
		entity interface{}
		fields []string
		unique bool
	}{
		{&User{}, []string{"email"}, true},
		{&Order{}, []string{"user_id"}, false},
		{&Product{}, []string{"name"}, false},
	}

	for _, idx := range indexes {
		if err := sqlRepo.CreateIndex(ctx, idx.entity, idx.fields, idx.unique); err != nil {
			log.Printf("Index creation info: %v", err)
		}
	}

	return nil
}

func runExamples(ctx context.Context, userService *UserService, provider gpa.Provider) error {
	fmt.Println("=== Bun Adapter Examples ===")

	// Example 1: Basic CRUD
	fmt.Println("\n1. Basic CRUD Operations")
	if err := basicCRUDExample(ctx, userService); err != nil {
		return fmt.Errorf("basic CRUD failed: %w", err)
	}

	// Example 2: Bulk operations
	fmt.Println("\n2. Bulk Operations")
	if err := bulkOperationsExample(ctx, userService); err != nil {
		return fmt.Errorf("bulk operations failed: %w", err)
	}

	// Example 3: Search
	fmt.Println("\n3. Search Operations")
	if err := searchExample(ctx, userService); err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Example 4: Pagination
	fmt.Println("\n4. Pagination")
	if err := paginationExample(ctx, userService); err != nil {
		return fmt.Errorf("pagination failed: %w", err)
	}

	// Example 5: Transactions
	fmt.Println("\n5. Transactions")
	if err := transactionExample(ctx, userService); err != nil {
		return fmt.Errorf("transactions failed: %w", err)
	}

	// Example 6: Relationships
	fmt.Println("\n6. Relationships")
	if err := relationshipExample(ctx, userService); err != nil {
		return fmt.Errorf("relationships failed: %w", err)
	}

	// Example 7: Advanced features
	fmt.Println("\n7. Advanced Features")
	if err := advancedFeaturesExample(ctx, userService); err != nil {
		return fmt.Errorf("advanced features failed: %w", err)
	}

	// Example 8: Event hooks
	fmt.Println("\n8. Event Hooks")
	if err := eventHooksExample(ctx, userService); err != nil {
		return fmt.Errorf("event hooks failed: %w", err)
	}

	fmt.Println("\n=== All Examples Completed Successfully! ===")
	return nil
}

func basicCRUDExample(ctx context.Context, userService *UserService) error {
	// Create user
	user := &User{
		Name:   "John Doe",
		Email:  "john@example.com",
		Age:    30,
		Status: "active",
	}

	if err := userService.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)

	// Read user
	foundUser, err := userService.GetUserByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	fmt.Printf("Found user: %s\n", foundUser.Name)

	// Update user
	if err := userService.UpdateUserStatus(ctx, user.ID, "premium"); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	fmt.Printf("Updated user status to premium\n")

	// Soft delete user
	if err := userService.SoftDeleteUser(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	fmt.Printf("Soft deleted user\n")

	return nil
}

func bulkOperationsExample(ctx context.Context, userService *UserService) error {
	// Create bulk users
	users := []User{
		{Name: "Alice Smith", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "active"},
		{Name: "Carol Williams", Email: "carol@example.com", Age: 28, Status: "inactive"},
	}

	if err := userService.BulkCreateUsers(ctx, users); err != nil {
		return fmt.Errorf("failed to bulk create users: %w", err)
	}
	fmt.Printf("Bulk created %d users\n", len(users))

	// Bulk update
	bunRepo := userService.userRepo
	affected, err := bunRepo.BulkUpdate(ctx,
		map[string]interface{}{"status": "verified"},
		gpa.Where("email", gpa.OpLike, "%@example.com"),
	)
	if err != nil {
		return fmt.Errorf("failed to bulk update: %w", err)
	}
	fmt.Printf("Bulk updated %d users\n", affected)

	return nil
}

func searchExample(ctx context.Context, userService *UserService) error {
	// Search users
	users, err := userService.SearchUsers(ctx, "Alice")
	if err != nil {
		return fmt.Errorf("failed to search users: %w", err)
	}
	fmt.Printf("Found %d users matching 'Alice'\n", len(users))

	for _, user := range users {
		fmt.Printf("  - %s (%s)\n", user.Name, user.Email)
	}

	return nil
}

func paginationExample(ctx context.Context, userService *UserService) error {
	// Get users with pagination
	users, totalCount, err := userService.GetUsersWithPagination(ctx, 1, 5)
	if err != nil {
		return fmt.Errorf("failed to get paginated users: %w", err)
	}

	fmt.Printf("Page 1: %d users (total: %d)\n", len(users), totalCount)
	for _, user := range users {
		fmt.Printf("  - %s (%s)\n", user.Name, user.Email)
	}

	return nil
}

func transactionExample(ctx context.Context, userService *UserService) error {
	// Create user and order in transaction
	user := &User{
		Name:   "Transaction User",
		Email:  "transaction@example.com",
		Age:    25,
		Status: "active",
	}

	order := &Order{
		ProductName: "Test Product",
		Amount:      99.99,
		Status:      "pending",
	}

	if err := userService.CreateUserWithOrder(ctx, user, order); err != nil {
		return fmt.Errorf("failed to create user with order: %w", err)
	}

	fmt.Printf("Created user %s with order %d in transaction\n", user.Name, order.ID)
	return nil
}

func relationshipExample(ctx context.Context, userService *UserService) error {
	// Find a user with existing orders
	users, _, err := userService.GetUsersWithPagination(ctx, 1, 1)
	if err != nil || len(users) == 0 {
		return fmt.Errorf("no users found for relationship example")
	}

	user := users[0]

	// Get user with orders
	userWithOrders, err := userService.GetUserWithOrders(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get user with orders: %w", err)
	}

	fmt.Printf("User %s has %d orders\n", userWithOrders.Name, len(userWithOrders.Orders))
	for _, order := range userWithOrders.Orders {
		fmt.Printf("  - Order #%d: %s ($%.2f)\n", order.ID, order.ProductName, order.Amount)
	}

	return nil
}

func advancedFeaturesExample(ctx context.Context, userService *UserService) error {
	bunRepo := userService.userRepo

	// Get user statistics
	stats, err := userService.GetUserStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user stats: %w", err)
	}

	fmt.Printf("User Statistics:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Health check
	health, err := bunRepo.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	fmt.Printf("\nHealth Status: %s\n", health.Status)

	// Connection stats
	connStats, err := bunRepo.GetConnectionStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection stats: %w", err)
	}

	fmt.Printf("Connection Stats:\n")
	for key, value := range connStats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	return nil
}

func eventHooksExample(ctx context.Context, userService *UserService) error {
	// Create repository with hooks
	hooks := &UserHooks{}
	hookedRepo := userService.userRepo.WithHooks(hooks)

	// Create user with hooks
	user := &User{
		Name:   "Hook User",
		Email:  "hook@example.com",
		Age:    30,
		Status: "active",
	}

	if err := hookedRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user with hooks: %w", err)
	}

	// Update user with hooks
	user.Status = "premium"
	if err := hookedRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user with hooks: %w", err)
	}

	fmt.Printf("User operations completed with hooks\n")
	return nil
}
