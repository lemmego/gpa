package gpagorm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lemmego/gpa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test models
type TestUser struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Age       int       `gorm:"not null" json:"age"`
	Status    string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Orders []TestOrder `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

type TestOrder struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	ProductName string    `gorm:"size:255;not null" json:"product_name"`
	Amount      float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status      string    `gorm:"size:20;default:'pending'" json:"status"`
	OrderDate   time.Time `gorm:"not null" json:"order_date"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	User TestUser `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type TestProduct struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null;index" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock       int       `gorm:"not null;default:0" json:"stock"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Test suite
type GormAdapterTestSuite struct {
	suite.Suite
	provider    gpa.Provider
	userRepo    gpa.Repository
	orderRepo   gpa.Repository
	productRepo gpa.Repository
	ctx         context.Context
}

func (suite *GormAdapterTestSuite) SetupSuite() {
	// Use SQLite for testing
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "silent",
			},
		},
	}

	provider, err := gpa.NewProvider("gorm", config)
	require.NoError(suite.T(), err)

	suite.provider = provider
	suite.userRepo = provider.RepositoryFor(&TestUser{})
	suite.orderRepo = provider.RepositoryFor(&TestOrder{})
	suite.productRepo = provider.RepositoryFor(&TestProduct{})
	suite.ctx = context.Background()

	// Auto-migrate tables
	migrator := suite.userRepo.(gpa.MigratableRepository)
	require.NoError(suite.T(), migrator.MigrateTable(suite.ctx, &TestUser{}))
	require.NoError(suite.T(), migrator.MigrateTable(suite.ctx, &TestOrder{}))
	require.NoError(suite.T(), migrator.MigrateTable(suite.ctx, &TestProduct{}))
}

func (suite *GormAdapterTestSuite) TearDownSuite() {
	if suite.provider != nil {
		suite.provider.Close()
	}
}

func (suite *GormAdapterTestSuite) SetupTest() {
	// Clean up tables before each test
	sqlRepo := suite.userRepo.(gpa.SQLRepository)

	// Delete data (ignore errors if tables don't exist)
	sqlRepo.ExecSQL(suite.ctx, "DELETE FROM test_orders")
	sqlRepo.ExecSQL(suite.ctx, "DELETE FROM test_users")
	sqlRepo.ExecSQL(suite.ctx, "DELETE FROM test_products")

	// Ensure tables exist by running migration again
	migrator := suite.userRepo.(gpa.MigratableRepository)
	migrator.MigrateTable(suite.ctx, &TestUser{})
	migrator.MigrateTable(suite.ctx, &TestOrder{})
	migrator.MigrateTable(suite.ctx, &TestProduct{})
}

// =====================================
// Provider Tests
// =====================================

func (suite *GormAdapterTestSuite) TestProviderFactory() {
	factory := &Factory{}

	// Test supported drivers
	drivers := factory.SupportedDrivers()
	expected := []string{"postgres", "postgresql", "mysql", "sqlite", "sqlite3", "sqlserver", "mssql"}
	assert.ElementsMatch(suite.T(), expected, drivers)

	// Test provider creation
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}
	provider, err := factory.Create(config)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), provider)
	defer provider.Close()

	// Test provider info
	info := provider.ProviderInfo()
	assert.Equal(suite.T(), "GORM", info.Name)
	assert.Equal(suite.T(), "1.0.0", info.Version)
	assert.Equal(suite.T(), gpa.DatabaseTypeSQL, info.DatabaseType)
	assert.Contains(suite.T(), info.Features, gpa.FeatureTransactions)
}

func (suite *GormAdapterTestSuite) TestProviderHealth() {
	err := suite.provider.Health()
	assert.NoError(suite.T(), err)
}

func (suite *GormAdapterTestSuite) TestProviderSupportedFeatures() {
	features := suite.provider.SupportedFeatures()
	expectedFeatures := []gpa.Feature{
		gpa.FeatureTransactions,
		gpa.FeatureJSONQueries,
		gpa.FeatureIndexing,
		gpa.FeatureAggregation,
	}
	assert.ElementsMatch(suite.T(), expectedFeatures, features)
}

// =====================================
// Basic CRUD Tests
// =====================================

