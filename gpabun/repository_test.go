package gpabun

import (
	"context"
	"database/sql"
	"testing"

	"github.com/lemmego/gpa"
)

type TestUser struct {
	ID    int64  `bun:",pk,autoincrement"`
	Name  string `bun:"name"`
	Email string `bun:"email"`
	Age   int    `bun:"age"`
}

func setupTestRepository(t *testing.T) (*Repository[TestUser], func()) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Create test table
	_, err = provider.db.NewCreateTable().Model((*TestUser)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	repo := &Repository[TestUser]{
		db:       provider.db,
		provider: provider,
	}

	cleanup := func() {
		provider.Close()
	}

	return repo, cleanup
}

func TestRepositoryCreate(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	user := &TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}
}

func TestRepositoryCreateBatch(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	users := []*TestUser{
		{Name: "User 1", Email: "user1@example.com", Age: 25},
		{Name: "User 2", Email: "user2@example.com", Age: 30},
		{Name: "User 3", Email: "user3@example.com", Age: 35},
	}

	err := repo.CreateBatch(ctx, users)
	if err != nil {
		t.Errorf("Failed to create batch: %v", err)
	}

	// Verify all users have IDs
	for i, user := range users {
		if user.ID == 0 {
			t.Errorf("Expected user %d to have ID set", i)
		}
	}
}

func TestRepositoryCreateBatchEmpty(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	var users []*TestUser

	err := repo.CreateBatch(ctx, users)
	if err != nil {
		t.Errorf("Failed to handle empty batch: %v", err)
	}
}

func TestRepositoryFindByID(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user first
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Find by ID
	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Errorf("Failed to find user by ID: %v", err)
	}

	if found.Name != user.Name {
		t.Errorf("Expected name '%s', got '%s'", user.Name, found.Name)
	}
	if found.Email != user.Email {
		t.Errorf("Expected email '%s', got '%s'", user.Email, found.Email)
	}
}

func TestRepositoryFindByIDNotFound(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	_, err := repo.FindByID(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}

	if !gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestRepositoryFindAll(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25},
		{Name: "Bob", Email: "bob@example.com", Age: 30},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Find all users
	found, err := repo.FindAll(ctx)
	if err != nil {
		t.Errorf("Failed to find all users: %v", err)
	}

	if len(found) != 3 {
		t.Errorf("Expected 3 users, got %d", len(found))
	}
}

func TestRepositoryUpdate(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update the user
	user.Name = "John Smith"
	user.Age = 31
	err = repo.Update(ctx, user)
	if err != nil {
		t.Errorf("Failed to update user: %v", err)
	}

	// Verify the update
	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to find updated user: %v", err)
	}

	if found.Name != "John Smith" {
		t.Errorf("Expected name 'John Smith', got '%s'", found.Name)
	}
	if found.Age != 31 {
		t.Errorf("Expected age 31, got %d", found.Age)
	}
}

func TestRepositoryUpdatePartial(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Partial update
	updates := map[string]interface{}{
		"age": 31,
	}
	err = repo.UpdatePartial(ctx, user.ID, updates)
	if err != nil {
		t.Errorf("Failed to update user partially: %v", err)
	}

	// Verify the update
	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to find updated user: %v", err)
	}

	if found.Age != 31 {
		t.Errorf("Expected age 31, got %d", found.Age)
	}
	if found.Name != "John Doe" {
		t.Errorf("Expected name unchanged, got '%s'", found.Name)
	}
}

func TestRepositoryDelete(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Delete the user
	err = repo.Delete(ctx, user.ID)
	if err != nil {
		t.Errorf("Failed to delete user: %v", err)
	}

	// Verify deletion
	_, err = repo.FindByID(ctx, user.ID)
	if err == nil {
		t.Error("Expected error when finding deleted user")
	}
}

