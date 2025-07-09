// Package gpagorm provides a GORM adapter for the Go Persistence API (GPA)
package gpagorm

import (
	"context"
	"errors"

	"github.com/lemmego/gpa"
	"gorm.io/gorm"
)

// SQLResult implements gpa.Result interface
type SQLResult struct {
	rowsAffected int64
	lastInsertID int64
}

// RowsAffected returns the number of rows affected
func (r *SQLResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// LastInsertId returns the last insert ID
func (r *SQLResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// =====================================
// Generic GORM Repository Implementation
// =====================================

// Repository implements type-safe GORM operations using Go generics.
// Provides compile-time type safety for all CRUD and SQL operations.
type Repository[T any] struct {
	db       *gorm.DB
	provider *Provider
}

// convertGormError converts GORM errors to GPA errors
func convertGormError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gpa.NewError(gpa.ErrorTypeNotFound, "record not found")
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return gpa.NewError(gpa.ErrorTypeDuplicate, "duplicate key")
	}
	// If it's already a GPA error, return it as is
	if gpaErr, ok := err.(gpa.GPAError); ok {
		return gpaErr
	}
	return gpa.NewErrorWithCause(gpa.ErrorTypeDatabase, "database error", err)
}

// NewRepository creates a new generic GORM repository for type T.
// Example: userRepo := NewRepository[User](db, provider)
func NewRepository[T any](db *gorm.DB, provider *Provider) *Repository[T] {
	return &Repository[T]{
		db:       db,
		provider: provider,
	}
}

// =====================================
// RepositoryG[T] Implementation
// =====================================

// Create inserts a new entity with compile-time type safety.
func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	result := r.db.WithContext(ctx).Create(entity)
	return convertGormError(result.Error)
}

// CreateBatch inserts multiple entities with compile-time type safety.
func (r *Repository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	result := r.db.WithContext(ctx).CreateInBatches(entities, 100)
	return convertGormError(result.Error)
}

// FindByID retrieves a single entity by ID with compile-time type safety.
func (r *Repository[T]) FindByID(ctx context.Context, id interface{}) (*T, error) {
	var entity T
	result := r.db.WithContext(ctx).First(&entity, id)
	if err := convertGormError(result.Error); err != nil {
		return nil, err
	}
	return &entity, nil
}

// FindAll retrieves all entities with compile-time type safety.
func (r *Repository[T]) FindAll(ctx context.Context, opts ...gpa.QueryOption) ([]*T, error) {
	query := r.buildQuery(opts...)
	var entities []*T
	result := query.WithContext(ctx).Find(&entities)
	if err := convertGormError(result.Error); err != nil {
		return nil, err
	}
	return entities, nil
}

// Update modifies an existing entity with compile-time type safety.
func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	result := r.db.WithContext(ctx).Save(entity)
	return convertGormError(result.Error)
}

// UpdatePartial modifies specific fields of an entity.
func (r *Repository[T]) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	var entity T
	result := r.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return convertGormError(result.Error)
	}
	if result.RowsAffected == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found",
		}
	}
	return nil
}

// Delete removes an entity by ID with compile-time type safety.
func (r *Repository[T]) Delete(ctx context.Context, id interface{}) error {
	var entity T
	result := r.db.WithContext(ctx).Delete(&entity, id)
	if result.Error != nil {
		return convertGormError(result.Error)
	}
	if result.RowsAffected == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found",
		}
	}
	return nil
}

// DeleteByCondition removes entities matching a condition.
func (r *Repository[T]) DeleteByCondition(ctx context.Context, condition gpa.Condition) error {
	var entity T
	query := r.db.WithContext(ctx).Model(&entity)
	query = r.applyCondition(query, condition)
	result := query.Delete(&entity)
	return convertGormError(result.Error)
}

// Query retrieves entities based on query options with compile-time type safety.
func (r *Repository[T]) Query(ctx context.Context, opts ...gpa.QueryOption) ([]*T, error) {
	query := r.buildQuery(opts...)
	var entities []*T
	result := query.WithContext(ctx).Find(&entities)
	if err := convertGormError(result.Error); err != nil {
		return nil, err
	}
	return entities, nil
}

// QueryOne retrieves a single entity based on query options.
func (r *Repository[T]) QueryOne(ctx context.Context, opts ...gpa.QueryOption) (*T, error) {
	query := r.buildQuery(opts...)
	var entity T
	result := query.WithContext(ctx).First(&entity)
	if err := convertGormError(result.Error); err != nil {
		return nil, err
	}
	return &entity, nil
}