func (suite *GormAdapterTestSuite) TestCreate() {
	user := &TestUser{
		Name:   "John Doe",
		Email:  "john@example.com",
		Age:    30,
		Status: "active",
	}

	err := suite.userRepo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), user.ID)
	assert.NotZero(suite.T(), user.CreatedAt)
	assert.NotZero(suite.T(), user.UpdatedAt)
}

func (suite *GormAdapterTestSuite) TestCreateBatch() {
	users := []*TestUser{
		{Name: "Alice Smith", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 28, Status: "active"},
	}

	err := suite.userRepo.CreateBatch(suite.ctx, users)
	assert.NoError(suite.T(), err)

	// Verify all users were created
	for _, user := range users {
		assert.NotZero(suite.T(), user.ID)
	}
}

func (suite *GormAdapterTestSuite) TestFindByID() {
	// Create a user first
	user := &TestUser{
		Name:   "Jane Doe",
		Email:  "jane@example.com",
		Age:    28,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Find by ID
	var foundUser TestUser
	err = suite.userRepo.FindByID(suite.ctx, user.ID, &foundUser)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, foundUser.ID)
	assert.Equal(suite.T(), user.Name, foundUser.Name)
	assert.Equal(suite.T(), user.Email, foundUser.Email)
}

func (suite *GormAdapterTestSuite) TestFindByIDNotFound() {
	var user TestUser
	err := suite.userRepo.FindByID(suite.ctx, 999, &user)
	assert.Error(suite.T(), err)

	gpaErr, ok := err.(gpa.GPAError)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), gpa.ErrorTypeNotFound, gpaErr.Type)
}

func (suite *GormAdapterTestSuite) TestUpdate() {
	// Create a user first
	user := &TestUser{
		Name:   "Update Test",
		Email:  "update@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Update the user
	user.Name = "Updated Name"
	user.Age = 31
	err = suite.userRepo.Update(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Verify the update
	var updatedUser TestUser
	err = suite.userRepo.FindByID(suite.ctx, user.ID, &updatedUser)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Name", updatedUser.Name)
	assert.Equal(suite.T(), 31, updatedUser.Age)
}

func (suite *GormAdapterTestSuite) TestUpdatePartial() {
	// Create a user first
	user := &TestUser{
		Name:   "Partial Update Test",
		Email:  "partial@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Update only specific fields
	updates := map[string]interface{}{
		"name": "Partially Updated",
		"age":  35,
	}
	err = suite.userRepo.UpdatePartial(suite.ctx, user.ID, updates)
	assert.NoError(suite.T(), err)

	// Verify the update
	var updatedUser TestUser
	err = suite.userRepo.FindByID(suite.ctx, user.ID, &updatedUser)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Partially Updated", updatedUser.Name)
	assert.Equal(suite.T(), 35, updatedUser.Age)
	assert.Equal(suite.T(), "partial@example.com", updatedUser.Email) // Should remain unchanged
}

func (suite *GormAdapterTestSuite) TestDelete() {
	// Create a user first
	user := &TestUser{
		Name:   "Delete Test",
		Email:  "delete@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Delete the user
	err = suite.userRepo.Delete(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)

	// Verify the user is deleted
	var deletedUser TestUser
	err = suite.userRepo.FindByID(suite.ctx, user.ID, &deletedUser)
	assert.Error(suite.T(), err)
	gpaErr, ok := err.(gpa.GPAError)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), gpa.ErrorTypeNotFound, gpaErr.Type)
}

// =====================================
// Query Tests
// =====================================

func (suite *GormAdapterTestSuite) TestFindAll() {
	// Create test data
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{Name: "Charlie", Email: "charlie@example.com", Age: 28, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Find all users
	var foundUsers []TestUser
	err = suite.userRepo.FindAll(suite.ctx, &foundUsers)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), foundUsers, 3)
}

func (suite *GormAdapterTestSuite) TestQueryWithConditions() {
	// Create test data
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{Name: "Charlie", Email: "charlie@example.com", Age: 28, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query with simple condition
	var activeUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &activeUsers,
		gpa.Where("status", gpa.OpEqual, "active"))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), activeUsers, 2)

	// Query with multiple conditions
	var youngActiveUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &youngActiveUsers,
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.Where("age", gpa.OpLessThan, 30))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), youngActiveUsers, 2)
}

