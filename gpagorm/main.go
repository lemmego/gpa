// Package gpagorm provides a GORM adapter for the Go Persistence API (GPA)
package gpagorm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/lemmego/gpa"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// =====================================
// Provider Implementation
// =====================================

// Provider implements gpa.Provider using GORM
type Provider struct {
	db     *gorm.DB
	config gpa.Config
}

// Factory implements gpa.ProviderFactory
type Factory struct{}

// Create creates a new GORM provider instance
func (f *Factory) Create(config gpa.Config) (gpa.Provider, error) {
	provider := &Provider{config: config}

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	}

	// Apply custom configurations from options
	if options, ok := config.Options["gorm"]; ok {
		if gormOpts, ok := options.(map[string]interface{}); ok {
			if logLevel, ok := gormOpts["log_level"].(string); ok {
				switch logLevel {
				case "silent":
					gormConfig.Logger = logger.Default.LogMode(logger.Silent)
				case "error":
					gormConfig.Logger = logger.Default.LogMode(logger.Error)
				case "warn":
					gormConfig.Logger = logger.Default.LogMode(logger.Warn)
				case "info":
					gormConfig.Logger = logger.Default.LogMode(logger.Info)
				}
			}

			if singularTable, ok := gormOpts["singular_table"].(bool); ok {
				gormConfig.NamingStrategy = schema.NamingStrategy{
					SingularTable: singularTable,
				}
			}
		}
	}

	// Initialize database connection
	var dialector gorm.Dialector
	var err error

	switch strings.ToLower(config.Driver) {
	case "postgres", "postgresql":
		dialector = postgres.Open(buildPostgresDSN(config))
	case "mysql":
		dialector = mysql.Open(buildMySQLDSN(config))
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(config.Database)
	case "sqlserver", "mssql":
		dialector = sqlserver.Open(buildSQLServerDSN(config))
	default:
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeUnsupported,
			Message: fmt.Sprintf("unsupported driver: %s", config.Driver),
		}
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to connect to database",
			Cause:   err,
		}
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to get underlying sql.DB",
			Cause:   err,
		}
	}

	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	provider.db = db
	return provider, nil
}

// SupportedDrivers returns the list of supported database drivers
func (f *Factory) SupportedDrivers() []string {
	return []string{"postgres", "postgresql", "mysql", "sqlite", "sqlite3", "sqlserver", "mssql"}
}

// Repository returns a repository for the given entity type
func (p *Provider) Repository(entityType reflect.Type) gpa.Repository {
	return &Repository{
		db:         p.db,
		entityType: entityType,
		provider:   p,
	}
}

// RepositoryFor returns a repository for the given entity instance
func (p *Provider) RepositoryFor(entity interface{}) gpa.Repository {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	return p.Repository(entityType)
}

// Configure applies configuration changes
func (p *Provider) Configure(config gpa.Config) error {
	p.config = config
	return nil
}

// Health checks the database connection health
func (p *Provider) Health() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to get underlying sql.DB",
			Cause:   err,
		}
	}
	return sqlDB.Ping()
}

// Close closes the database connection
func (p *Provider) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SupportedFeatures returns the list of supported features
func (p *Provider) SupportedFeatures() []gpa.Feature {
	return []gpa.Feature{
		gpa.FeatureTransactions,
		gpa.FeatureJSONQueries,
		gpa.FeatureIndexing,
		gpa.FeatureAggregation,
	}
}

// ProviderInfo returns information about this provider
func (p *Provider) ProviderInfo() gpa.ProviderInfo {
	return gpa.ProviderInfo{
		Name:         "GORM",
		Version:      "1.0.0",
		DatabaseType: gpa.DatabaseTypeSQL,
		Features:     p.SupportedFeatures(),
	}
}

// =====================================
// Repository Implementation
// =====================================

// Repository implements gpa.Repository and gpa.SQLRepository using GORM
type Repository struct {
	db         *gorm.DB
	entityType reflect.Type
	provider   *Provider
}