func TestRepositoryQuery(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25},
		{Name: "Bob", Email: "bob@example.com", Age: 30},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Query (currently same as FindAll)
	results, err := repo.Query(ctx)
	if err != nil {
		t.Errorf("Failed to query users: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestRepositoryQueryOne(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Query one
	found, err := repo.QueryOne(ctx)
	if err != nil {
		t.Errorf("Failed to query one user: %v", err)
	}

	if found.Name != user.Name {
		t.Errorf("Expected name '%s', got '%s'", user.Name, found.Name)
	}
}

func TestRepositoryQueryOneNotFound(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Query one with no data
	_, err := repo.QueryOne(ctx)
	if err == nil {
		t.Error("Expected error for empty result set")
	}

	if !gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestRepositoryCount(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25},
		{Name: "Bob", Email: "bob@example.com", Age: 30},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Count all users
	count, err := repo.Count(ctx)
	if err != nil {
		t.Errorf("Failed to count users: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestRepositoryExists(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Check existence with no data
	exists, err := repo.Exists(ctx)
	if err != nil {
		t.Errorf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected no users to exist")
	}

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err = repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Check existence with data
	exists, err = repo.Exists(ctx)
	if err != nil {
		t.Errorf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected users to exist")
	}
}

func TestRepositoryTransaction(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Successful transaction
	err := repo.Transaction(ctx, func(tx gpa.Transaction[TestUser]) error {
		user1 := &TestUser{Name: "User 1", Email: "user1@example.com", Age: 25}
		user2 := &TestUser{Name: "User 2", Email: "user2@example.com", Age: 30}

		if err := tx.Create(ctx, user1); err != nil {
			return err
		}
		if err := tx.Create(ctx, user2); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Errorf("Transaction failed: %v", err)
	}

	// Verify both users were created
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 users after transaction, got %d", count)
	}
}

func TestRepositoryTransactionRollback(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial user
	initialUser := &TestUser{Name: "Initial", Email: "initial@example.com", Age: 20}
	err := repo.Create(ctx, initialUser)
	if err != nil {
		t.Fatalf("Failed to create initial user: %v", err)
	}

	// Failed transaction (should rollback)
	err = repo.Transaction(ctx, func(tx gpa.Transaction[TestUser]) error {
		user1 := &TestUser{Name: "User 1", Email: "user1@example.com", Age: 25}
		if err := tx.Create(ctx, user1); err != nil {
			return err
		}

		// This should cause a rollback
		return gpa.NewError(gpa.ErrorTypeValidation, "test error")
	})

	if err == nil {
		t.Error("Expected transaction to fail")
	}

	// Verify only initial user exists (transaction was rolled back)
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 user after failed transaction, got %d", count)
	}
}

func TestRepositoryRawQuery(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Raw query
	results, err := repo.RawQuery(ctx, "SELECT * FROM test_users WHERE age > ?", []interface{}{25})
	if err != nil {
		t.Errorf("Failed to execute raw query: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", results[0].Name)
	}
}

func TestRepositoryRawExec(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Raw exec
	result, err := repo.RawExec(ctx, "UPDATE test_users SET age = ? WHERE id = ?", []interface{}{35, user.ID})
	if err != nil {
		t.Errorf("Failed to execute raw exec: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		t.Errorf("Failed to get rows affected: %v", err)
	}
	if rows != 1 {
		t.Errorf("Expected 1 row affected, got %d", rows)
	}

	// Verify the update
	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to find updated user: %v", err)
	}
	if found.Age != 35 {
		t.Errorf("Expected age 35, got %d", found.Age)
	}
}

func TestRepositoryGetEntityInfo(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	info, err := repo.GetEntityInfo()
	if err != nil {
		t.Errorf("Failed to get entity info: %v", err)
	}

	if info.Name != "TestUser" {
		t.Errorf("Expected entity name 'TestUser', got '%s'", info.Name)
	}
	if info.TableName != "TestUser" {
		t.Errorf("Expected table name 'TestUser', got '%s'", info.TableName)
	}
}

func TestRepositoryClose(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	err := repo.Close()
	if err != nil {
		t.Errorf("Failed to close repository: %v", err)
	}
}

func TestTransactionMethods(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	// Create a transaction
	tx := &Transaction[TestUser]{
		Repository: repo,
	}

	// Test transaction methods (these are no-ops in Bun)
	err := tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}

	err = tx.Rollback()
	if err != nil {
		t.Errorf("Failed to rollback transaction: %v", err)
	}

	err = tx.SetSavepoint("test")
	if err != nil {
		t.Errorf("Failed to set savepoint: %v", err)
	}

	err = tx.RollbackToSavepoint("test")
	if err != nil {
		t.Errorf("Failed to rollback to savepoint: %v", err)
	}
}

func TestResult(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user to get a result
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Execute a raw command to get a result
	result, err := repo.RawExec(ctx, "UPDATE test_users SET age = ? WHERE id = ?", []interface{}{35, user.ID})
	if err != nil {
		t.Fatalf("Failed to execute raw command: %v", err)
	}

	// Test result methods
	rows, err := result.RowsAffected()
	if err != nil {
		t.Errorf("Failed to get rows affected: %v", err)
	}
	if rows != 1 {
		t.Errorf("Expected 1 row affected, got %d", rows)
	}

	// LastInsertId might not be available for UPDATE
	_, err = result.LastInsertId()
	if err != nil {
		t.Logf("LastInsertId not available (expected for UPDATE): %v", err)
	}
}

func TestConvertBunError(t *testing.T) {
	// Test nil error
	err := convertBunError(nil)
	if err != nil {
		t.Error("Expected nil error to remain nil")
	}

	// Test ErrNoRows
	err = convertBunError(sql.ErrNoRows)
	if !gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
		t.Error("Expected not found error type for ErrNoRows")
	}

	// Test other errors
	originalErr := gpa.NewError(gpa.ErrorTypeDatabase, "database error")
	err = convertBunError(originalErr)
	if !gpa.IsErrorType(err, gpa.ErrorTypeConnection) {
		t.Error("Expected connection error type for generic errors")
	}
}

func TestDeleteByCondition(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []*TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25},
		{Name: "Bob", Email: "bob@example.com", Age: 30},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Create a mock condition
	condition := &mockCondition{
		field: "age",
		value: 30,
	}

	// Delete users with age > 30 (this is a simplified test)
	err := repo.DeleteByCondition(ctx, condition)
	if err != nil {
		t.Errorf("Failed to delete by condition: %v", err)
	}
}

// Mock condition for testing
type mockCondition struct {
	field string
	value interface{}
}

func (c *mockCondition) String() string {
	return c.field + " > ?"
}

func (c *mockCondition) Value() interface{} {
	return c.value
}

func (c *mockCondition) Field() string {
	return c.field
}

func (c *mockCondition) Operator() gpa.Operator {
	return gpa.OpGreaterThan
}

func TestRepositoryWithBunQuery(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test direct Bun query usage
	var users []*TestUser
	err = repo.db.NewSelect().Model(&users).Where("age > ?", 25).Scan(ctx)
	if err != nil {
		t.Errorf("Failed to execute Bun query: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}