func (suite *GormAdapterTestSuite) TestQueryWithOrConditions() {
	// Create test data
	users := []*TestUser{
		{Name: "Alice Smith", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{Name: "Alice Brown", Email: "alice.brown@example.com", Age: 28, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query with OR conditions
	var aliceUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &aliceUsers,
		gpa.OrOption(
			gpa.WhereCondition("name", gpa.OpLike, "%Alice%"),
			gpa.WhereCondition("email", gpa.OpLike, "%alice%"),
		))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), aliceUsers, 2)
}

func (suite *GormAdapterTestSuite) TestQueryWithComplexConditions() {
	// Create test data
	users := []*TestUser{
		{Name: "Alice Smith", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 28, Status: "active"},
		{Name: "David Wilson", Email: "david@example.com", Age: 22, Status: "pending"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Complex query: (status = 'active' OR status = 'pending') AND age > 23
	var filteredUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &filteredUsers,
		gpa.OrOption(
			gpa.WhereCondition("status", gpa.OpEqual, "active"),
			gpa.WhereCondition("status", gpa.OpEqual, "pending"),
		),
		gpa.Where("age", gpa.OpGreaterThan, 23))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), filteredUsers, 2) // Alice and Charlie
}

func (suite *GormAdapterTestSuite) TestQueryWithOrdering() {
	// Create test data
	users := []*TestUser{
		{Name: "Charlie", Email: "charlie@example.com", Age: 28, Status: "active"},
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query with ordering
	var orderedUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &orderedUsers,
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrderBy("name", gpa.OrderAsc))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), orderedUsers, 3)
	assert.Equal(suite.T(), "Alice", orderedUsers[0].Name)
	assert.Equal(suite.T(), "Bob", orderedUsers[1].Name)
	assert.Equal(suite.T(), "Charlie", orderedUsers[2].Name)
}

func (suite *GormAdapterTestSuite) TestQueryWithLimitOffset() {
	// Create test data
	users := make([]*TestUser, 10)
	for i := 0; i < 10; i++ {
		users[i] = &TestUser{
			Name:   fmt.Sprintf("User%d", i+1),
			Email:  fmt.Sprintf("user%d@example.com", i+1),
			Age:    20 + i,
			Status: "active",
		}
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query with limit and offset
	var paginatedUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &paginatedUsers,
		gpa.Where("status", gpa.OpEqual, "active"),
		gpa.OrderBy("id", gpa.OrderAsc),
		gpa.Limit(3),
		gpa.Offset(2))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), paginatedUsers, 3)
	assert.Equal(suite.T(), "User3", paginatedUsers[0].Name)
}

func (suite *GormAdapterTestSuite) TestQueryOne() {
	// Create test data
	user := &TestUser{
		Name:   "Single User",
		Email:  "single@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Query one
	var foundUser TestUser
	err = suite.userRepo.QueryOne(suite.ctx, &foundUser,
		gpa.Where("email", gpa.OpEqual, "single@example.com"))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, foundUser.ID)
	assert.Equal(suite.T(), user.Name, foundUser.Name)
}

func (suite *GormAdapterTestSuite) TestCount() {
	// Create test data
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{Name: "Charlie", Email: "charlie@example.com", Age: 28, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Count all users
	totalCount, err := suite.userRepo.Count(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), totalCount)

	// Count active users
	activeCount, err := suite.userRepo.Count(suite.ctx,
		gpa.Where("status", gpa.OpEqual, "active"))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), activeCount)
}

