// Package gpabun provides a Bun adapter for the Go Persistence API (GPA)
package gpabun

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/uptrace/bun/dialect"
	"reflect"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lemmego/gpa"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

// =====================================
// Provider Implementation
// =====================================

// Provider implements gpa.Provider using Bun
type Provider struct {
	db     *bun.DB
	config gpa.Config
}

// Factory implements gpa.ProviderFactory
type Factory struct{}

// Create creates a new Bun provider instance
func (f *Factory) Create(config gpa.Config) (gpa.Provider, error) {
	provider := &Provider{config: config}

	// Initialize database connection
	var sqlDB *sql.DB
	var err error

	switch strings.ToLower(config.Driver) {
	case "postgres", "postgresql":
		sqlDB, err = createPostgresConnection(config)
	case "mysql":
		sqlDB, err = createMySQLConnection(config)
	case "sqlite", "sqlite3":
		sqlDB, err = createSQLiteConnection(config)
	default:
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeUnsupported,
			Message: fmt.Sprintf("unsupported driver: %s", config.Driver),
		}
	}

	if err != nil {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to connect to database",
			Cause:   err,
		}
	}

	// Configure connection pool
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

	// Create Bun database instance
	var bunDB *bun.DB
	switch strings.ToLower(config.Driver) {
	case "postgres", "postgresql":
		bunDB = bun.NewDB(sqlDB, pgdialect.New())
	case "mysql":
		bunDB = bun.NewDB(sqlDB, mysqldialect.New())
	case "sqlite", "sqlite3":
		bunDB = bun.NewDB(sqlDB, sqlitedialect.New())
	}

	// Configure Bun options
	if options, ok := config.Options["bun"]; ok {
		if bunOpts, ok := options.(map[string]interface{}); ok {
			// Add query hook for logging if enabled
			if logLevel, ok := bunOpts["log_level"].(string); ok && logLevel != "silent" {
				bunDB.AddQueryHook(bundebug.NewQueryHook(
					bundebug.WithVerbose(logLevel == "debug"),
				))
			}
		}
	}

	provider.db = bunDB
	return provider, nil
}

// SupportedDrivers returns the list of supported database drivers
func (f *Factory) SupportedDrivers() []string {
	return []string{"postgres", "postgresql", "mysql", "sqlite", "sqlite3"}
}

// Repository returns a repository for the given entity type
func (p *Provider) Repository(entityType reflect.Type) gpa.Repository {
	return &Repository{
		db:         p.db, // bun.DB implements bun.IDB
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
	sqlDB := p.db.DB
	return sqlDB.Ping()
}

// Close closes the database connection
func (p *Provider) Close() error {
	return p.db.Close()
}

// SupportedFeatures returns the list of supported features
func (p *Provider) SupportedFeatures() []gpa.Feature {
	return []gpa.Feature{
		gpa.FeatureTransactions,
		gpa.FeatureJSONQueries,
		gpa.FeatureIndexing,
		gpa.FeatureAggregation,
		gpa.FeatureFullTextSearch,
	}
}

// ProviderInfo returns information about this provider
func (p *Provider) ProviderInfo() gpa.ProviderInfo {
	return gpa.ProviderInfo{
		Name:         "Bun",
		Version:      "1.0.0",
		DatabaseType: gpa.DatabaseTypeSQL,
		Features:     p.SupportedFeatures(),
	}
}

// =====================================
// Repository Implementation
// =====================================

// Repository implements gpa.Repository and gpa.SQLRepository using Bun
type Repository struct {
	db         bun.IDB // Use interface instead of concrete type
	entityType reflect.Type
	provider   *Provider
}

// Create creates a new entity
func (r *Repository) Create(ctx context.Context, entity interface{}) error {
	_, err := r.db.NewInsert().Model(entity).Exec(ctx)
	return convertBunError(err)
}

// CreateBatch creates multiple entities in a batch
func (r *Repository) CreateBatch(ctx context.Context, entities interface{}) error {
	_, err := r.db.NewInsert().Model(entities).Exec(ctx)
	return convertBunError(err)
}

// FindByID finds an entity by its ID
func (r *Repository) FindByID(ctx context.Context, id interface{}, dest interface{}) error {
	err := r.db.NewSelect().Model(dest).Where("id = ?", id).Scan(ctx)
	return convertBunError(err)
}

// FindAll finds all entities matching the given options
func (r *Repository) FindAll(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildSelectQuery(dest, opts...)
	err := query.Scan(ctx)
	return convertBunError(err)
}

// Update updates an entity
func (r *Repository) Update(ctx context.Context, entity interface{}) error {
	_, err := r.db.NewUpdate().Model(entity).WherePK().Exec(ctx)
	return convertBunError(err)
}

// UpdatePartial updates specific fields of an entity
func (r *Repository) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	// Create a new instance of the entity type
	entity := reflect.New(r.entityType).Interface()

	query := r.db.NewUpdate().Model(entity).Where("id = ?", id)

	// Apply updates one by one
	for key, value := range updates {
		query = query.Set("? = ?", bun.Ident(key), value)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return convertBunError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return convertBunError(err)
	}

	if rowsAffected == 0 {
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
	result, err := r.db.NewDelete().Model(entity).Where("id = ?", id).Exec(ctx)

	if err != nil {
		return convertBunError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return convertBunError(err)
	}

	if rowsAffected == 0 {
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
	query := r.db.NewDelete().Model(entity)
	query = r.applyConditionToDelete(query, condition)

	_, err := query.Exec(ctx)
	return convertBunError(err)
}

// Query executes a query with the given options
func (r *Repository) Query(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildSelectQuery(dest, opts...)
	err := query.Scan(ctx)
	return convertBunError(err)
}

// QueryOne executes a query and returns a single result
func (r *Repository) QueryOne(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildSelectQuery(dest, opts...)
	err := query.Scan(ctx)
	return convertBunError(err)
}

// Count counts entities matching the given options
func (r *Repository) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.NewSelect().Model(entity)
	query = r.applyConditionsToSelect(query, opts...)

	count, err := query.Count(ctx)
	return int64(count), convertBunError(err)
}

// Exists checks if any entity matches the given options
func (r *Repository) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	return count > 0, err
}

// Transaction executes a function within a transaction
func (r *Repository) Transaction(ctx context.Context, fn gpa.TransactionFunc) error {
	// Get the underlying *bun.DB from the interface
	var bunDB *bun.DB
	switch db := r.db.(type) {
	case *bun.DB:
		bunDB = db
	case bun.Tx:
		// If we're already in a transaction, just execute the function
		txRepo := &Transaction{
			Repository: &Repository{
				db:         db,
				entityType: r.entityType,
				provider:   r.provider,
			},
		}
		return fn(txRepo)
	default:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeTransaction,
			Message: "unable to start transaction: invalid database type",
		}
	}

	return bunDB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRepo := &Transaction{
			Repository: &Repository{
				db:         tx, // bun.Tx implements bun.IDB
				entityType: r.entityType,
				provider:   r.provider,
			},
		}
		return fn(txRepo)
	})
}

// RawQuery executes a raw SQL query
func (r *Repository) RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error {
	err := r.db.NewRaw(query, args...).Scan(ctx, dest)
	return convertBunError(err)
}