// Count returns the number of entities matching query options.
func (r *Repository[T]) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	query := r.buildQuery(opts...)
	var count int64
	var entity T
	result := query.WithContext(ctx).Model(&entity).Count(&count)
	return count, convertGormError(result.Error)
}

// Exists checks if any entity matches the query options.
func (r *Repository[T]) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	return count > 0, err
}

// Transaction executes a function within a transaction with type safety.
func (r *Repository[T]) Transaction(ctx context.Context, fn gpa.TransactionFunc[T]) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &Transaction[T]{
			Repository: &Repository[T]{
				db:       tx,
				provider: r.provider,
			},
		}
		return fn(txRepo)
	})
}

// RawQuery executes a raw SQL query with compile-time type safety.
func (r *Repository[T]) RawQuery(ctx context.Context, query string, args []interface{}) ([]*T, error) {
	var entities []*T
	result := r.db.WithContext(ctx).Raw(query, args...).Scan(&entities)
	if err := convertGormError(result.Error); err != nil {
		return nil, err
	}
	return entities, nil
}

// RawExec executes a raw SQL statement.
func (r *Repository[T]) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	result := r.db.WithContext(ctx).Exec(query, args...)
	if result.Error != nil {
		return nil, convertGormError(result.Error)
	}
	return &SQLResult{
		rowsAffected: result.RowsAffected,
	}, nil
}

// GetEntityInfo returns metadata about entity type T.
func (r *Repository[T]) GetEntityInfo() (*gpa.EntityInfo, error) {
	var zero T
	stmt := &gorm.Statement{DB: r.db}
	err := stmt.Parse(&zero)
	if err != nil {
		return nil, convertGormError(err)
	}

	info := &gpa.EntityInfo{
		Name:      stmt.Schema.Name,
		TableName: stmt.Schema.Table,
		Fields:    make([]gpa.FieldInfo, 0, len(stmt.Schema.Fields)),
	}

	// Convert GORM fields to GPA fields
	for _, field := range stmt.Schema.Fields {
		fieldInfo := gpa.FieldInfo{
			Name:            field.Name,
			Type:            field.FieldType,
			DatabaseType:    string(field.DataType),
			Tag:             string(field.Tag),
			IsPrimaryKey:    field.PrimaryKey,
			IsNullable:      field.NotNull == false,
			IsAutoIncrement: field.AutoIncrement,
			DefaultValue:    field.DefaultValue,
		}

		if field.Size > 0 {
			fieldInfo.MaxLength = int(field.Size)
		}
		if field.Precision > 0 {
			fieldInfo.Precision = int(field.Precision)
		}
		if field.Scale > 0 {
			fieldInfo.Scale = int(field.Scale)
		}

		info.Fields = append(info.Fields, fieldInfo)

		if field.PrimaryKey {
			info.PrimaryKey = append(info.PrimaryKey, field.Name)
		}
	}

	return info, nil
}

// Close closes the repository (no-op for GORM).
func (r *Repository[T]) Close() error {
	return nil
}

// =====================================
// SQLRepositoryG[T] Implementation
// =====================================

// FindBySQL executes a raw SQL SELECT query with compile-time type safety.
func (r *Repository[T]) FindBySQL(ctx context.Context, sql string, args []interface{}) ([]*T, error) {
	return r.RawQuery(ctx, sql, args)
}

// ExecSQL executes a raw SQL statement.
func (r *Repository[T]) ExecSQL(ctx context.Context, sql string, args ...interface{}) (gpa.Result, error) {
	return r.RawExec(ctx, sql, args)
}

// FindWithRelations retrieves entities with preloaded relationships.
func (r *Repository[T]) FindWithRelations(ctx context.Context, relations []string, opts ...gpa.QueryOption) ([]*T, error) {
	// Add preloads to the options
	allOpts := make([]gpa.QueryOption, 0, len(opts)+1)
	allOpts = append(allOpts, gpa.Preload(relations...))
	allOpts = append(allOpts, opts...)
	return r.Query(ctx, allOpts...)
}

// FindByIDWithRelations retrieves an entity by ID with preloaded relationships.
func (r *Repository[T]) FindByIDWithRelations(ctx context.Context, id interface{}, relations []string) (*T, error) {
	db := r.db.WithContext(ctx)
	
	// Apply preloads
	for _, relation := range relations {
		db = db.Preload(relation)
	}
	
	var entity T
	result := db.First(&entity, id)
	if err := convertGormError(result.Error); err != nil {
		return nil, err
	}
	return &entity, nil
}