func (suite *GormAdapterTestSuite) TestExists() {
	// Create test data
	user := &TestUser{
		Name:   "Exists Test",
		Email:  "exists@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Test exists - should be true
	exists, err := suite.userRepo.Exists(suite.ctx,
		gpa.Where("email", gpa.OpEqual, "exists@example.com"))
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test exists - should be false
	exists, err = suite.userRepo.Exists(suite.ctx,
		gpa.Where("email", gpa.OpEqual, "nonexistent@example.com"))
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// =====================================
// Relationship Tests
// =====================================

func (suite *GormAdapterTestSuite) TestPreloadRelationships() {
	// Create user with orders
	user := &TestUser{
		Name:   "User with Orders",
		Email:  "userorders@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create orders for the user
	orders := []*TestOrder{
		{UserID: user.ID, ProductName: "Product 1", Amount: 100.0, Status: "pending", OrderDate: time.Now()},
		{UserID: user.ID, ProductName: "Product 2", Amount: 200.0, Status: "completed", OrderDate: time.Now()},
	}
	err = suite.orderRepo.CreateBatch(suite.ctx, orders)
	require.NoError(suite.T(), err)

	// Query user with preloaded orders
	var userWithOrders TestUser
	err = suite.userRepo.Query(suite.ctx, &userWithOrders,
		gpa.Where("id", gpa.OpEqual, user.ID),
		gpa.Preload("Orders"))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, userWithOrders.ID)
	assert.Len(suite.T(), userWithOrders.Orders, 2)
}

func (suite *GormAdapterTestSuite) TestFindByIDWithRelations() {
	// Create user with orders
	user := &TestUser{
		Name:   "User with Relations",
		Email:  "relations@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	order := &TestOrder{
		UserID:      user.ID,
		ProductName: "Test Product",
		Amount:      150.0,
		Status:      "pending",
		OrderDate:   time.Now(),
	}
	err = suite.orderRepo.Create(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Check if repository supports relationship methods
	if relRepo, ok := suite.userRepo.(*Repository); ok {
		var userWithOrders TestUser
		err = relRepo.FindByIDWithRelations(suite.ctx, user.ID, &userWithOrders, []string{"Orders"})
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.ID, userWithOrders.ID)
		assert.Len(suite.T(), userWithOrders.Orders, 1)
	}
}

func (suite *GormAdapterTestSuite) TestAssociationManager() {
	// Create user
	user := &TestUser{
		Name:   "Association Test",
		Email:  "association@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Check if repository supports association management
	if gormRepo, ok := suite.userRepo.(*Repository); ok {
		// Get association manager
		ordersAssoc := gormRepo.Association(suite.ctx, user, "Orders")

		// Count orders (should be 0)
		count, err := ordersAssoc.Count()
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(0), count)

		// Add an order
		newOrder := &TestOrder{
			ProductName: "Associated Product",
			Amount:      99.99,
			Status:      "pending",
			OrderDate:   time.Now(),
		}
		err = ordersAssoc.Append(newOrder)
		assert.NoError(suite.T(), err)
		assert.NotZero(suite.T(), newOrder.ID)

		// Count orders (should be 1)
		count, err = ordersAssoc.Count()
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), count)

		// Find orders
		var foundOrders []TestOrder
		err = ordersAssoc.Find(&foundOrders)
		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), foundOrders, 1)
		assert.Equal(suite.T(), "Associated Product", foundOrders[0].ProductName)
	}
}

// =====================================
// Transaction Tests
// =====================================

func (suite *GormAdapterTestSuite) TestTransaction() {
	// Test successful transaction
	var createdUserID uint
	err := suite.userRepo.Transaction(suite.ctx, func(tx gpa.Transaction) error {
		user := &TestUser{
			Name:   "Transaction User",
			Email:  "transaction@example.com",
			Age:    30,
			Status: "active",
		}
		if err := tx.Create(suite.ctx, user); err != nil {
			return err
		}
		createdUserID = user.ID

		order := &TestOrder{
			UserID:      user.ID,
			ProductName: "Transaction Product",
			Amount:      250.0,
			Status:      "pending",
			OrderDate:   time.Now(),
		}
		return tx.Create(suite.ctx, order)
	})

	assert.NoError(suite.T(), err)

	// Verify both user and order were created
	var user TestUser
	err = suite.userRepo.FindByID(suite.ctx, createdUserID, &user)
	assert.NoError(suite.T(), err)

	var orders []TestOrder
	err = suite.orderRepo.Query(suite.ctx, &orders,
		gpa.Where("user_id", gpa.OpEqual, createdUserID))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), orders, 1)
}

func (suite *GormAdapterTestSuite) TestTransactionRollback() {
	// Count users before transaction
	initialCount, err := suite.userRepo.Count(suite.ctx)
	require.NoError(suite.T(), err)

	// Test transaction rollback
	err = suite.userRepo.Transaction(suite.ctx, func(tx gpa.Transaction) error {
		user := &TestUser{
			Name:   "Rollback User",
			Email:  "rollback@example.com",
			Age:    30,
			Status: "active",
		}
		if err := tx.Create(suite.ctx, user); err != nil {
			return err
		}

		// Force rollback with an error
		return fmt.Errorf("intentional error for rollback")
	})

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "intentional error")

	// Verify no user was created
	finalCount, err := suite.userRepo.Count(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), initialCount, finalCount)
}

// =====================================
// SQL Repository Tests
// =====================================