// Create creates a new entity
func (r *Repository) Create(ctx context.Context, entity interface{}) error {
	result := r.db.WithContext(ctx).Create(entity)
	return convertGormError(result.Error)
}

// CreateBatch creates multiple entities in a batch
func (r *Repository) CreateBatch(ctx context.Context, entities interface{}) error {
	result := r.db.WithContext(ctx).CreateInBatches(entities, 100)
	return convertGormError(result.Error)
}

// FindByID finds an entity by its ID
func (r *Repository) FindByID(ctx context.Context, id interface{}, dest interface{}) error {
	result := r.db.WithContext(ctx).First(dest, id)
	return convertGormError(result.Error)
}

// FindAll finds all entities matching the given options
func (r *Repository) FindAll(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildQuery(opts...)
	result := query.WithContext(ctx).Find(dest)
	return convertGormError(result.Error)
}

// Update updates an entity
func (r *Repository) Update(ctx context.Context, entity interface{}) error {
	result := r.db.WithContext(ctx).Save(entity)
	return convertGormError(result.Error)
}

// UpdatePartial updates specific fields of an entity
func (r *Repository) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	// Create a new instance of the entity type
	entity := reflect.New(r.entityType).Interface()

	result := r.db.WithContext(ctx).Model(entity).Where("id = ?", id).Updates(updates)
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

// Delete deletes an entity by ID
func (r *Repository) Delete(ctx context.Context, id interface{}) error {
	entity := reflect.New(r.entityType).Interface()
	result := r.db.WithContext(ctx).Delete(entity, id)

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

// DeleteByCondition deletes entities matching the given condition
func (r *Repository) DeleteByCondition(ctx context.Context, condition gpa.Condition) error {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.WithContext(ctx).Model(entity)
	query = r.applyCondition(query, condition)

	result := query.Delete(entity)
	return convertGormError(result.Error)
}

// Query executes a query with the given options
func (r *Repository) Query(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildQuery(opts...)
	result := query.WithContext(ctx).Find(dest)
	return convertGormError(result.Error)
}

// QueryOne executes a query and returns a single result
func (r *Repository) QueryOne(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildQuery(opts...)
	result := query.WithContext(ctx).First(dest)
	return convertGormError(result.Error)
}

// Count counts entities matching the given options
func (r *Repository) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	query := r.buildQuery(opts...)

	var count int64
	entity := reflect.New(r.entityType).Interface()
	result := query.WithContext(ctx).Model(entity).Count(&count)

	return count, convertGormError(result.Error)
}

// Exists checks if any entity matches the given options
func (r *Repository) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	return count > 0, err
}

// Transaction executes a function within a transaction
func (r *Repository) Transaction(ctx context.Context, fn gpa.TransactionFunc) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &Transaction{
			Repository: &Repository{
				db:         tx,
				entityType: r.entityType,
				provider:   r.provider,
			},
		}
		return fn(txRepo)
	})
}

// RawQuery executes a raw SQL query
func (r *Repository) RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error {
	result := r.db.WithContext(ctx).Raw(query, args...).Scan(dest)
	return convertGormError(result.Error)
}

// RawExec executes a raw SQL statement
func (r *Repository) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	result := r.db.WithContext(ctx).Exec(query, args...)
	if result.Error != nil {
		return nil, convertGormError(result.Error)
	}

	return &Result{
		rowsAffected: result.RowsAffected,
	}, nil
}

