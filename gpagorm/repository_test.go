package gpagorm

import (
	"context"
	"testing"

	"github.com/lemmego/gpa"
)

type TestUser struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"size:255"`
	Email string `gorm:"uniqueIndex;size:255"`
	Age   int
}

func setupTestProvider(t *testing.T) (*Provider, func()) {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Auto-migrate the test table
	err = provider.db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Fatalf("Failed to migrate test table: %v", err)
	}

	cleanup := func() {
		provider.Close()
	}

	return provider, cleanup
}

func TestRepositoryCreate(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

func TestRepositoryFindByID(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

func TestRepositoryFindAllWithOptions(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

	// Find with conditions
	found, err := repo.FindAll(ctx, 
		gpa.Where("age", gpa.OpGreaterThan, 25),
		gpa.OrderBy("age", gpa.OrderAsc),
		gpa.Limit(2),
	)
	if err != nil {
		t.Errorf("Failed to find users with options: %v", err)
	}

	if len(found) != 2 {
		t.Errorf("Expected 2 users, got %d", len(found))
	}

	if found[0].Age != 30 {
		t.Errorf("Expected first user age 30, got %d", found[0].Age)
	}
	if found[1].Age != 35 {
		t.Errorf("Expected second user age 35, got %d", found[1].Age)
	}
}

func TestRepositoryUpdate(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

func TestRepositoryDeleteByCondition(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

	// Delete users older than 25
	condition := gpa.BasicCondition{
		FieldName: "age",
		Op:        gpa.OpGreaterThan,
		Val:       25,
	}
	err := repo.DeleteByCondition(ctx, condition)
	if err != nil {
		t.Errorf("Failed to delete by condition: %v", err)
	}

	// Verify only Alice remains
	remaining, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Failed to find remaining users: %v", err)
	}

	if len(remaining) != 1 {
		t.Errorf("Expected 1 remaining user, got %d", len(remaining))
	}
	if remaining[0].Name != "Alice" {
		t.Errorf("Expected Alice to remain, got '%s'", remaining[0].Name)
	}
}

func TestRepositoryQuery(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

	// Query with complex conditions
	results, err := repo.Query(ctx,
		gpa.Where("age", gpa.OpGreaterThanOrEqual, 30),
		gpa.OrderBy("name", gpa.OrderAsc),
	)
	if err != nil {
		t.Errorf("Failed to query users: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if results[0].Name != "Bob" {
		t.Errorf("Expected first result to be Bob, got '%s'", results[0].Name)
	}
}

func TestRepositoryQueryOne(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Query one
	found, err := repo.QueryOne(ctx, gpa.Where("email", gpa.OpEqual, "john@example.com"))
	if err != nil {
		t.Errorf("Failed to query one user: %v", err)
	}

	if found.Name != user.Name {
		t.Errorf("Expected name '%s', got '%s'", user.Name, found.Name)
	}
}

func TestRepositoryCount(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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

	// Count with condition
	count, err = repo.Count(ctx, gpa.Where("age", gpa.OpGreaterThan, 25))
	if err != nil {
		t.Errorf("Failed to count users with condition: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestRepositoryExists(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
	ctx := context.Background()

	// Check non-existent user
	exists, err := repo.Exists(ctx, gpa.Where("email", gpa.OpEqual, "nonexistent@example.com"))
	if err != nil {
		t.Errorf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected user not to exist")
	}

	// Create a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err = repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Check existing user
	exists, err = repo.Exists(ctx, gpa.Where("email", gpa.OpEqual, "john@example.com"))
	if err != nil {
		t.Errorf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected user to exist")
	}
}

func TestRepositoryTransaction(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)
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
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)

	info, err := repo.GetEntityInfo()
	if err != nil {
		t.Errorf("Failed to get entity info: %v", err)
	}

	if info.Name != "TestUser" {
		t.Errorf("Expected entity name 'TestUser', got '%s'", info.Name)
	}
	if info.TableName != "test_users" {
		t.Errorf("Expected table name 'test_users', got '%s'", info.TableName)
	}
	if len(info.Fields) == 0 {
		t.Error("Expected fields to be populated")
	}

	// Check for ID field
	var idField *gpa.FieldInfo
	for i, field := range info.Fields {
		if field.Name == "ID" {
			idField = &info.Fields[i]
			break
		}
	}
	if idField == nil {
		t.Error("Expected ID field to be found")
	} else if !idField.IsPrimaryKey {
		t.Error("Expected ID field to be primary key")
	}
}

func TestRepositoryClose(t *testing.T) {
	provider, cleanup := setupTestProvider(t)
	defer cleanup()

	repo := NewRepository[TestUser](provider.db, provider)

	err := repo.Close()
	if err != nil {
		t.Errorf("Failed to close repository: %v", err)
	}
}

func TestConvertGormError(t *testing.T) {
	// Test nil error
	err := convertGormError(nil)
	if err != nil {
		t.Error("Expected nil error to remain nil")
	}

	// Test record not found
	err = convertGormError(gpa.NewError(gpa.ErrorTypeNotFound, "record not found"))
	if !gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
		t.Error("Expected not found error type")
	}

	// Test other errors
	originalErr := gpa.NewError(gpa.ErrorTypeDatabase, "database error")
	err = convertGormError(originalErr)
	if !gpa.IsErrorType(err, gpa.ErrorTypeDatabase) {
		t.Error("Expected database error type")
	}
}

func TestSQLResult(t *testing.T) {
	result := &SQLResult{
		rowsAffected: 5,
		lastInsertID: 123,
	}

	rows, err := result.RowsAffected()
	if err != nil {
		t.Errorf("Failed to get rows affected: %v", err)
	}
	if rows != 5 {
		t.Errorf("Expected 5 rows affected, got %d", rows)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Errorf("Failed to get last insert ID: %v", err)
	}
	if id != 123 {
		t.Errorf("Expected last insert ID 123, got %d", id)
	}
}