func (suite *GormAdapterTestSuite) TestRawQuery() {
	// Create test data
	users := []*TestUser{
		{Name: "Raw Query User 1", Email: "raw1@example.com", Age: 25, Status: "active"},
		{Name: "Raw Query User 2", Email: "raw2@example.com", Age: 35, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Execute raw query
	sqlRepo := suite.userRepo.(gpa.SQLRepository)
	var results []struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	err = sqlRepo.FindBySQL(suite.ctx,
		"SELECT name, age FROM test_users WHERE status = ? ORDER BY age",
		[]interface{}{"active"},
		&results)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 2)
	assert.Equal(suite.T(), "Raw Query User 1", results[0].Name)
	assert.Equal(suite.T(), 25, results[0].Age)
}

func (suite *GormAdapterTestSuite) TestRawExec() {
	// Create test data
	user := &TestUser{
		Name:   "Raw Exec User",
		Email:  "rawexec@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Execute raw update
	sqlRepo := suite.userRepo.(gpa.SQLRepository)
	result, err := sqlRepo.ExecSQL(suite.ctx,
		"UPDATE test_users SET age = ? WHERE id = ?",
		35, user.ID)
	assert.NoError(suite.T(), err)

	rowsAffected, err := result.RowsAffected()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), rowsAffected)

	// Verify the update
	var updatedUser TestUser
	err = suite.userRepo.FindByID(suite.ctx, user.ID, &updatedUser)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 35, updatedUser.Age)
}

func (suite *GormAdapterTestSuite) TestGetEntityInfo() {
	sqlRepo := suite.userRepo.(gpa.SQLRepository)

	entityInfo, err := sqlRepo.GetEntityInfo(&TestUser{})
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "TestUser", entityInfo.Name)
	assert.Equal(suite.T(), "test_users", entityInfo.TableName)
	assert.True(suite.T(), len(entityInfo.Fields) > 0)
	assert.Contains(suite.T(), entityInfo.PrimaryKey, "ID")

	// Check some specific fields
	var idField, emailField *gpa.FieldInfo
	for i := range entityInfo.Fields {
		field := &entityInfo.Fields[i]
		switch field.Name {
		case "ID":
			idField = field
		case "Email":
			emailField = field
		}
	}

	assert.NotNil(suite.T(), idField)
	assert.True(suite.T(), idField.IsPrimaryKey)
	assert.True(suite.T(), idField.IsAutoIncrement)

	assert.NotNil(suite.T(), emailField)
	assert.False(suite.T(), emailField.IsPrimaryKey)
	assert.False(suite.T(), emailField.IsNullable)
}

func (suite *GormAdapterTestSuite) TestMigrateTable() {
	migrator := suite.productRepo.(gpa.MigratableRepository)

	// The table should already exist from SetupSuite, but let's test anyway
	err := migrator.MigrateTable(suite.ctx, &TestProduct{})
	assert.NoError(suite.T(), err)

	// Verify we can create products
	product := &TestProduct{
		Name:        "Migration Test Product",
		Description: "Test product for migration",
		Price:       99.99,
		Stock:       10,
		IsActive:    true,
	}
	err = suite.productRepo.Create(suite.ctx, product)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), product.ID)
}

// =====================================
// Error Handling Tests
// =====================================

func (suite *GormAdapterTestSuite) TestDuplicateKeyError() {
	// Create a user
	user1 := &TestUser{
		Name:   "Duplicate Test",
		Email:  "duplicate@example.com",
		Age:    30,
		Status: "active",
	}
	err := suite.userRepo.Create(suite.ctx, user1)
	require.NoError(suite.T(), err)

	// Try to create another user with the same email
	user2 := &TestUser{
		Name:   "Duplicate Test 2",
		Email:  "duplicate@example.com", // Same email
		Age:    25,
		Status: "active",
	}
	err = suite.userRepo.Create(suite.ctx, user2)
	assert.Error(suite.T(), err)

	gpaErr, ok := err.(gpa.GPAError)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), gpa.ErrorTypeDuplicate, gpaErr.Type)
}

func (suite *GormAdapterTestSuite) TestUpdateNonExistentRecord() {
	updates := map[string]interface{}{
		"name": "Non-existent User",
	}
	err := suite.userRepo.UpdatePartial(suite.ctx, 999, updates)
	assert.Error(suite.T(), err)

	gpaErr, ok := err.(gpa.GPAError)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), gpa.ErrorTypeNotFound, gpaErr.Type)
}