// RawExec executes a raw SQL statement
func (r *Repository) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	result, err := r.db.NewRaw(query, args...).Exec(ctx)
	if err != nil {
		return nil, convertBunError(err)
	}

	lastInsertId, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return &Result{
		lastInsertId: lastInsertId,
		rowsAffected: rowsAffected,
	}, nil
}

// GetEntityInfo returns metadata about the entity
func (r *Repository) GetEntityInfo(entity interface{}) (*gpa.EntityInfo, error) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	info := &gpa.EntityInfo{
		Name:      t.Name(),
		TableName: getTableName(t),
		Fields:    make([]gpa.FieldInfo, 0, t.NumField()),
	}

	// Extract field information using reflection
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		bunTag := field.Tag.Get("bun")

		fieldInfo := gpa.FieldInfo{
			Name:            field.Name,
			Type:            field.Type,
			Tag:             string(field.Tag),
			IsNullable:      isNullableField(field),
			IsPrimaryKey:    isPrimaryKeyField(bunTag),
			IsAutoIncrement: isAutoIncrementField(bunTag),
		}

		// Extract database type and other info from bun tag
		fieldInfo.DatabaseType = extractDatabaseType(bunTag)

		info.Fields = append(info.Fields, fieldInfo)

		if fieldInfo.IsPrimaryKey {
			info.PrimaryKey = append(info.PrimaryKey, field.Name)
		}
	}

	return info, nil
}

// Close closes the repository (no-op for Bun)
func (r *Repository) Close() error {
	return nil
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
	_, err := r.db.NewCreateTable().Model(entity).IfNotExists().Exec(ctx)
	return convertBunError(err)
}

// DropTable drops the table for the entity
func (r *Repository) DropTable(ctx context.Context, entity interface{}) error {
	_, err := r.db.NewDropTable().Model(entity).IfExists().Exec(ctx)
	return convertBunError(err)
}


// CreateIndex creates an index on the specified fields
func (r *Repository) CreateIndex(ctx context.Context, entity interface{}, fields []string, unique bool) error {
	tableName := getTableName(reflect.TypeOf(entity))
	if tableName == "" {
		// Try to get table name from entity if it has TableName method
		if tn, ok := entity.(interface{ TableName() string }); ok {
			tableName = tn.TableName()
		} else {
			// Fallback to type name
			t := reflect.TypeOf(entity)
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			tableName = getTableName(t)
		}
	}

	indexName := fmt.Sprintf("idx_%s_%s", tableName, strings.Join(fields, "_"))

	query := r.db.NewCreateIndex().
		Table(tableName).
		Index(indexName).
		Column(fields...)

	if unique {
		query = query.Unique()
	}

	_, err := query.Exec(ctx)
	return convertBunError(err)
}

// DropIndex drops an index
func (r *Repository) DropIndex(ctx context.Context, entity interface{}, indexName string) error {
	_, err := r.db.NewDropIndex().Index(indexName).Exec(ctx)
	return convertBunError(err)
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
	// Bun handles commit automatically when the transaction function returns nil
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	// Bun handles rollback automatically when the transaction function returns an error
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

// buildSelectQuery builds a Bun select query from GPA query options
func (r *Repository) buildSelectQuery(dest interface{}, opts ...gpa.QueryOption) *bun.SelectQuery {
	query := &gpa.Query{}

	// Apply all options
	for _, opt := range opts {
		opt.Apply(query)
	}

	selectQuery := r.db.NewSelect().Model(dest)

	// Apply conditions
	selectQuery = r.applyConditionsToSelect(selectQuery, opts...)

	// Apply field selection
	if len(query.Fields) > 0 {
		selectQuery = selectQuery.Column(query.Fields...)
	}

	// Apply ordering
	for _, order := range query.Orders {
		selectQuery = selectQuery.Order(fmt.Sprintf("%s %s", order.Field, order.Direction))
	}

	// Apply limit
	if query.Limit != nil {
		selectQuery = selectQuery.Limit(*query.Limit)
	}

	// Apply offset
	if query.Offset != nil {
		selectQuery = selectQuery.Offset(*query.Offset)
	}

	// Apply joins
	for _, join := range query.Joins {
		joinClause := fmt.Sprintf("%s JOIN %s", join.Type, join.Table)
		if join.Alias != "" {
			joinClause += " AS " + join.Alias
		}
		selectQuery = selectQuery.Join(joinClause)
		if join.Condition != "" {
			selectQuery = selectQuery.JoinOn(join.Condition)
		}
	}

	// Apply grouping
	if len(query.Groups) > 0 {
		selectQuery = selectQuery.Group(query.Groups...)
	}

	// Apply having conditions
	for _, having := range query.Having {
		selectQuery = r.applyConditionToSelect(selectQuery, having)
	}

	// Apply distinct
	if query.Distinct {
		selectQuery = selectQuery.Distinct()
	}

	// Apply relation loading (preloads)
	for _, preload := range query.Preloads {
		selectQuery = selectQuery.Relation(preload)
	}

	return selectQuery
}

// applyConditionsToSelect applies conditions to a select query
func (r *Repository) applyConditionsToSelect(query *bun.SelectQuery, opts ...gpa.QueryOption) *bun.SelectQuery {
	gpaQuery := &gpa.Query{}
	for _, opt := range opts {
		opt.Apply(gpaQuery)
	}

	for _, condition := range gpaQuery.Conditions {
		query = r.applyConditionToSelect(query, condition)
	}

	return query
}

// applyConditionToSelect applies a condition to a select query
func (r *Repository) applyConditionToSelect(query *bun.SelectQuery, condition gpa.Condition) *bun.SelectQuery {
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		return r.applyBasicConditionToSelect(query, cond)
	case gpa.CompositeCondition:
		return r.applyCompositeConditionToSelect(query, cond)
	case gpa.SubQueryCondition:
		return r.applySubQueryConditionToSelect(query, cond)
	default:
		// Fallback to string representation
		return query.Where(condition.String(), condition.Value())
	}
}

// applyBasicConditionToSelect applies a basic condition to a select query
func (r *Repository) applyBasicConditionToSelect(query *bun.SelectQuery, condition gpa.BasicCondition) *bun.SelectQuery {
	field := condition.Field()
	op := condition.Operator()
	value := condition.Value()

	switch op {
	case gpa.OpEqual:
		return query.Where("? = ?", bun.Ident(field), value)
	case gpa.OpNotEqual:
		return query.Where("? != ?", bun.Ident(field), value)
	case gpa.OpGreaterThan:
		return query.Where("? > ?", bun.Ident(field), value)
	case gpa.OpGreaterThanOrEqual:
		return query.Where("? >= ?", bun.Ident(field), value)
	case gpa.OpLessThan:
		return query.Where("? < ?", bun.Ident(field), value)
	case gpa.OpLessThanOrEqual:
		return query.Where("? <= ?", bun.Ident(field), value)
	case gpa.OpLike:
		return query.Where("? LIKE ?", bun.Ident(field), value)
	case gpa.OpNotLike:
		return query.Where("? NOT LIKE ?", bun.Ident(field), value)
	case gpa.OpIn:
		return query.Where("? IN (?)", bun.Ident(field), bun.In(value))
	case gpa.OpNotIn:
		return query.Where("? NOT IN (?)", bun.Ident(field), bun.In(value))
	case gpa.OpIsNull:
		return query.Where("? IS NULL", bun.Ident(field))
	case gpa.OpIsNotNull:
		return query.Where("? IS NOT NULL", bun.Ident(field))
	case gpa.OpBetween:
		if values, ok := value.([]interface{}); ok && len(values) == 2 {
			return query.Where("? BETWEEN ? AND ?", bun.Ident(field), values[0], values[1])
		}
		return query
	case gpa.OpExists:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			args := []interface{}{bun.Safe(subQuery.Query)}
			args = append(args, subQuery.Args...)
			return query.Where("EXISTS (?)", args...)
		}
		return query
	case gpa.OpNotExists:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			args := []interface{}{bun.Safe(subQuery.Query)}
			args = append(args, subQuery.Args...)
			return query.Where("NOT EXISTS (?)", args...)
		}
		return query
	case gpa.OpInSubQuery:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			args := []interface{}{bun.Ident(field), bun.Safe(subQuery.Query)}
			args = append(args, subQuery.Args...)
			return query.Where("? IN (?)", args...)
		}
		return query
	case gpa.OpNotInSubQuery:
		if subQuery, ok := value.(gpa.SubQuery); ok {
			args := []interface{}{bun.Ident(field), bun.Safe(subQuery.Query)}
			args = append(args, subQuery.Args...)
			return query.Where("? NOT IN (?)", args...)
		}
		return query
	default:
		// Fallback
		return query.Where("? ? ?", bun.Ident(field), bun.Safe(string(op)), value)
	}
}