// CreateTable creates a new table for entity type T.
func (r *Repository[T]) CreateTable(ctx context.Context) error {
	var zero T
	migrator := r.db.Migrator()
	if migrator.HasTable(&zero) {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: "table already exists",
		}
	}
	err := migrator.CreateTable(&zero)
	return convertGormError(err)
}

// DropTable drops the table for entity type T.
func (r *Repository[T]) DropTable(ctx context.Context) error {
	var zero T
	migrator := r.db.Migrator()
	err := migrator.DropTable(&zero)
	return convertGormError(err)
}

// CreateIndex creates an index on the specified fields.
func (r *Repository[T]) CreateIndex(ctx context.Context, fields []string, unique bool) error {
	var zero T
	migrator := r.db.Migrator()
	
	// Generate index name
	stmt := &gorm.Statement{DB: r.db}
	err := stmt.Parse(&zero)
	if err != nil {
		return convertGormError(err)
	}
	
	indexName := "idx_" + stmt.Schema.Table + "_" + fields[0]
	for _, field := range fields[1:] {
		indexName += "_" + field
	}
	
	// Check if index already exists
	if migrator.HasIndex(&zero, indexName) {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: "index already exists: " + indexName,
		}
	}
	
	err = migrator.CreateIndex(&zero, indexName)
	return convertGormError(err)
}

// DropIndex removes an index.
func (r *Repository[T]) DropIndex(ctx context.Context, indexName string) error {
	var zero T
	migrator := r.db.Migrator()
	err := migrator.DropIndex(&zero, indexName)
	return convertGormError(err)
}

// =====================================
// MigratableRepositoryG[T] Implementation
// =====================================

// MigrateTable migrates the table schema for entity type T.
func (r *Repository[T]) MigrateTable(ctx context.Context) error {
	var zero T
	err := r.db.AutoMigrate(&zero)
	return convertGormError(err)
}

// GetMigrationStatus returns the current migration status for entity type T.
func (r *Repository[T]) GetMigrationStatus(ctx context.Context) (gpa.MigrationStatus, error) {
	var zero T
	migrator := r.db.Migrator()
	
	status := gpa.MigrationStatus{
		TableExists:     migrator.HasTable(&zero),
		CurrentVersion:  "current",
		RequiredVersion: "current",
		NeedsMigration:  false,
		PendingChanges:  []string{},
	}
	
	// In a more sophisticated implementation, you would compare the current schema
	// with the expected schema from the struct definition
	
	return status, nil
}

// GetTableInfo returns detailed information about the current table structure.
func (r *Repository[T]) GetTableInfo(ctx context.Context) (gpa.TableInfo, error) {
	var zero T
	stmt := &gorm.Statement{DB: r.db}
	err := stmt.Parse(&zero)
	if err != nil {
		return gpa.TableInfo{}, convertGormError(err)
	}
	
	info := gpa.TableInfo{
		Name:        stmt.Schema.Table,
		Columns:     make([]gpa.ColumnInfo, 0, len(stmt.Schema.Fields)),
		Indexes:     []gpa.IndexInfo{},
		Constraints: []gpa.ConstraintInfo{},
	}
	
	// Convert GORM fields to column info
	for _, field := range stmt.Schema.Fields {
		columnInfo := gpa.ColumnInfo{
			Name:         field.DBName,
			Type:         string(field.DataType),
			IsNullable:   !field.NotNull,
			IsPrimaryKey: field.PrimaryKey,
			IsUnique:     field.Unique,
			DefaultValue: field.DefaultValue,
		}
		
		if field.Size > 0 {
			columnInfo.MaxLength = int(field.Size)
		}
		if field.Precision > 0 {
			columnInfo.Precision = int(field.Precision)
		}
		if field.Scale > 0 {
			columnInfo.Scale = int(field.Scale)
		}
		
		info.Columns = append(info.Columns, columnInfo)
	}
	
	return info, nil
}

// =====================================
// TransactionG Implementation
// =====================================

// TransactionG implements gpa.TransactionG using GORM with type safety.
type Transaction[T any] struct {
	*Repository[T]
}

// Commit commits the transaction (handled automatically by GORM).
func (t *Transaction[T]) Commit() error {
	return nil
}

// Rollback rolls back the transaction (handled automatically by GORM).
func (t *Transaction[T]) Rollback() error {
	return nil
}

// SetSavepoint creates a savepoint within the transaction.
func (t *Transaction[T]) SetSavepoint(name string) error {
	return convertGormError(t.db.Exec("SAVEPOINT " + name).Error)
}

// RollbackToSavepoint rolls back to a previously created savepoint.
func (t *Transaction[T]) RollbackToSavepoint(name string) error {
	return convertGormError(t.db.Exec("ROLLBACK TO SAVEPOINT " + name).Error)
}