func (suite *GormAdapterTestSuite) TestDeleteNonExistentRecord() {
	err := suite.userRepo.Delete(suite.ctx, 999)
	assert.Error(suite.T(), err)

	gpaErr, ok := err.(gpa.GPAError)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), gpa.ErrorTypeNotFound, gpaErr.Type)
}

// =====================================
// Advanced Query Tests
// =====================================

func (suite *GormAdapterTestSuite) TestQueryWithBetween() {
	// Create test data with different ages
	users := []*TestUser{
		{Name: "Young User", Email: "young@example.com", Age: 20, Status: "active"},
		{Name: "Middle User 1", Email: "middle1@example.com", Age: 25, Status: "active"},
		{Name: "Middle User 2", Email: "middle2@example.com", Age: 30, Status: "active"},
		{Name: "Old User", Email: "old@example.com", Age: 40, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query users between ages 22 and 32
	var middleAgedUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &middleAgedUsers,
		gpa.Where("age", gpa.OpBetween, []interface{}{22, 32}),
		gpa.OrderBy("age", gpa.OrderAsc))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), middleAgedUsers, 2)
	assert.Equal(suite.T(), "Middle User 1", middleAgedUsers[0].Name)
	assert.Equal(suite.T(), "Middle User 2", middleAgedUsers[1].Name)
}

func (suite *GormAdapterTestSuite) TestQueryWithInOperator() {
	// Create test data
	users := []*TestUser{
		{Name: "Active User", Email: "active@example.com", Age: 25, Status: "active"},
		{Name: "Pending User", Email: "pending@example.com", Age: 30, Status: "pending"},
		{Name: "Inactive User", Email: "inactive@example.com", Age: 35, Status: "inactive"},
		{Name: "Premium User", Email: "premium@example.com", Age: 40, Status: "premium"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query users with specific statuses
	var filteredUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &filteredUsers,
		gpa.Where("status", gpa.OpIn, []string{"active", "premium"}),
		gpa.OrderBy("name", gpa.OrderAsc))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), filteredUsers, 2)
	assert.Equal(suite.T(), "Active User", filteredUsers[0].Name)
	assert.Equal(suite.T(), "Premium User", filteredUsers[1].Name)
}

func (suite *GormAdapterTestSuite) TestQueryWithLikeOperator() {
	// Create test data
	users := []*TestUser{
		{Name: "John Smith", Email: "john.smith@example.com", Age: 25, Status: "active"},
		{Name: "Jane Johnson", Email: "jane.johnson@example.com", Age: 30, Status: "active"},
		{Name: "Bob Wilson", Email: "bob.wilson@example.com", Age: 35, Status: "active"},
	}
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)

	// Query users with names containing "J"
	var jUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &jUsers,
		gpa.Where("name", gpa.OpLike, "%J%"),
		gpa.OrderBy("name", gpa.OrderAsc))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), jUsers, 2)
	assert.Equal(suite.T(), "Jane Johnson", jUsers[0].Name)
	assert.Equal(suite.T(), "John Smith", jUsers[1].Name)
}

// =====================================
// Performance and Edge Case Tests
// =====================================

func (suite *GormAdapterTestSuite) TestLargeDataSetOperations() {
	// Create a large number of users
	const numUsers = 100
	users := make([]*TestUser, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = &TestUser{
			Name:   fmt.Sprintf("User%03d", i+1),
			Email:  fmt.Sprintf("user%03d@example.com", i+1),
			Age:    20 + (i % 50),
			Status: []string{"active", "inactive", "pending"}[i%3],
		}
	}

	// Test batch creation
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	assert.NoError(suite.T(), err)

	// Test counting
	count, err := suite.userRepo.Count(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(numUsers), count)

	// Test pagination
	var page1Users []TestUser
	err = suite.userRepo.Query(suite.ctx, &page1Users,
		gpa.OrderBy("id", gpa.OrderAsc),
		gpa.Limit(10),
		gpa.Offset(0))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), page1Users, 10)

	var page2Users []TestUser
	err = suite.userRepo.Query(suite.ctx, &page2Users,
		gpa.OrderBy("id", gpa.OrderAsc),
		gpa.Limit(10),
		gpa.Offset(10))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), page2Users, 10)

	// Ensure different results
	assert.NotEqual(suite.T(), page1Users[0].ID, page2Users[0].ID)
}