// applyCompositeConditionToSelect applies a composite condition to a select query
func (r *Repository) applyCompositeConditionToSelect(query *bun.SelectQuery, condition gpa.CompositeCondition) *bun.SelectQuery {
	if len(condition.Conditions) == 0 {
		return query
	}

	// For composite conditions, we need to build a single WHERE clause
	var parts []string
	var values []interface{}

	for _, subCondition := range condition.Conditions {
		part, vals := r.buildConditionPart(subCondition)
		if part != "" {
			parts = append(parts, part)
			values = append(values, vals...)
		}
	}

	if len(parts) == 0 {
		return query
	}

	whereClause := strings.Join(parts, fmt.Sprintf(" %s ", condition.Logic))
	return query.Where(whereClause, values...)
}

// buildConditionPart builds a condition part for composite conditions
func (r *Repository) buildConditionPart(condition gpa.Condition) (string, []interface{}) {
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		field := cond.Field()
		op := cond.Operator()
		value := cond.Value()

		switch op {
		case gpa.OpIsNull:
			return fmt.Sprintf("%s IS NULL", field), nil
		case gpa.OpIsNotNull:
			return fmt.Sprintf("%s IS NOT NULL", field), nil
		case gpa.OpBetween:
			if values, ok := value.([]interface{}); ok && len(values) == 2 {
				return fmt.Sprintf("%s BETWEEN ? AND ?", field), values
			}
			return "", nil
		default:
			return fmt.Sprintf("%s %s ?", field, op), []interface{}{value}
		}
	case gpa.CompositeCondition:
		// For nested composite conditions
		var parts []string
		var values []interface{}

		for _, subCond := range cond.Conditions {
			part, vals := r.buildConditionPart(subCond)
			if part != "" {
				parts = append(parts, part)
				values = append(values, vals...)
			}
		}

		if len(parts) == 0 {
			return "", nil
		}

		return fmt.Sprintf("(%s)", strings.Join(parts, fmt.Sprintf(" %s ", cond.Logic))), values
	default:
		return condition.String(), []interface{}{condition.Value()}
	}
}