// GetEntityInfo returns metadata about the entity
func (r *Repository) GetEntityInfo(entity interface{}) (*gpa.EntityInfo, error) {
	stmt := &gorm.Statement{DB: r.db}
	err := stmt.Parse(entity)
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

// Close closes the repository (no-op for GORM)
func (r *Repository) Close() error {
	return nil
}

// =====================================
// Relationship Methods
// =====================================

// FindWithRelations finds entities with preloaded relationships
func (r *Repository) FindWithRelations(ctx context.Context, dest interface{}, relations []string, opts ...gpa.QueryOption) error {
	// Add preloads to the options
	allOpts := make([]gpa.QueryOption, 0, len(opts)+1)
	allOpts = append(allOpts, gpa.Preload(relations...))
	allOpts = append(allOpts, opts...)

	return r.Query(ctx, dest, allOpts...)
}

// FindByIDWithRelations finds an entity by ID with preloaded relationships
func (r *Repository) FindByIDWithRelations(ctx context.Context, id interface{}, dest interface{}, relations []string) error {
	db := r.db.WithContext(ctx)

	// Apply preloads
	for _, relation := range relations {
		db = db.Preload(relation)
	}

	result := db.First(dest, id)
	return convertGormError(result.Error)
}

// Association provides access to GORM's association mode for relationship management
func (r *Repository) Association(ctx context.Context, entity interface{}, field string) AssociationManager {
	return &associationManager{
		db:     r.db.WithContext(ctx),
		entity: entity,
		field:  field,
	}
}

// AssociationManager provides methods for managing associations
type AssociationManager interface {
	Find(dest interface{}) error
	Append(values ...interface{}) error
	Replace(values ...interface{}) error
	Delete(values ...interface{}) error
	Clear() error
	Count() (int64, error)
}

// associationManager implements AssociationManager using GORM
type associationManager struct {
	db     *gorm.DB
	entity interface{}
	field  string
}

func (a *associationManager) Find(dest interface{}) error {
	err := a.db.Model(a.entity).Association(a.field).Find(dest)
	return convertGormError(err)
}

func (a *associationManager) Append(values ...interface{}) error {
	err := a.db.Model(a.entity).Association(a.field).Append(values...)
	return convertGormError(err)
}

func (a *associationManager) Replace(values ...interface{}) error {
	err := a.db.Model(a.entity).Association(a.field).Replace(values...)
	return convertGormError(err)
}

func (a *associationManager) Delete(values ...interface{}) error {
	err := a.db.Model(a.entity).Association(a.field).Delete(values...)
	return convertGormError(err)
}

func (a *associationManager) Clear() error {
	err := a.db.Model(a.entity).Association(a.field).Clear()
	return convertGormError(err)
}

func (a *associationManager) Count() (int64, error) {
	count := a.db.Model(a.entity).Association(a.field).Count()
	return count, nil
}

// =====================================
// SQL Repository Implementation
// =====================================

// FindBySQL executes a raw SQL query
func (r *Repository) FindBySQL(ctx context.Context, sql string, args []interface{}, dest interface{}) error {
	return r.RawQuery(ctx, sql, args, dest)
}

// ExecSQL executes a raw SQL statement
func (r *Repository) ExecSQL(ctx context.Context, sql string, args ...interface{}) (gpa.Result, error) {
	return r.RawExec(ctx, sql, args)
}

// CreateTable creates a table for the entity
func (r *Repository) CreateTable(ctx context.Context, entity interface{}) error {
	migrator := r.db.Migrator()
	if migrator.HasTable(entity) {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: "table already exists",
		}
	}

	err := migrator.CreateTable(entity)
	return convertGormError(err)
}

// DropTable drops the table for the entity
func (r *Repository) DropTable(ctx context.Context, entity interface{}) error {
	migrator := r.db.Migrator()
	err := migrator.DropTable(entity)
	return convertGormError(err)
}

// MigrateTable migrates the table schema for the entity
func (r *Repository) MigrateTable(ctx context.Context, entity interface{}) error {
	err := r.db.AutoMigrate(entity)
	return convertGormError(err)
}

// CreateIndex creates an index on the specified fields
func (r *Repository) CreateIndex(ctx context.Context, entity interface{}, fields []string, unique bool) error {
	migrator := r.db.Migrator()

	// Generate index name
	stmt := &gorm.Statement{DB: r.db}
	err := stmt.Parse(entity)
	if err != nil {
		return convertGormError(err)
	}

	indexName := fmt.Sprintf("idx_%s_%s", stmt.Schema.Table, strings.Join(fields, "_"))

	// Check if index already exists
	if migrator.HasIndex(entity, indexName) {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: fmt.Sprintf("index %s already exists", indexName),
		}
	}

	err = migrator.CreateIndex(entity, indexName)
	return convertGormError(err)
}

