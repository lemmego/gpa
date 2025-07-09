package gpa

import "context"

// =====================================
// Core Repository Interfaces
// =====================================

// Repository is the main interface for database operations.
// Provides compile-time type safety for all CRUD operations.
// This is the PRIMARY interface - use this for all new code.
type Repository[T any] interface {
	// ===============================
	// Basic CRUD Operations
	// ===============================
	
	// Create inserts a new entity into the database.
	// Returns error if the entity already exists or validation fails.
	// Example: err := Create(ctx, &user)
	Create(ctx context.Context, entity *T) error
	
	// CreateBatch inserts multiple entities in a single operation for better performance.
	// May not be atomic depending on the database implementation.
	// Example: err := CreateBatch(ctx, []*User{user1, user2, user3})
	CreateBatch(ctx context.Context, entities []*T) error
	
	// FindByID retrieves a single entity by its primary key.
	// Returns the entity directly with compile-time type safety.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	// Example: user, err := FindByID(ctx, 123)
	FindByID(ctx context.Context, id interface{}) (*T, error)
	
	// FindAll retrieves all entities of type T, optionally filtered by query options.
	// Returns a slice of entity pointers with compile-time type safety.
	// Use QueryOptions like Where(), Limit(), OrderBy() to filter and sort results.
	// Example: users, err := FindAll(ctx, Where("active", "=", true), Limit(10))
	FindAll(ctx context.Context, opts ...QueryOption) ([]*T, error)
	
	// Update modifies an existing entity. The entity must have an ID field.
	// Replaces the entire entity with the new values.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	// Example: err := Update(ctx, &user)
	Update(ctx context.Context, entity *T) error
	
	// UpdatePartial modifies specific fields of an entity without replacing the whole entity.
	// The updates map contains field names as keys and new values as values.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	// Example: err := UpdatePartial(ctx, 123, map[string]interface{}{"status": "inactive"})
	UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error
	
	// Delete removes an entity by its primary key.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	// Example: err := Delete(ctx, 123)
	Delete(ctx context.Context, id interface{}) error
	
	// DeleteByCondition removes all entities matching the given condition.
	// Use Where() conditions to specify which entities to delete.
	// Example: err := DeleteByCondition(ctx, Where("status", "=", "inactive"))
	DeleteByCondition(ctx context.Context, condition Condition) error

	// ===============================
	// Query Operations
	// ===============================
	
	// Query retrieves entities based on the provided query options.
	// More flexible than FindAll, supports complex conditions, joins, subqueries.
	// Returns a slice of entity pointers with compile-time type safety.
	// Example: users, err := Query(ctx, Where("age", ">", 18), OrderBy("name", "ASC"))
	Query(ctx context.Context, opts ...QueryOption) ([]*T, error)
	
	// QueryOne retrieves a single entity based on query options.
	// Equivalent to Query() with Limit(1) but returns ErrorTypeNotFound if no match.
	// Returns the entity directly with compile-time type safety.
	// Example: user, err := QueryOne(ctx, Where("email", "=", "user@example.com"))
	QueryOne(ctx context.Context, opts ...QueryOption) (*T, error)
	
	// Count returns the number of entities matching the query options.
	// Does not retrieve the actual entities, just counts them.
	// Useful for pagination and analytics.
	// Example: count, err := Count(ctx, Where("active", "=", true))
	Count(ctx context.Context, opts ...QueryOption) (int64, error)
	
	// Exists checks if any entities match the query options.
	// More efficient than Count() when you only need to know if matches exist.
	// Returns true if at least one entity matches, false otherwise.
	// Example: exists, err := Exists(ctx, Where("email", "=", "user@example.com"))
	Exists(ctx context.Context, opts ...QueryOption) (bool, error)

	// ===============================
	// Advanced Operations
	// ===============================
	
	// Transaction executes a function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// If the function completes successfully, the transaction is committed.
	// Not all databases support transactions (e.g., some NoSQL databases).
	// Example: err := Transaction(ctx, func(tx Transaction[T]) error { return tx.Create(ctx, &user) })
	Transaction(ctx context.Context, fn TransactionFunc[T]) error
	
	// RawQuery executes a database-specific query and returns results.
	// Returns a slice of entity pointers with compile-time type safety.
	// The query string and args format depend on the database type.
	// Example: users, err := RawQuery(ctx, "SELECT * FROM users WHERE age > ?", []interface{}{18})
	RawQuery(ctx context.Context, query string, args []interface{}) ([]*T, error)
	
	// RawExec executes a database-specific command that doesn't return data.
	// Used for database-specific operations like creating indexes, triggers, etc.
	// Returns a Result object with information about rows affected, etc.
	// Example: result, err := RawExec(ctx, "UPDATE users SET last_login = NOW()", nil)
	RawExec(ctx context.Context, query string, args []interface{}) (Result, error)

	// ===============================
	// Metadata Operations
	// ===============================
	
	// GetEntityInfo returns metadata about the entity type T.
	// Includes field information, primary keys, indexes, and relationships.
	// Useful for reflection, validation, and building dynamic UIs.
	// Example: info, err := GetEntityInfo()
	GetEntityInfo() (*EntityInfo, error)
	
	// Close closes the repository and releases any resources.
	// Should be called when the repository is no longer needed.
	// May close database connections, file handles, etc.
	// Example: err := Close()
	Close() error
}

// TransactionFunc represents a function that runs within a transaction
type TransactionFunc[T any] func(tx Transaction[T]) error

// Transaction interface for transactional operations.
// This is the PRIMARY transaction interface - use this for all new code.
type Transaction[T any] interface {
	Repository[T]
	
	// Commit permanently saves all changes made within this transaction.
	// Once committed, changes cannot be rolled back.
	// Returns error if the commit fails (e.g., constraint violations).
	Commit() error
	
	// Rollback discards all changes made within this transaction.
	// Can be called manually or automatically when the transaction function returns an error.
	// Returns error if the rollback fails.
	Rollback() error
	
	// SetSavepoint creates a savepoint within the transaction.
	// Allows partial rollback to a specific point without rolling back the entire transaction.
	// Not supported by all databases.
	SetSavepoint(name string) error
	
	// RollbackToSavepoint rolls back to a previously created savepoint.
	// Only changes made after the savepoint are discarded.
	// Not supported by all databases.
	RollbackToSavepoint(name string) error
}

// Result represents the result of a database operation
type Result interface {
	// LastInsertId returns the integer generated by the database
	// in response to a command. Typically this will be from an
	// "auto increment" column when inserting a new row.
	LastInsertId() (int64, error)
	
	// RowsAffected returns the number of rows affected by an
	// update, insert, or delete. Not every database or database
	// driver may support this.
	RowsAffected() (int64, error)
}