// applySubQueryConditionToSelect applies a subquery condition to a select query
func (r *Repository) applySubQueryConditionToSelect(query *bun.SelectQuery, condition gpa.SubQueryCondition) *bun.SelectQuery {
	subQuery := condition.SubQuery

	switch subQuery.Type {
	case gpa.SubQueryTypeExists:
		if subQuery.Operator == gpa.OpNotExists {
			return query.Where("NOT EXISTS ("+subQuery.Query+")", subQuery.Args...)
		}
		return query.Where("EXISTS ("+subQuery.Query+")", subQuery.Args...)

	case gpa.SubQueryTypeIn:
		if subQuery.Operator == gpa.OpNotInSubQuery {
			return query.Where("? NOT IN ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		}
		return query.Where("? IN ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)

	case gpa.SubQueryTypeCorrelated:
		// Correlated subqueries can use EXISTS or scalar operators
		switch subQuery.Operator {
		case gpa.OpExists:
			return query.Where("EXISTS ("+subQuery.Query+")", subQuery.Args...)
		case gpa.OpNotExists:
			return query.Where("NOT EXISTS ("+subQuery.Query+")", subQuery.Args...)
		case gpa.OpGreaterThan:
			return query.Where("? > ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThan:
			return query.Where("? < ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpEqual:
			return query.Where("? = ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpGreaterThanOrEqual:
			return query.Where("? >= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThanOrEqual:
			return query.Where("? <= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpNotEqual:
			return query.Where("? != ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		default:
			return query.Where(condition.String())
		}

	case gpa.SubQueryTypeScalar:
		// For scalar subqueries, use the operator directly
		switch subQuery.Operator {
		case gpa.OpGreaterThan:
			return query.Where("? > ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThan:
			return query.Where("? < ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpEqual:
			return query.Where("? = ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpGreaterThanOrEqual:
			return query.Where("? >= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThanOrEqual:
			return query.Where("? <= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpNotEqual:
			return query.Where("? != ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		default:
			return query.Where(condition.String())
		}

	default:
		// Fallback to string representation
		return query.Where(condition.String())
	}
}

// applyConditionToDelete applies a condition to a delete query
func (r *Repository) applyConditionToDelete(query *bun.DeleteQuery, condition gpa.Condition) *bun.DeleteQuery {
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		return r.applyBasicConditionToDelete(query, cond)
	case gpa.CompositeCondition:
		return r.applyCompositeConditionToDelete(query, cond)
	case gpa.SubQueryCondition:
		return r.applySubQueryConditionToDelete(query, cond)
	default:
		return query.Where(condition.String(), condition.Value())
	}
}

// applyBasicConditionToDelete applies a basic condition to a delete query
func (r *Repository) applyBasicConditionToDelete(query *bun.DeleteQuery, condition gpa.BasicCondition) *bun.DeleteQuery {
	field := condition.Field()
	op := condition.Operator()
	value := condition.Value()

	switch op {
	case gpa.OpEqual:
		return query.Where("? = ?", bun.Ident(field), value)
	case gpa.OpNotEqual:
		return query.Where("? != ?", bun.Ident(field), value)
	case gpa.OpIn:
		return query.Where("? IN (?)", bun.Ident(field), bun.In(value))
	case gpa.OpNotIn:
		return query.Where("? NOT IN (?)", bun.Ident(field), bun.In(value))
	case gpa.OpIsNull:
		return query.Where("? IS NULL", bun.Ident(field))
	case gpa.OpIsNotNull:
		return query.Where("? IS NOT NULL", bun.Ident(field))
	default:
		return query.Where("? ? ?", bun.Ident(field), bun.Safe(string(op)), value)
	}
}

// applyCompositeConditionToDelete applies a composite condition to a delete query
func (r *Repository) applyCompositeConditionToDelete(query *bun.DeleteQuery, condition gpa.CompositeCondition) *bun.DeleteQuery {
	if len(condition.Conditions) == 0 {
		return query
	}

	var parts []string
	var values []interface{}

	for _, subCondition := range condition.Conditions {
		part, vals := r.buildConditionPart(subCondition)
		if part != "" {
			parts = append(parts, part)
			values = append(values, vals...)
		}
	}

	if len(parts) == 0 {
		return query
	}

	whereClause := strings.Join(parts, fmt.Sprintf(" %s ", condition.Logic))
	return query.Where(whereClause, values...)
}

// =====================================
// Connection Helpers
// =====================================

// createPostgresConnection creates a PostgreSQL connection
func createPostgresConnection(config gpa.Config) (*sql.DB, error) {
	if config.ConnectionURL != "" {
		return sql.Open("postgres", config.ConnectionURL)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	if config.SSL.Enabled {
		dsn = strings.Replace(dsn, "sslmode=disable", "sslmode="+config.SSL.Mode, 1)
	}

	return sql.Open("postgres", dsn)
}

// createMySQLConnection creates a MySQL connection
func createMySQLConnection(config gpa.Config) (*sql.DB, error) {
	if config.ConnectionURL != "" {
		return sql.Open("mysql", config.ConnectionURL)
	}

	mysqlConfig := mysql.Config{
		User:   config.Username,
		Passwd: config.Password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName: config.Database,
	}

	return sql.Open("mysql", mysqlConfig.FormatDSN())
}

// createSQLiteConnection creates a SQLite connection
func createSQLiteConnection(config gpa.Config) (*sql.DB, error) {
	return sql.Open("sqlite3", config.Database)
}

// =====================================
// Helper Functions
// =====================================

// getTableName extracts table name from struct type
func getTableName(t reflect.Type) string {
	// Check if type implements a TableName method
	if t.Implements(reflect.TypeOf((*interface{ TableName() string })(nil)).Elem()) {
		// This would need to be handled differently since we can't call methods on types
		// For now, use the struct name
	}

	// Convert struct name to snake_case
	return toSnakeCase(t.Name())
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isNullableField checks if a field is nullable
func isNullableField(field reflect.StructField) bool {
	bunTag := field.Tag.Get("bun")
	return strings.Contains(bunTag, "nullzero") ||
		field.Type.Kind() == reflect.Ptr ||
		(field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8) // []byte
}

// isPrimaryKeyField checks if a field is a primary key
func isPrimaryKeyField(bunTag string) bool {
	return strings.Contains(bunTag, ",pk") || strings.Contains(bunTag, "pk,") || bunTag == "pk"
}

// isAutoIncrementField checks if a field is auto-increment
func isAutoIncrementField(bunTag string) bool {
	return strings.Contains(bunTag, "autoincrement") || strings.Contains(bunTag, "identity")
}

// extractDatabaseType extracts database type from bun tag
func extractDatabaseType(bunTag string) string {
	parts := strings.Split(bunTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "type:") {
			return strings.TrimPrefix(part, "type:")
		}
	}
	return ""
}

// =====================================
// Error Conversion
// =====================================

// convertBunError converts Bun errors to GPA errors
func convertBunError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case err == sql.ErrNoRows:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "record not found",
			Cause:   err,
		}
	case strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique"):
		return gpa.GPAError{
			Type:    gpa.ErrorTypeDuplicate,
			Message: "duplicate key violation",
			Cause:   err,
		}
	case strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "constraint"):
		return gpa.GPAError{
			Type:    gpa.ErrorTypeConstraint,
			Message: "constraint violation",
			Cause:   err,
		}
	case strings.Contains(err.Error(), "timeout"):
		return gpa.GPAError{
			Type:    gpa.ErrorTypeTimeout,
			Message: "operation timeout",
			Cause:   err,
		}
	case strings.Contains(err.Error(), "connection"):
		return gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "connection error",
			Cause:   err,
		}
	default:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "database operation failed",
			Cause:   err,
		}
	}
}

// =====================================
// Relationship Management
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
	query := r.db.NewSelect().Model(dest).Where("id = ?", id)

	// Apply preloads (relations in Bun)
	for _, relation := range relations {
		query = query.Relation(relation)
	}

	err := query.Scan(ctx)
	return convertBunError(err)
}

// =====================================
// Advanced Query Features
// =====================================

// FindWithPagination finds entities with pagination support
func (r *Repository) FindWithPagination(ctx context.Context, dest interface{}, page, pageSize int, opts ...gpa.QueryOption) (int64, error) {
	// First get the total count
	totalCount, err := r.Count(ctx, opts...)
	if err != nil {
		return 0, err
	}

	// Add pagination to options
	allOpts := make([]gpa.QueryOption, 0, len(opts)+2)
	allOpts = append(allOpts, opts...)
	allOpts = append(allOpts, gpa.Limit(pageSize))
	allOpts = append(allOpts, gpa.Offset((page-1)*pageSize))

	// Execute the query
	err = r.Query(ctx, dest, allOpts...)
	if err != nil {
		return 0, err
	}

	return totalCount, nil
}

// BulkInsert performs bulk insert with better performance
func (r *Repository) BulkInsert(ctx context.Context, entities interface{}, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	_, err := r.db.NewInsert().Model(entities).Exec(ctx)
	return convertBunError(err)
}

// BulkUpdate performs bulk update operations
func (r *Repository) BulkUpdate(ctx context.Context, updates map[string]interface{}, condition gpa.Condition) (int64, error) {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.NewUpdate().Model(entity)

	// Apply the condition
	query = r.applyConditionToUpdate(query, condition)

	// Apply updates
	for key, value := range updates {
		query = query.Set("? = ?", bun.Ident(key), value)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return 0, convertBunError(err)
	}

	rowsAffected, err := result.RowsAffected()
	return rowsAffected, convertBunError(err)
}

// applyConditionToUpdate applies a condition to an update query
func (r *Repository) applyConditionToUpdate(query *bun.UpdateQuery, condition gpa.Condition) *bun.UpdateQuery {
	switch cond := condition.(type) {
	case gpa.SubQueryCondition:
		return r.applySubQueryConditionToUpdate(query, cond)
	case gpa.BasicCondition:
		field := cond.Field()
		op := cond.Operator()
		value := cond.Value()

		switch op {
		case gpa.OpEqual:
			return query.Where("? = ?", bun.Ident(field), value)
		case gpa.OpNotEqual:
			return query.Where("? != ?", bun.Ident(field), value)
		case gpa.OpIn:
			return query.Where("? IN (?)", bun.Ident(field), bun.In(value))
		case gpa.OpNotIn:
			return query.Where("? NOT IN (?)", bun.Ident(field), bun.In(value))
		case gpa.OpIsNull:
			return query.Where("? IS NULL", bun.Ident(field))
		case gpa.OpIsNotNull:
			return query.Where("? IS NOT NULL", bun.Ident(field))
		default:
			return query.Where("? ? ?", bun.Ident(field), bun.Safe(string(op)), value)
		}
	case gpa.CompositeCondition:
		// For composite conditions in updates
		var parts []string
		var values []interface{}

		for _, subCondition := range cond.Conditions {
			part, vals := r.buildConditionPart(subCondition)
			if part != "" {
				parts = append(parts, part)
				values = append(values, vals...)
			}
		}

		if len(parts) > 0 {
			whereClause := strings.Join(parts, fmt.Sprintf(" %s ", cond.Logic))
			return query.Where(whereClause, values...)
		}
		return query
	default:
		return query.Where(condition.String(), condition.Value())
	}
}

// applySubQueryConditionToDelete applies a subquery condition to a delete query
func (r *Repository) applySubQueryConditionToDelete(query *bun.DeleteQuery, condition gpa.SubQueryCondition) *bun.DeleteQuery {
	subQuery := condition.SubQuery

	switch subQuery.Type {
	case gpa.SubQueryTypeExists:
		if subQuery.Operator == gpa.OpNotExists {
			return query.Where("NOT EXISTS ("+subQuery.Query+")", subQuery.Args...)
		}
		return query.Where("EXISTS ("+subQuery.Query+")", subQuery.Args...)

	case gpa.SubQueryTypeIn:
		if subQuery.Operator == gpa.OpNotInSubQuery {
			return query.Where("? NOT IN ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		}
		return query.Where("? IN ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)

	case gpa.SubQueryTypeCorrelated:
		switch subQuery.Operator {
		case gpa.OpExists:
			return query.Where("EXISTS ("+subQuery.Query+")", subQuery.Args...)
		case gpa.OpNotExists:
			return query.Where("NOT EXISTS ("+subQuery.Query+")", subQuery.Args...)
		case gpa.OpGreaterThan:
			return query.Where("? > ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThan:
			return query.Where("? < ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpEqual:
			return query.Where("? = ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpGreaterThanOrEqual:
			return query.Where("? >= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThanOrEqual:
			return query.Where("? <= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpNotEqual:
			return query.Where("? != ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		default:
			return query.Where(condition.String())
		}

	case gpa.SubQueryTypeScalar:
		switch subQuery.Operator {
		case gpa.OpGreaterThan:
			return query.Where("? > ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThan:
			return query.Where("? < ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpEqual:
			return query.Where("? = ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpGreaterThanOrEqual:
			return query.Where("? >= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThanOrEqual:
			return query.Where("? <= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpNotEqual:
			return query.Where("? != ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		default:
			return query.Where(condition.String())
		}

	default:
		return query.Where(condition.String())
	}
}

// applySubQueryConditionToUpdate applies a subquery condition to an update query
func (r *Repository) applySubQueryConditionToUpdate(query *bun.UpdateQuery, condition gpa.SubQueryCondition) *bun.UpdateQuery {
	subQuery := condition.SubQuery

	switch subQuery.Type {
	case gpa.SubQueryTypeExists:
		if subQuery.Operator == gpa.OpNotExists {
			return query.Where("NOT EXISTS ("+subQuery.Query+")", subQuery.Args...)
		}
		return query.Where("EXISTS ("+subQuery.Query+")", subQuery.Args...)

	case gpa.SubQueryTypeIn:
		if subQuery.Operator == gpa.OpNotInSubQuery {
			return query.Where("? NOT IN ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		}
		return query.Where("? IN ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)

	case gpa.SubQueryTypeCorrelated:
		switch subQuery.Operator {
		case gpa.OpExists:
			return query.Where("EXISTS ("+subQuery.Query+")", subQuery.Args...)
		case gpa.OpNotExists:
			return query.Where("NOT EXISTS ("+subQuery.Query+")", subQuery.Args...)
		case gpa.OpGreaterThan:
			return query.Where("? > ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThan:
			return query.Where("? < ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpEqual:
			return query.Where("? = ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpGreaterThanOrEqual:
			return query.Where("? >= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThanOrEqual:
			return query.Where("? <= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpNotEqual:
			return query.Where("? != ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		default:
			return query.Where(condition.String())
		}

	case gpa.SubQueryTypeScalar:
		switch subQuery.Operator {
		case gpa.OpGreaterThan:
			return query.Where("? > ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThan:
			return query.Where("? < ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpEqual:
			return query.Where("? = ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpGreaterThanOrEqual:
			return query.Where("? >= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpLessThanOrEqual:
			return query.Where("? <= ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		case gpa.OpNotEqual:
			return query.Where("? != ("+subQuery.Query+")", append([]interface{}{bun.Ident(subQuery.Field)}, subQuery.Args...)...)
		default:
			return query.Where(condition.String())
		}

	default:
		return query.Where(condition.String())
	}
}

// =====================================
// Full-Text Search Support
// =====================================

// FullTextSearch performs full-text search (PostgreSQL specific)
func (r *Repository) FullTextSearch(ctx context.Context, dest interface{}, searchTerm string, fields []string, opts ...gpa.QueryOption) error {
	query := r.buildSelectQuery(dest, opts...)

	// Build the full-text search query
	// This is PostgreSQL specific - would need adaptation for other databases
	searchQuery := fmt.Sprintf("to_tsvector('english', %s) @@ plainto_tsquery('english', ?)",
		strings.Join(fields, " || ' ' || "))

	query = query.Where(searchQuery, searchTerm)

	err := query.Scan(ctx)
	return convertBunError(err)
}

// =====================================
// JSON Operations (PostgreSQL/MySQL)
// =====================================

// QueryJSON queries JSON fields
func (r *Repository) QueryJSON(ctx context.Context, dest interface{}, jsonField string, jsonPath string, value interface{}, opts ...gpa.QueryOption) error {
	query := r.buildSelectQuery(dest, opts...)

	// This is database-specific - PostgreSQL example
	query = query.Where("?->? = ?", bun.Ident(jsonField), jsonPath, value)

	err := query.Scan(ctx)
	return convertBunError(err)
}

// UpdateJSON updates JSON fields
func (r *Repository) UpdateJSON(ctx context.Context, id interface{}, jsonField string, jsonPath string, value interface{}) error {
	entity := reflect.New(r.entityType).Interface()

	// PostgreSQL JSONB update syntax
	_, err := r.db.NewUpdate().
		Model(entity).
		Set("? = jsonb_set(?, ?, ?)", bun.Ident(jsonField), bun.Ident(jsonField), jsonPath, value).
		Where("id = ?", id).
		Exec(ctx)

	return convertBunError(err)
}

// =====================================
// Aggregation Operations
// =====================================

// Aggregate performs aggregation operations
func (r *Repository) Aggregate(ctx context.Context, result interface{}, aggregateFunc string, field string, opts ...gpa.QueryOption) error {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.NewSelect().Model(entity)

	// Apply conditions
	query = r.applyConditionsToSelect(query, opts...)

	// Apply the aggregation
	query = query.ColumnExpr(fmt.Sprintf("%s(?) as result", aggregateFunc), bun.Ident(field))

	err := query.Scan(ctx, result)
	return convertBunError(err)
}

// GroupBy performs group by operations with aggregation
func (r *Repository) GroupBy(ctx context.Context, dest interface{}, groupFields []string, aggregations map[string]string, opts ...gpa.QueryOption) error {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.NewSelect().Model(entity)

	// Apply conditions
	query = r.applyConditionsToSelect(query, opts...)

	// Add group by fields
	for _, field := range groupFields {
		query = query.Column(field)
	}

	// Add aggregations
	for alias, aggExpr := range aggregations {
		query = query.ColumnExpr(fmt.Sprintf("%s as %s", aggExpr, alias))
	}

	// Apply group by
	query = query.Group(groupFields...)

	err := query.Scan(ctx, dest)
	return convertBunError(err)
}

// =====================================
// Soft Delete Support
// =====================================

// SoftDelete marks an entity as deleted without removing it
func (r *Repository) SoftDelete(ctx context.Context, id interface{}) error {
	entity := reflect.New(r.entityType).Interface()

	result, err := r.db.NewUpdate().
		Model(entity).
		Set("deleted_at = ?", time.Now()).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)

	if err != nil {
		return convertBunError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return convertBunError(err)
	}

	if rowsAffected == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found or already deleted",
		}
	}

	return nil
}

// FindWithDeleted finds entities including soft-deleted ones
func (r *Repository) FindWithDeleted(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	query := r.buildSelectQuery(dest, opts...)
	// Remove the default deleted_at IS NULL condition if it exists
	err := query.Scan(ctx)
	return convertBunError(err)
}

// Restore restores a soft-deleted entity
func (r *Repository) Restore(ctx context.Context, id interface{}) error {
	entity := reflect.New(r.entityType).Interface()

	result, err := r.db.NewUpdate().
		Model(entity).
		Set("deleted_at = NULL").
		Where("id = ?", id).
		Where("deleted_at IS NOT NULL").
		Exec(ctx)

	if err != nil {
		return convertBunError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return convertBunError(err)
	}

	if rowsAffected == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found or not deleted",
		}
	}

	return nil
}

// =====================================
// Batch Operations
// =====================================

// BatchCreate creates multiple entities in optimized batches
func (r *Repository) BatchCreate(ctx context.Context, entities interface{}, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	// Use Bun's bulk insert with ON CONFLICT handling
	_, err := r.db.NewInsert().
		Model(entities).
		On("CONFLICT DO NOTHING"). // PostgreSQL syntax
		Exec(ctx)

	return convertBunError(err)
}

// BatchUpdate updates multiple entities in a single query
func (r *Repository) BatchUpdate(ctx context.Context, entities interface{}) error {
	// This would require building a complex VALUES clause
	// For now, iterate through entities
	v := reflect.ValueOf(entities)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "entities must be a slice",
		}
	}

	// Use transaction for batch updates
	return r.Transaction(ctx, func(tx gpa.Transaction) error {
		for i := 0; i < v.Len(); i++ {
			entity := v.Index(i).Interface()
			if err := tx.Update(ctx, entity); err != nil {
				return err
			}
		}
		return nil
	})
}

// =====================================
// Upsert Operations
// =====================================

// Upsert performs insert or update operation
func (r *Repository) Upsert(ctx context.Context, entity interface{}, conflictColumns []string, updateColumns []string) error {
	query := r.db.NewInsert().Model(entity)

	// PostgreSQL UPSERT syntax
	if len(conflictColumns) > 0 {
		onConflict := fmt.Sprintf("(%s)", strings.Join(conflictColumns, ", "))
		query = query.On(fmt.Sprintf("CONFLICT %s DO UPDATE", onConflict))

		if len(updateColumns) > 0 {
			updateSet := make([]string, len(updateColumns))
			for i, col := range updateColumns {
				updateSet[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
			}
			query = query.Set(strings.Join(updateSet, ", "))
		} else {
			query = query.Set("updated_at = EXCLUDED.updated_at")
		}
	}

	_, err := query.Exec(ctx)
	return convertBunError(err)
}

// =====================================
// Statistics and Monitoring
// =====================================

// GetTableStats returns table statistics
func (r *Repository) GetTableStats(ctx context.Context) (map[string]interface{}, error) {
	tableName := getTableName(r.entityType)

	var stats struct {
		TableName    string     `bun:"table_name"`
		RowCount     int64      `bun:"row_count"`
		TableSize    string     `bun:"table_size"`
		IndexSize    string     `bun:"index_size"`
		TotalSize    string     `bun:"total_size"`
		LastAnalyzed *time.Time `bun:"last_analyzed"`
	}

	// PostgreSQL specific query
	err := r.db.NewRaw(`
		SELECT
			schemaname||'.'||tablename as table_name,
			n_tup_ins - n_tup_del as row_count,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as table_size,
			pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as index_size,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
			last_analyze as last_analyzed
		FROM pg_stat_user_tables
		WHERE tablename = ?
	`, tableName).Scan(ctx, &stats)

	if err != nil {
		return nil, convertBunError(err)
	}

	result := map[string]interface{}{
		"table_name":    stats.TableName,
		"row_count":     stats.RowCount,
		"table_size":    stats.TableSize,
		"index_size":    stats.IndexSize,
		"total_size":    stats.TotalSize,
		"last_analyzed": stats.LastAnalyzed,
	}

	return result, nil
}

// =====================================
// Schema Information
// =====================================

// GetTableSchema returns detailed table schema information
func (r *Repository) GetTableSchema(ctx context.Context) (map[string]interface{}, error) {
	tableName := getTableName(r.entityType)

	var columns []struct {
		ColumnName    string  `bun:"column_name"`
		DataType      string  `bun:"data_type"`
		IsNullable    string  `bun:"is_nullable"`
		ColumnDefault *string `bun:"column_default"`
		MaxLength     *int    `bun:"character_maximum_length"`
	}

	// PostgreSQL specific query
	err := r.db.NewRaw(`
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length
		FROM information_schema.columns
		WHERE table_name = ?
		ORDER BY ordinal_position
	`, tableName).Scan(ctx, &columns)

	if err != nil {
		return nil, convertBunError(err)
	}

	result := map[string]interface{}{
		"table_name": tableName,
		"columns":    columns,
	}

	return result, nil
}

// =====================================
// Event Hooks Support
// =====================================

// WithHooks wraps the repository with event hooks
func (r *Repository) WithHooks(hooks gpa.EventHook) *HookedRepository {
	return &HookedRepository{
		Repository: r,
		hooks:      hooks,
	}
}

// HookedRepository wraps a repository with event hooks
type HookedRepository struct {
	*Repository
	hooks gpa.EventHook
}

// Create creates an entity with before/after hooks
func (hr *HookedRepository) Create(ctx context.Context, entity interface{}) error {
	if err := hr.hooks.BeforeCreate(ctx, entity); err != nil {
		return err
	}

	if err := hr.Repository.Create(ctx, entity); err != nil {
		return err
	}

	return hr.hooks.AfterCreate(ctx, entity)
}

// Update updates an entity with before/after hooks
func (hr *HookedRepository) Update(ctx context.Context, entity interface{}) error {
	if err := hr.hooks.BeforeUpdate(ctx, entity); err != nil {
		return err
	}

	if err := hr.Repository.Update(ctx, entity); err != nil {
		return err
	}

	return hr.hooks.AfterUpdate(ctx, entity)
}

// Delete deletes an entity with before/after hooks
func (hr *HookedRepository) Delete(ctx context.Context, id interface{}) error {
	// We need to fetch the entity first for the hooks
	entity := reflect.New(hr.entityType).Interface()
	if err := hr.Repository.FindByID(ctx, id, entity); err != nil {
		return err
	}

	if err := hr.hooks.BeforeDelete(ctx, entity); err != nil {
		return err
	}

	if err := hr.Repository.Delete(ctx, id); err != nil {
		return err
	}

	return hr.hooks.AfterDelete(ctx, entity)
}

// =====================================
// Registration and Initialization
// =====================================

// =====================================
// Missing Helper Functions
// =====================================

// createPgDriverConnection creates a PostgreSQL connection using pgdriver
func createPgDriverConnection(config gpa.Config) (*sql.DB, error) {
	dsn := buildPostgresDSN(config)
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	return sql.OpenDB(connector), nil
}

// buildPostgresDSN builds a PostgreSQL DSN string
func buildPostgresDSN(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	params := []string{}
	if config.SSL.Enabled {
		params = append(params, "sslmode="+config.SSL.Mode)
		if config.SSL.CertFile != "" {
			params = append(params, "sslcert="+config.SSL.CertFile)
		}
		if config.SSL.KeyFile != "" {
			params = append(params, "sslkey="+config.SSL.KeyFile)
		}
		if config.SSL.CAFile != "" {
			params = append(params, "sslrootcert="+config.SSL.CAFile)
		}
	} else {
		params = append(params, "sslmode=disable")
	}

	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn
}

// buildMySQLDSN builds a MySQL DSN string
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

// =====================================
// Additional Query Methods
// =====================================

// FindFirst finds the first entity matching the conditions
func (r *Repository) FindFirst(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	allOpts := make([]gpa.QueryOption, 0, len(opts)+1)
	allOpts = append(allOpts, opts...)
	allOpts = append(allOpts, gpa.Limit(1))

	return r.Query(ctx, dest, allOpts...)
}

// FindLast finds the last entity matching the conditions
func (r *Repository) FindLast(ctx context.Context, dest interface{}, orderField string, opts ...gpa.QueryOption) error {
	allOpts := make([]gpa.QueryOption, 0, len(opts)+2)
	allOpts = append(allOpts, opts...)
	allOpts = append(allOpts, gpa.OrderBy(orderField, gpa.OrderDesc))
	allOpts = append(allOpts, gpa.Limit(1))

	return r.Query(ctx, dest, allOpts...)
}

// CountWhere counts entities matching a specific condition
func (r *Repository) CountWhere(ctx context.Context, condition gpa.Condition) (int64, error) {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.NewSelect().Model(entity)
	query = r.applyConditionToSelect(query, condition)

	count, err := query.Count(ctx)
	return int64(count), convertBunError(err)
}

// ExistsWhere checks if any entity matches a specific condition
func (r *Repository) ExistsWhere(ctx context.Context, condition gpa.Condition) (bool, error) {
	count, err := r.CountWhere(ctx, condition)
	return count > 0, err
}

// =====================================
// Batch Operations Extensions
// =====================================

// BatchCreateWithResult creates multiple entities and returns the results
func (r *Repository) BatchCreateWithResult(ctx context.Context, entities interface{}) ([]interface{}, error) {
	v := reflect.ValueOf(entities)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "entities must be a slice",
		}
	}

	// Convert to slice of interfaces
	results := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		results[i] = v.Index(i).Interface()
	}

	_, err := r.db.NewInsert().Model(entities).Exec(ctx)
	if err != nil {
		return nil, convertBunError(err)
	}

	return results, nil
}

// BatchDeleteWhere deletes multiple entities matching a condition
func (r *Repository) BatchDeleteWhere(ctx context.Context, condition gpa.Condition) (int64, error) {
	entity := reflect.New(r.entityType).Interface()
	query := r.db.NewDelete().Model(entity)
	query = r.applyConditionToDelete(query, condition)

	result, err := query.Exec(ctx)
	if err != nil {
		return 0, convertBunError(err)
	}

	rowsAffected, err := result.RowsAffected()
	return rowsAffected, convertBunError(err)
}

// =====================================
// Advanced SQL Operations
// =====================================

// ExecWithResult executes a raw SQL statement and returns detailed results
func (r *Repository) ExecWithResult(ctx context.Context, sql string, args ...interface{}) (*DetailedResult, error) {
	result, err := r.db.NewRaw(sql, args...).Exec(ctx)
	if err != nil {
		return nil, convertBunError(err)
	}

	lastInsertId, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return &DetailedResult{
		LastInsertId: lastInsertId,
		RowsAffected: rowsAffected,
		SQL:          sql,
		Args:         args,
	}, nil
}

// DetailedResult provides detailed information about SQL execution results
type DetailedResult struct {
	LastInsertId int64
	RowsAffected int64
	SQL          string
	Args         []interface{}
}

// GetLastInsertId returns the last insert ID
func (r *DetailedResult) GetLastInsertId() int64 {
	return r.LastInsertId
}

// GetRowsAffected returns the number of affected rows
func (r *DetailedResult) GetRowsAffected() int64 {
	return r.RowsAffected
}

// GetSQL returns the executed SQL
func (r *DetailedResult) GetSQL() string {
	return r.SQL
}

// GetArgs returns the SQL arguments
func (r *DetailedResult) GetArgs() []interface{} {
	return r.Args
}

// =====================================
// Connection Pool Management
// =====================================

// GetConnectionStats returns connection pool statistics
func (r *Repository) GetConnectionStats(ctx context.Context) (map[string]interface{}, error) {
	// Get the underlying *bun.DB
	var bunDB *bun.DB
	switch db := r.db.(type) {
	case *bun.DB:
		bunDB = db
	default:
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "unable to get connection stats: invalid database type",
		}
	}

	sqlDB := bunDB.DB
	stats := sqlDB.Stats()

	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}, nil
}

// =====================================
// Query Debugging and Logging
// =====================================

// ExplainQuery returns the query execution plan
func (r *Repository) ExplainQuery(ctx context.Context, opts ...gpa.QueryOption) (string, error) {
	// For PostgreSQL, we can use EXPLAIN with the actual query
	if r.IsPostgreSQL() {
		entity := reflect.New(r.entityType).Interface()
		query := r.buildSelectQuery(entity, opts...)

		// Execute EXPLAIN directly
		var explanation []string
		err := r.db.NewRaw("EXPLAIN ?", query).Scan(ctx, &explanation)
		if err != nil {
			return "", convertBunError(err)
		}

		return strings.Join(explanation, "\n"), nil
	}

	// For other databases, return a simple message
	return "EXPLAIN not supported for this database type", nil
}

// GetQuerySQL returns the SQL that would be executed for a query
func (r *Repository) GetQuerySQL(opts ...gpa.QueryOption) (string, []interface{}, error) {
	entity := reflect.New(r.entityType).Interface()
	query := r.buildSelectQuery(entity, opts...)

	// Get the underlying *bun.DB for formatter
	var bunDB *bun.DB
	switch db := r.db.(type) {
	case *bun.DB:
		bunDB = db
	default:
		return "", nil, gpa.GPAError{
			Type:    gpa.ErrorTypeUnsupported,
			Message: "get query SQL not supported for this database type",
		}
	}

	// Use a buffer to capture the formatted query
	buf := make([]byte, 0, 1024)
	buf, err := query.AppendQuery(bunDB.Formatter(), buf)
	if err != nil {
		return "", nil, convertBunError(err)
	}

	// Since Bun formats the query with actual values, we return empty args
	// The SQL string will contain the actual values, not placeholders
	return string(buf), []interface{}{}, nil
}

// =====================================
// Transaction Isolation Levels
// =====================================

// TransactionWithIsolation executes a function within a transaction with specific isolation level
func (r *Repository) TransactionWithIsolation(ctx context.Context, isolationLevel sql.IsolationLevel, fn gpa.TransactionFunc) error {
	// Get the underlying *bun.DB
	var bunDB *bun.DB
	switch db := r.db.(type) {
	case *bun.DB:
		bunDB = db
	case bun.Tx:
		// If we're already in a transaction, just execute the function
		txRepo := &Transaction{
			Repository: &Repository{
				db:         db,
				entityType: r.entityType,
				provider:   r.provider,
			},
		}
		return fn(txRepo)
	default:
		return gpa.GPAError{
			Type:    gpa.ErrorTypeTransaction,
			Message: "unable to start transaction: invalid database type",
		}
	}

	txOpts := &sql.TxOptions{
		Isolation: isolationLevel,
	}

	return bunDB.RunInTx(ctx, txOpts, func(ctx context.Context, tx bun.Tx) error {
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

// =====================================
// Database-Specific Features
// =====================================

// PostgreSQL specific methods
func (r *Repository) PostgreSQLSpecific() *PostgreSQLRepository {
	return &PostgreSQLRepository{Repository: r}
}

// PostgreSQLRepository provides PostgreSQL-specific methods
type PostgreSQLRepository struct {
	*Repository
}

// Vacuum performs VACUUM operation on the table
func (pg *PostgreSQLRepository) Vacuum(ctx context.Context, analyze bool) error {
	tableName := getTableName(pg.entityType)

	sql := fmt.Sprintf("VACUUM %s", tableName)
	if analyze {
		sql = fmt.Sprintf("VACUUM ANALYZE %s", tableName)
	}

	_, err := pg.db.NewRaw(sql).Exec(ctx)
	return convertBunError(err)
}

// ReindexTable rebuilds indexes for the table
func (pg *PostgreSQLRepository) ReindexTable(ctx context.Context) error {
	tableName := getTableName(pg.entityType)
	sql := fmt.Sprintf("REINDEX TABLE %s", tableName)

	_, err := pg.db.NewRaw(sql).Exec(ctx)
	return convertBunError(err)
}

// GetTableSize returns the size of the table
func (pg *PostgreSQLRepository) GetTableSize(ctx context.Context) (int64, error) {
	tableName := getTableName(pg.entityType)

	var size int64
	err := pg.db.NewRaw("SELECT pg_total_relation_size(?)", tableName).Scan(ctx, &size)
	return size, convertBunError(err)
}

// =====================================
// Health Check and Monitoring
// =====================================

// HealthCheck performs a comprehensive health check
func (r *Repository) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	health := &HealthStatus{
		Timestamp: time.Now(),
		Status:    "healthy",
		Checks:    make(map[string]interface{}),
	}

	// Check database connection - get underlying sql.DB
	var sqlDB *sql.DB
	switch db := r.db.(type) {
	case *bun.DB:
		sqlDB = db.DB
	default:
		health.Status = "unhealthy"
		health.Checks["connection"] = map[string]interface{}{
			"status": "failed",
			"error":  "unable to get underlying sql.DB",
		}
		return health, nil
	}

	if err := sqlDB.Ping(); err != nil {
		health.Status = "unhealthy"
		health.Checks["connection"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		return health, nil
	}
	health.Checks["connection"] = map[string]interface{}{
		"status": "ok",
	}

	// Check table existence
	tableName := getTableName(r.entityType)
	var exists bool
	err := r.db.NewRaw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)", tableName).Scan(ctx, &exists)
	if err != nil || !exists {
		health.Status = "degraded"
		health.Checks["table"] = map[string]interface{}{
			"status": "failed",
			"table":  tableName,
			"error":  fmt.Sprintf("table %s does not exist", tableName),
		}
	} else {
		health.Checks["table"] = map[string]interface{}{
			"status": "ok",
			"table":  tableName,
		}
	}

	// Check connection pool stats
	if stats, err := r.GetConnectionStats(ctx); err == nil {
		health.Checks["connection_pool"] = stats
	}

	return health, nil
}

// HealthStatus represents the health status of the repository
type HealthStatus struct {
	Timestamp time.Time              `json:"timestamp"`
	Status    string                 `json:"status"` // healthy, degraded, unhealthy
	Checks    map[string]interface{} `json:"checks"`
}

// =====================================
// Utility Methods
// =====================================

// Truncate removes all data from the table
func (r *Repository) Truncate(ctx context.Context) error {
	tableName := getTableName(r.entityType)

	// Use different syntax based on database type
	var sql string
	switch r.db.(type) {
	default:
		sql = fmt.Sprintf("TRUNCATE TABLE %s", tableName)
	}

	_, err := r.db.NewRaw(sql).Exec(ctx)
	return convertBunError(err)
}

// GetTableName returns the table name for the entity
func (r *Repository) GetTableName() string {
	return getTableName(r.entityType)
}

// Clone creates a copy of the repository
func (r *Repository) Clone() *Repository {
	return &Repository{
		db:         r.db,
		entityType: r.entityType,
		provider:   r.provider,
	}
}

// WithDB returns a new repository with a different database connection
func (r *Repository) WithDB(db bun.IDB) *Repository {
	return &Repository{
		db:         db,
		entityType: r.entityType,
		provider:   r.provider,
	}
}

// =====================================
// Streaming and Cursor Support
// =====================================

// StreamQuery executes a query and streams results
func (r *Repository) StreamQuery(ctx context.Context, batchSize int, opts ...gpa.QueryOption) (<-chan interface{}, <-chan error) {
	resultChan := make(chan interface{}, batchSize)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		offset := 0
		for {
			// Create a slice to hold the batch
			batchType := reflect.SliceOf(r.entityType)
			batch := reflect.New(batchType).Interface()

			// Add pagination to options
			batchOpts := make([]gpa.QueryOption, 0, len(opts)+2)
			batchOpts = append(batchOpts, opts...)
			batchOpts = append(batchOpts, gpa.Limit(batchSize))
			batchOpts = append(batchOpts, gpa.Offset(offset))

			// Execute query
			if err := r.Query(ctx, batch, batchOpts...); err != nil {
				errorChan <- err
				return
			}

			// Send results
			batchValue := reflect.ValueOf(batch).Elem()
			if batchValue.Len() == 0 {
				break // No more results
			}

			for i := 0; i < batchValue.Len(); i++ {
				select {
				case resultChan <- batchValue.Index(i).Interface():
				case <-ctx.Done():
					errorChan <- ctx.Err()
					return
				}
			}

			// Check if we got fewer results than requested (end of data)
			if batchValue.Len() < batchSize {
				break
			}

			offset += batchSize
		}
	}()

	return resultChan, errorChan
}

// =====================================
// Database Driver Specific Implementations
// =====================================

// GetDialect returns the database dialect
func (r *Repository) GetDialect() string {
	switch db := r.db.(type) {
	case *bun.DB:
		dialectName := db.Dialect().Name()
		switch dialectName {
		case dialect.PG:
			return "postgres"
		case dialect.MySQL:
			return "mysql"
		case dialect.SQLite:
			return "sqlite"
		default:
			return "unknown"
		}
	}
	return "unknown"
}

// IsPostgreSQL checks if the database is PostgreSQL
func (r *Repository) IsPostgreSQL() bool {
	return r.GetDialect() == "postgres"
}

// IsMySQL checks if the database is MySQL
func (r *Repository) IsMySQL() bool {
	return r.GetDialect() == "mysql"
}

// IsSQLite checks if the database is SQLite
func (r *Repository) IsSQLite() bool {
	return r.GetDialect() == "sqlite"
}

// =====================================
// Final Registration
// =====================================

// init registers the Bun provider factory
func init() {
	gpa.RegisterProvider("bun", &Factory{})
}