// DropIndex drops an index
func (r *Repository) DropIndex(ctx context.Context, entity interface{}, indexName string) error {
	migrator := r.db.Migrator()
	err := migrator.DropIndex(entity, indexName)
	return convertGormError(err)
}

// =====================================
// Transaction Implementation
// =====================================

// Transaction implements gpa.Transaction
type Transaction struct {
	*Repository
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	// GORM handles commit automatically when the transaction function returns nil
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	// GORM handles rollback automatically when the transaction function returns an error
	return nil
}

// =====================================
// Result Implementation
// =====================================

// Result implements gpa.Result
type Result struct {
	lastInsertId int64
	rowsAffected int64
}

// LastInsertId returns the last insert ID
func (r *Result) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

// RowsAffected returns the number of affected rows
func (r *Result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// =====================================
// Query Building Helpers
// =====================================

// buildQuery builds a GORM query from GPA query options
func (r *Repository) buildQuery(opts ...gpa.QueryOption) *gorm.DB {
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
		db = db.Order(fmt.Sprintf("%s %s", order.Field, order.Direction))
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
		joinClause := fmt.Sprintf("%s JOIN %s", join.Type, join.Table)
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
		db = db.Group(strings.Join(query.Groups, ", "))
	}

	// Apply having conditions
	for _, having := range query.Having {
		db = r.applyHaving(db, having)
	}

	// Apply distinct
	if query.Distinct {
		db = db.Distinct()
	}

	// Apply locking
	if query.Lock != gpa.LockNone {
		switch query.Lock {
		case gpa.LockForUpdate:
			db = db.Clauses(clause.Locking{Strength: "UPDATE"})
		case gpa.LockForShare:
			db = db.Clauses(clause.Locking{Strength: "SHARE"})
		}
	}

	return db
}

// applyCondition applies a condition to the GORM query
func (r *Repository) applyCondition(db *gorm.DB, condition gpa.Condition) *gorm.DB {
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		return r.applyBasicCondition(db, cond)
	case gpa.CompositeCondition:
		return r.applyCompositeCondition(db, cond)
	case gpa.SubQueryCondition:
		return r.applySubQueryCondition(db, cond)
	default:
		// Fallback to string representation
		return db.Where(condition.String(), condition.Value())
	}
}

// applyBasicCondition applies a basic condition
func (r *Repository) applyBasicCondition(db *gorm.DB, condition gpa.BasicCondition) *gorm.DB {
	field := condition.Field()
	op := condition.Operator()
	value := condition.Value()

	switch op {
	case gpa.OpEqual:
		return db.Where(fmt.Sprintf("%s = ?", field), value)
	case gpa.OpNotEqual:
		return db.Where(fmt.Sprintf("%s != ?", field), value)
	case gpa.OpGreaterThan:
		return db.Where(fmt.Sprintf("%s > ?", field), value)
	case gpa.OpGreaterThanOrEqual:
		return db.Where(fmt.Sprintf("%s >= ?", field), value)
	case gpa.OpLessThan:
		return db.Where(fmt.Sprintf("%s < ?", field), value)
	case gpa.OpLessThanOrEqual:
		return db.Where(fmt.Sprintf("%s <= ?", field), value)
	case gpa.OpLike:
		return db.Where(fmt.Sprintf("%s LIKE ?", field), value)
	case gpa.OpNotLike:
		return db.Where(fmt.Sprintf("%s NOT LIKE ?", field), value)
	case gpa.OpIn:
		return db.Where(fmt.Sprintf("%s IN ?", field), value)
	case gpa.OpNotIn:
		return db.Where(fmt.Sprintf("%s NOT IN ?", field), value)
	case gpa.OpIsNull:
		return db.Where(fmt.Sprintf("%s IS NULL", field))
	case gpa.OpIsNotNull:
		return db.Where(fmt.Sprintf("%s IS NOT NULL", field))
	case gpa.OpBetween:
		if values, ok := value.([]interface{}); ok && len(values) == 2 {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), values[0], values[1])
		}
		return db
	case gpa.OpExists:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			return db.Where(fmt.Sprintf("EXISTS (%s)", subQuery.Query), subQuery.Args...)
		}
		return db
	case gpa.OpNotExists:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			return db.Where(fmt.Sprintf("NOT EXISTS (%s)", subQuery.Query), subQuery.Args...)
		}
		return db
	case gpa.OpInSubQuery:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			return db.Where(fmt.Sprintf("%s IN (%s)", field, subQuery.Query), subQuery.Args...)
		}
		return db
	case gpa.OpNotInSubQuery:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			return db.Where(fmt.Sprintf("%s NOT IN (%s)", field, subQuery.Query), subQuery.Args...)
		}
		return db
	default:
		// Fallback
		return db.Where(fmt.Sprintf("%s %s ?", field, op), value)
	}
}

