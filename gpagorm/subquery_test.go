package gpagorm

import (
	"context"
	"testing"
	"time"

	"github.com/lemmego/gpa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSubQueryWithGORM tests subquery functionality specifically with GORM
func TestSubQueryWithGORM(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create test tables
	err = db.AutoMigrate(&TestUser{}, &TestOrder{})
	require.NoError(t, err)

	// Create repository
	provider := &Provider{db: db}
	userRepo := provider.RepositoryFor(&TestUser{})
	orderRepo := provider.RepositoryFor(&TestOrder{})

	ctx := context.Background()

	// Create test data
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 30, Status: "active"},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35, Status: "inactive"},
	}

	for _, user := range users {
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)
	}

	// Create orders for Alice and Bob
	orders := []*TestOrder{
		{UserID: users[0].ID, ProductName: "Product A", Amount: 100.0, Status: "completed", OrderDate: time.Now()},
		{UserID: users[0].ID, ProductName: "Product B", Amount: 150.0, Status: "completed", OrderDate: time.Now()},
		{UserID: users[1].ID, ProductName: "Product C", Amount: 200.0, Status: "pending", OrderDate: time.Now()},
	}

	for _, order := range orders {
		err := orderRepo.Create(ctx, order)
		require.NoError(t, err)
	}

	// Test EXISTS subquery
	t.Run("EXISTS subquery", func(t *testing.T) {
		var usersWithOrders []TestUser
		err := userRepo.Query(ctx, &usersWithOrders,
			gpa.ExistsSubQuery("SELECT 1 FROM test_orders WHERE test_orders.user_id = test_users.id"),
		)
		assert.NoError(t, err)
		assert.Len(t, usersWithOrders, 2) // Alice and Bob have orders
		
		// Verify the correct users were found
		names := make([]string, len(usersWithOrders))
		for i, user := range usersWithOrders {
			names[i] = user.Name
		}
		assert.Contains(t, names, "Alice")
		assert.Contains(t, names, "Bob")
		assert.NotContains(t, names, "Charlie")
	})

	// Test NOT EXISTS subquery
	t.Run("NOT EXISTS subquery", func(t *testing.T) {
		var usersWithoutOrders []TestUser
		err := userRepo.Query(ctx, &usersWithoutOrders,
			gpa.NotExistsSubQuery("SELECT 1 FROM test_orders WHERE test_orders.user_id = test_users.id"),
		)
		assert.NoError(t, err)
		assert.Len(t, usersWithoutOrders, 1) // Only Charlie has no orders
		assert.Equal(t, "Charlie", usersWithoutOrders[0].Name)
	})

	// Test IN subquery
	t.Run("IN subquery", func(t *testing.T) {
		var ordersFromActiveUsers []TestOrder
		err := orderRepo.Query(ctx, &ordersFromActiveUsers,
			gpa.InSubQuery("user_id", "SELECT id FROM test_users WHERE status = ?", "active"),
		)
		assert.NoError(t, err)
		assert.Len(t, ordersFromActiveUsers, 3) // All 3 orders are from active users
	})

	// Test scalar subquery
	t.Run("Scalar subquery", func(t *testing.T) {
		var expensiveOrders []TestOrder
		err := orderRepo.Query(ctx, &expensiveOrders,
			gpa.WhereSubQuery("amount", gpa.OpGreaterThan, "SELECT AVG(amount) FROM test_orders"),
		)
		assert.NoError(t, err)
		// Average is (100+150+200)/3 = 150, so orders > 150 should be found
		assert.Len(t, expensiveOrders, 1) // Only the 200.0 order
		assert.Equal(t, 200.0, expensiveOrders[0].Amount)
	})

	// Test complex query with multiple subqueries
	t.Run("Complex query with multiple subqueries", func(t *testing.T) {
		var result []TestUser
		err := userRepo.Query(ctx, &result,
			gpa.Where("status", gpa.OpEqual, "active"),
			gpa.ExistsSubQuery("SELECT 1 FROM test_orders WHERE test_orders.user_id = test_users.id AND amount > ?", 120.0),
			gpa.OrderBy("name", gpa.OrderAsc),
		)
		assert.NoError(t, err)
		// Should find Alice (has 150.0 order) and Bob (has 200.0 order), but not Charlie
		assert.Len(t, result, 2)
		assert.Equal(t, "Alice", result[0].Name) // Ordered by name ASC
		assert.Equal(t, "Bob", result[1].Name)
	})
}

// TestSubQueryErrorHandling tests error cases with GORM subqueries
func TestSubQueryErrorHandling(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TestUser{})
	require.NoError(t, err)

	provider := &Provider{db: db}
	userRepo := provider.RepositoryFor(&TestUser{})
	ctx := context.Background()

	// Test invalid subquery (should not cause panic)
	t.Run("Invalid subquery", func(t *testing.T) {
		var users []TestUser
		err := userRepo.Query(ctx, &users,
			gpa.ExistsSubQuery("SELECT 1 FROM non_existent_table"),
		)
		// Should return an error, not panic
		assert.Error(t, err)
	})

	// Test subquery with syntax error
	t.Run("Syntax error in subquery", func(t *testing.T) {
		var users []TestUser
		err := userRepo.Query(ctx, &users,
			gpa.ExistsSubQuery("INVALID SQL SYNTAX"),
		)
		// Should return an error
		assert.Error(t, err)
	})
}

// TestSubQueryWithJoins tests subqueries combined with joins
func TestSubQueryWithJoins(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TestUser{}, &TestOrder{})
	require.NoError(t, err)

	provider := &Provider{db: db}
	userRepo := provider.RepositoryFor(&TestUser{})
	ctx := context.Background()

	// Create test data
	user := &TestUser{Name: "TestUser", Email: "test@example.com", Age: 30, Status: "active"}
	err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Test query that combines subqueries with other conditions
	t.Run("Subquery with WHERE conditions", func(t *testing.T) {
		var users []TestUser
		err := userRepo.Query(ctx, &users,
			gpa.Where("status", gpa.OpEqual, "active"),
			gpa.Where("age", gpa.OpGreaterThan, 25),
			gpa.NotExistsSubQuery("SELECT 1 FROM test_orders WHERE test_orders.user_id = test_users.id AND status = ?", "cancelled"),
		)
		assert.NoError(t, err)
		assert.Len(t, users, 1) // Should find the test user
	})
}