func (suite *GormAdapterTestSuite) TestConcurrentOperations() {
	// For SQLite in-memory DB, concurrent operations are limited
	// Instead test sequential batch operations that simulate concurrency
	const numBatches = 5
	const usersPerBatch = 10

	// Create batches sequentially (SQLite limitation)
	totalCreated := 0
	for batchId := 0; batchId < numBatches; batchId++ {
		users := make([]*TestUser, usersPerBatch)
		for i := 0; i < usersPerBatch; i++ {
			users[i] = &TestUser{
				Name:   fmt.Sprintf("Batch%d-User%d", batchId, i),
				Email:  fmt.Sprintf("b%d-u%d@example.com", batchId, i),
				Age:    25,
				Status: "active",
			}
		}
		err := suite.userRepo.CreateBatch(suite.ctx, users)
		assert.NoError(suite.T(), err, "Batch %d failed", batchId)
		totalCreated += usersPerBatch
	}

	// Verify total count
	count, err := suite.userRepo.Count(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(totalCreated), count)
}

// =====================================
// Configuration Tests
// =====================================

func TestProviderWithDifferentConfigurations(t *testing.T) {
	factory := &Factory{}

	// Test with different log levels
	configs := []gpa.Config{
		{
			Driver:   "sqlite",
			Database: ":memory:",
			Options: map[string]interface{}{
				"gorm": map[string]interface{}{
					"log_level": "silent",
				},
			},
		},
		{
			Driver:   "sqlite",
			Database: ":memory:",
			Options: map[string]interface{}{
				"gorm": map[string]interface{}{
					"log_level":      "info",
					"singular_table": true,
				},
			},
		},
	}

	for i, config := range configs {
		provider, err := factory.Create(config)
		assert.NoError(t, err, "Config %d failed", i)
		assert.NotNil(t, provider, "Config %d returned nil provider", i)

		err = provider.Health()
		assert.NoError(t, err, "Health check failed for config %d", i)

		provider.Close()
	}
}

func TestProviderWithInvalidDriver(t *testing.T) {
	factory := &Factory{}
	config := gpa.Config{
		Driver:   "unsupported_driver",
		Database: "test.db",
	}

	provider, err := factory.Create(config)
	assert.Error(t, err)
	assert.Nil(t, provider)

	gpaErr, ok := err.(gpa.GPAError)
	assert.True(t, ok)
	assert.Equal(t, gpa.ErrorTypeUnsupported, gpaErr.Type)
}

// =====================================
// Run the test suite
// =====================================

func TestGormAdapterSuite(t *testing.T) {
	suite.Run(t, new(GormAdapterTestSuite))
}

// =====================================
// Benchmark Tests
// =====================================

func BenchmarkCreate(b *testing.B) {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "silent",
			},
		},
	}

	provider, err := gpa.NewProvider("gorm", config)
	require.NoError(b, err)
	defer provider.Close()

	repo := provider.RepositoryFor(&TestUser{})
	ctx := context.Background()

	migrator := repo.(gpa.MigratableRepository)
	err = migrator.MigrateTable(ctx, &TestUser{})
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := &TestUser{
				Name:   fmt.Sprintf("Benchmark User %d", i),
				Email:  fmt.Sprintf("bench%d@example.com", i),
				Age:    25,
				Status: "active",
			}
			repo.Create(ctx, user)
			i++
		}
	})
}

func BenchmarkQuery(b *testing.B) {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "silent",
			},
		},
	}

	provider, err := gpa.NewProvider("gorm", config)
	require.NoError(b, err)
	defer provider.Close()

	repo := provider.RepositoryFor(&TestUser{})
	ctx := context.Background()

	migrator := repo.(gpa.MigratableRepository)
	err = migrator.MigrateTable(ctx, &TestUser{})
	require.NoError(b, err)

	// Create test data
	users := make([]*TestUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = &TestUser{
			Name:   fmt.Sprintf("Query User %d", i),
			Email:  fmt.Sprintf("query%d@example.com", i),
			Age:    20 + (i % 50),
			Status: []string{"active", "inactive"}[i%2],
		}
	}
	repo.CreateBatch(ctx, users)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var results []TestUser
			repo.Query(ctx, &results,
				gpa.Where("status", gpa.OpEqual, "active"),
				gpa.Limit(10))
		}
	})
}