// applyCompositeCondition applies a composite condition
func (r *Repository) applyCompositeCondition(db *gorm.DB, condition gpa.CompositeCondition) *gorm.DB {
	if len(condition.Conditions) == 0 {
		return db
	}

	// Build the composite condition properly
	var parts []string
	var values []interface{}

	for _, subCondition := range condition.Conditions {
		switch subCond := subCondition.(type) {
		case gpa.BasicCondition:
			switch subCond.Operator() {
			case gpa.OpIsNull:
				parts = append(parts, fmt.Sprintf("%s IS NULL", subCond.Field()))
			case gpa.OpIsNotNull:
				parts = append(parts, fmt.Sprintf("%s IS NOT NULL", subCond.Field()))
			case gpa.OpBetween:
				if betweenVals, ok := subCond.Value().([]interface{}); ok && len(betweenVals) == 2 {
					parts = append(parts, fmt.Sprintf("%s BETWEEN ? AND ?", subCond.Field()))
					values = append(values, betweenVals[0], betweenVals[1])
				}
			default:
				parts = append(parts, fmt.Sprintf("%s %s ?", subCond.Field(), subCond.Operator()))
				values = append(values, subCond.Value())
			}
		case gpa.CompositeCondition:
			// For nested composite conditions, we need to wrap them in parentheses
			nestedParts := []string{}
			for _, nestedCond := range subCond.Conditions {
				if basicCond, ok := nestedCond.(gpa.BasicCondition); ok {
					switch basicCond.Operator() {
					case gpa.OpIsNull:
						nestedParts = append(nestedParts, fmt.Sprintf("%s IS NULL", basicCond.Field()))
					case gpa.OpIsNotNull:
						nestedParts = append(nestedParts, fmt.Sprintf("%s IS NOT NULL", basicCond.Field()))
					default:
						nestedParts = append(nestedParts, fmt.Sprintf("%s %s ?", basicCond.Field(), basicCond.Operator()))
						values = append(values, basicCond.Value())
					}
				}
			}
			if len(nestedParts) > 0 {
				parts = append(parts, fmt.Sprintf("(%s)", strings.Join(nestedParts, fmt.Sprintf(" %s ", subCond.Logic))))
			}
		}
	}

	if len(parts) == 0 {
		return db
	}

	whereClause := strings.Join(parts, fmt.Sprintf(" %s ", condition.Logic))

	if len(values) > 0 {
		return db.Where(whereClause, values...)
	}
	return db.Where(whereClause)
}