// =====================================
// Helper Methods
// =====================================

// buildQuery builds a GORM query from GPA query options
func (r *Repository[T]) buildQuery(opts ...gpa.QueryOption) *gorm.DB {
	query := &gpa.Query{}
	
	// Apply all options
	for _, opt := range opts {
		opt.Apply(query)
	}
	
	db := r.db
	
	// Apply conditions
	for _, condition := range query.Conditions {
		db = r.applyCondition(db, condition)
	}
	
	// Apply field selection
	if len(query.Fields) > 0 {
		db = db.Select(query.Fields)
	}
	
	// Apply ordering
	for _, order := range query.Orders {
		db = db.Order(order.Field + " " + string(order.Direction))
	}
	
	// Apply limit
	if query.Limit != nil {
		db = db.Limit(*query.Limit)
	}
	
	// Apply offset
	if query.Offset != nil {
		db = db.Offset(*query.Offset)
	}
	
	// Apply joins
	for _, join := range query.Joins {
		joinClause := string(join.Type) + " JOIN " + join.Table
		if join.Alias != "" {
			joinClause += " AS " + join.Alias
		}
		if join.Condition != "" {
			joinClause += " ON " + join.Condition
		}
		db = db.Joins(joinClause)
	}
	
	// Apply preloads
	for _, preload := range query.Preloads {
		db = db.Preload(preload)
	}
	
	// Apply grouping
	if len(query.Groups) > 0 {
		for _, group := range query.Groups {
			db = db.Group(group)
		}
	}
	
	// Apply having conditions
	for _, having := range query.Having {
		db = r.applyHaving(db, having)
	}
	
	// Apply distinct
	if query.Distinct {
		db = db.Distinct()
	}
	
	return db
}

// applyCondition applies a condition to the GORM query
func (r *Repository[T]) applyCondition(db *gorm.DB, condition gpa.Condition) *gorm.DB {
	// Basic implementation - can be enhanced later
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		field := cond.Field()
		operator := cond.Operator()
		value := cond.Value()
		
		switch operator {
		case gpa.OpEqual:
			return db.Where(field+" = ?", value)
		case gpa.OpNotEqual:
			return db.Where(field+" != ?", value)
		case gpa.OpGreaterThan:
			return db.Where(field+" > ?", value)
		case gpa.OpGreaterThanOrEqual:
			return db.Where(field+" >= ?", value)
		case gpa.OpLessThan:
			return db.Where(field+" < ?", value)
		case gpa.OpLessThanOrEqual:
			return db.Where(field+" <= ?", value)
		case gpa.OpLike:
			return db.Where(field+" LIKE ?", value)
		case gpa.OpNotLike:
			return db.Where(field+" NOT LIKE ?", value)
		case gpa.OpIn:
			return db.Where(field+" IN ?", value)
		case gpa.OpNotIn:
			return db.Where(field+" NOT IN ?", value)
		case gpa.OpIsNull:
			return db.Where(field + " IS NULL")
		case gpa.OpIsNotNull:
			return db.Where(field + " IS NOT NULL")
		default:
			return db.Where(field+" = ?", value)
		}
	default:
		// For complex conditions, return the query unchanged for now
		return db
	}
}

// applyHaving applies a having condition
func (r *Repository[T]) applyHaving(db *gorm.DB, condition gpa.Condition) *gorm.DB {
	// Basic implementation - similar to applyCondition but for HAVING clause
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		field := cond.Field()
		operator := cond.Operator()
		value := cond.Value()
		
		switch operator {
		case gpa.OpEqual:
			return db.Having(field+" = ?", value)
		case gpa.OpNotEqual:
			return db.Having(field+" != ?", value)
		case gpa.OpGreaterThan:
			return db.Having(field+" > ?", value)
		case gpa.OpGreaterThanOrEqual:
			return db.Having(field+" >= ?", value)
		case gpa.OpLessThan:
			return db.Having(field+" < ?", value)
		case gpa.OpLessThanOrEqual:
			return db.Having(field+" <= ?", value)
		default:
			return db.Having(field+" = ?", value)
		}
	default:
		// For complex conditions, return the query unchanged for now
		return db
	}
}

// =====================================
// Compile-time Interface Checks
// =====================================

var (
	_ gpa.Repository[any]           = (*Repository[any])(nil)
	_ gpa.SQLRepository[any]        = (*Repository[any])(nil)
	_ gpa.MigratableRepository[any] = (*Repository[any])(nil)
	_ gpa.Transaction[any]          = (*Transaction[any])(nil)
)