package gpa

import "context"

// =====================================
// SQL-Specific Repository Interface
// =====================================

// SQLRepository extends Repository with SQL-specific operations.
// Provides additional functionality for SQL databases including raw SQL,
// relationships, and schema management.
type SQLRepository[T any] interface {
	Repository[T]

	// ===============================
	// Raw SQL Operations
	// ===============================
	
	// FindBySQL executes a raw SQL SELECT query and returns typed results.
	// Returns a slice of entity pointers with compile-time type safety.
	// Example: users, err := FindBySQL(ctx, "SELECT * FROM users WHERE age > ?", []interface{}{18})
	FindBySQL(ctx context.Context, sql string, args []interface{}) ([]*T, error)
	
	// ExecSQL executes a raw SQL command that doesn't return entities (INSERT, UPDATE, DELETE, DDL).
	// Returns a Result with information about rows affected, last insert ID, etc.
	// Example: result, err := ExecSQL(ctx, "UPDATE users SET status = ? WHERE active = ?", "inactive", false)
	ExecSQL(ctx context.Context, sql string, args ...interface{}) (Result, error)

	// ===============================
	// Relationship Operations
	// ===============================
	
	// FindWithRelations retrieves entities with their related entities preloaded.
	// Returns a slice of entity pointers with compile-time type safety.
	// The relations slice specifies which relationships to load.
	// Example: users, err := FindWithRelations(ctx, []string{"Posts", "Profile"}, Where("active", "=", true))
	FindWithRelations(ctx context.Context, relations []string, opts ...QueryOption) ([]*T, error)
	
	// FindByIDWithRelations retrieves a single entity by ID with relationships preloaded.
	// Returns the entity directly with compile-time type safety.
	// Example: user, err := FindByIDWithRelations(ctx, userID, []string{"Posts", "Comments"})
	FindByIDWithRelations(ctx context.Context, id interface{}, relations []string) (*T, error)

	// ===============================
	// Schema Management
	// ===============================
	
	// CreateTable creates a new table based on the entity structure.
	// Analyzes the entity type T's fields, tags, and relationships to generate appropriate SQL.
	// May create foreign key constraints, indexes, and other database objects.
	// Example: err := CreateTable(ctx)
	CreateTable(ctx context.Context) error
	
	// DropTable removes the table for entity type T from the database.
	// WARNING: This permanently deletes all data in the table.
	// May fail if there are foreign key constraints pointing to this table.
	// Example: err := DropTable(ctx)
	DropTable(ctx context.Context) error

	// ===============================
	// Index Management
	// ===============================
	
	// CreateIndex creates a database index on the specified fields of entity type T.
	// Improves query performance for the specified field combinations.
	// The unique parameter determines if the index should enforce uniqueness.
	// Example: err := CreateIndex(ctx, []string{"email", "status"}, true)
	CreateIndex(ctx context.Context, fields []string, unique bool) error
	
	// DropIndex removes a database index.
	// The indexName should match an existing index on the table for entity type T.
	// Example: err := DropIndex(ctx, "idx_users_email_status")
	DropIndex(ctx context.Context, indexName string) error
}

// =====================================
// Migration Support
// =====================================

// MigratableRepository extends SQLRepository with migration capabilities.
// Provides schema migration and evolution functionality.
type MigratableRepository[T any] interface {
	SQLRepository[T]

	// ===============================
	// Migration Operations
	// ===============================
	
	// MigrateTable performs automatic schema migration for entity type T.
	// Analyzes the current entity structure and updates the database schema accordingly.
	// Can add new columns, indexes, and constraints but typically won't remove existing ones.
	// Example: err := MigrateTable(ctx)
	MigrateTable(ctx context.Context) error
	
	// GetMigrationStatus returns the current migration status for entity type T.
	// Indicates whether the table exists, what version it's at, and if migration is needed.
	// Example: status, err := GetMigrationStatus(ctx)
	GetMigrationStatus(ctx context.Context) (MigrationStatus, error)
	
	// GetTableInfo returns detailed information about the current table structure.
	// Includes columns, indexes, constraints, and other database-specific metadata.
	// Example: info, err := GetTableInfo(ctx)
	GetTableInfo(ctx context.Context) (TableInfo, error)
}

// =====================================
// Migration Support Types
// =====================================

// MigrationStatus represents the migration status of a table
type MigrationStatus struct {
	TableExists     bool
	CurrentVersion  string
	RequiredVersion string
	NeedsMigration  bool
	PendingChanges  []string
}

// TableInfo represents detailed information about a database table
type TableInfo struct {
	Name       string
	Columns    []ColumnInfo
	Indexes    []IndexInfo
	Constraints []ConstraintInfo
}

// ColumnInfo represents information about a database column
type ColumnInfo struct {
	Name         string
	Type         string
	IsNullable   bool
	DefaultValue interface{}
	IsPrimaryKey bool
	IsUnique     bool
	MaxLength    int
	Precision    int
	Scale        int
}

// ConstraintInfo represents information about a database constraint
type ConstraintInfo struct {
	Name       string
	Type       string
	Fields     []string
	References string
}

// =====================================
// Association Management
// =====================================

// AssociationManager provides methods for managing entity associations in SQL databases
type AssociationManager interface {
	// Count returns the number of associated records
	Count(ctx context.Context) (int64, error)
	
	// Find retrieves associated records
	Find(ctx context.Context, dest interface{}) error
	
	// Append adds one or more associations
	Append(ctx context.Context, values ...interface{}) error
	
	// Replace replaces all associations with the given values
	Replace(ctx context.Context, values ...interface{}) error
	
	// Delete removes associations (but not the records themselves)
	Delete(ctx context.Context, values ...interface{}) error
	
	// Clear removes all associations
	Clear(ctx context.Context) error
}