// applySubQueryCondition applies a subquery condition
func (r *Repository) applySubQueryCondition(db *gorm.DB, condition gpa.SubQueryCondition) *gorm.DB {
	subQuery := condition.SubQuery

	switch subQuery.Type {
	case gpa.SubQueryTypeExists:
		if subQuery.Operator == gpa.OpNotExists {
			return db.Where(fmt.Sprintf("NOT EXISTS (%s)", subQuery.Query), subQuery.Args...)
		}
		return db.Where(fmt.Sprintf("EXISTS (%s)", subQuery.Query), subQuery.Args...)

	case gpa.SubQueryTypeIn:
		if subQuery.Operator == gpa.OpNotInSubQuery {
			return db.Where(fmt.Sprintf("%s NOT IN (%s)", subQuery.Field, subQuery.Query), subQuery.Args...)
		}
		return db.Where(fmt.Sprintf("%s IN (%s)", subQuery.Field, subQuery.Query), subQuery.Args...)

	case gpa.SubQueryTypeScalar, gpa.SubQueryTypeCorrelated:
		// For scalar and correlated subqueries, use the operator directly
		return db.Where(fmt.Sprintf("%s %s (%s)", subQuery.Field, subQuery.Operator, subQuery.Query), subQuery.Args...)

	default:
		// Fallback to string representation
		return db.Where(condition.String())
	}
}

// applyHaving applies a having condition
func (r *Repository) applyHaving(db *gorm.DB, condition gpa.Condition) *gorm.DB {
	// Similar to applyCondition but for HAVING clause
	return db.Having(condition.String(), condition.Value())
}

// =====================================
// Error Conversion
// =====================================

// convertGormError converts GORM errors to GPA errors
func convertGormError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case gorm.ErrRecordNotFound:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "record not found",
			Cause:   err,
		}
	case gorm.ErrInvalidTransaction:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeTransaction,
			Message: "invalid transaction",
			Cause:   err,
		}
	case gorm.ErrNotImplemented:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeUnsupported,
			Message: "operation not implemented",
			Cause:   err,
		}
	case gorm.ErrMissingWhereClause:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "missing where clause",
			Cause:   err,
		}
	case gorm.ErrUnsupportedRelation:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeUnsupported,
			Message: "unsupported relation",
			Cause:   err,
		}
	case gorm.ErrPrimaryKeyRequired:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "primary key required",
			Cause:   err,
		}
	case gorm.ErrModelValueRequired:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "model value required",
			Cause:   err,
		}
	case gorm.ErrInvalidData:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "invalid data",
			Cause:   err,
		}
	default:
		// Check for common database constraint errors
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique") {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeDuplicate,
				Message: "duplicate key violation",
				Cause:   err,
			}
		}
		if strings.Contains(errStr, "foreign key") || strings.Contains(errStr, "constraint") {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeConstraint,
				Message: "constraint violation",
				Cause:   err,
			}
		}
		if strings.Contains(errStr, "timeout") {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeTimeout,
				Message: "operation timeout",
				Cause:   err,
			}
		}
		if strings.Contains(errStr, "connection") {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeConnection,
				Message: "connection error",
				Cause:   err,
			}
		}

		// Default to generic error
		return gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "database operation failed",
			Cause:   err,
		}
	}
}

// =====================================
// DSN Builders
// =====================================

// buildPostgresDSN builds a PostgreSQL DSN
func buildPostgresDSN(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database)

	if config.SSL.Enabled {
		dsn += " sslmode=" + config.SSL.Mode
		if config.SSL.CertFile != "" {
			dsn += " sslcert=" + config.SSL.CertFile
		}
		if config.SSL.KeyFile != "" {
			dsn += " sslkey=" + config.SSL.KeyFile
		}
		if config.SSL.CAFile != "" {
			dsn += " sslrootcert=" + config.SSL.CAFile
		}
	} else {
		dsn += " sslmode=disable"
	}

	return dsn
}

// buildMySQLDSN builds a MySQL DSN
func buildMySQLDSN(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	if config.SSL.Enabled {
		dsn += "&tls=" + config.SSL.Mode
	}

	return dsn
}

// buildSQLServerDSN builds a SQL Server DSN
func buildSQLServerDSN(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
		config.Username, config.Password, config.Host, config.Port, config.Database)
}

// =====================================
// Registration
// =====================================

// init registers the GORM provider factory
func init() {
	gpa.RegisterProvider("gorm", &Factory{})
}
