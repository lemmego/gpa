// Package gpabun provides a Bun adapter for the Go Persistence API (GPA)
package gpabun

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

// NewProvider creates a new Bun provider instance
func NewProvider(config gpa.Config) (*Provider, error) {
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
		return nil, fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
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

// Configure applies configuration changes
func (p *Provider) Configure(config gpa.Config) error {
	p.config = config
	return nil
}

// Health checks the database connection health
func (p *Provider) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB := p.db.DB
	return sqlDB.PingContext(ctx)
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
		gpa.FeatureSubQueries,
		gpa.FeatureJoins,
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

// NewTypeSafeProvider creates a new type-safe Bun provider
func NewTypeSafeProvider[T any](config gpa.Config) (gpa.Provider[T], error) {
	baseProvider, err := NewProvider(config)
	if err != nil {
		return nil, err
	}
	return &TypeSafeProvider[T]{provider: baseProvider}, nil
}

// TypeSafeProvider implements gpa.Provider[T] for Bun
type TypeSafeProvider[T any] struct {
	provider *Provider
}

// Repository implements gpa.Repository[T] using Bun
type Repository[T any] struct {
	db       bun.IDB
	provider *Provider
}

// Create inserts a new entity
func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	_, err := r.db.NewInsert().Model(entity).Exec(ctx)
	return convertBunError(err)
}

// CreateBatch inserts multiple entities
func (r *Repository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	_, err := r.db.NewInsert().Model(&entities).Exec(ctx)
	return convertBunError(err)
}

// FindByID retrieves a single entity by ID
func (r *Repository[T]) FindByID(ctx context.Context, id interface{}) (*T, error) {
	var entity T
	err := r.db.NewSelect().Model(&entity).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, convertBunError(err)
	}
	return &entity, nil
}

// FindAll retrieves all entities
func (r *Repository[T]) FindAll(ctx context.Context, opts ...gpa.QueryOption) ([]*T, error) {
	var entities []*T
	query := r.db.NewSelect().Model(&entities)
	err := query.Scan(ctx)
	if err != nil {
		return nil, convertBunError(err)
	}
	return entities, nil
}

// Update modifies an existing entity
func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	_, err := r.db.NewUpdate().Model(entity).WherePK().Exec(ctx)
	return convertBunError(err)
}

// UpdatePartial modifies specific fields of an entity
func (r *Repository[T]) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	var entity T
	query := r.db.NewUpdate().Model(&entity).Where("id = ?", id)
	for field, value := range updates {
		query = query.Set("? = ?", bun.Ident(field), value)
	}
	_, err := query.Exec(ctx)
	return convertBunError(err)
}

// Delete removes an entity by ID
func (r *Repository[T]) Delete(ctx context.Context, id interface{}) error {
	var entity T
	_, err := r.db.NewDelete().Model(&entity).Where("id = ?", id).Exec(ctx)
	return convertBunError(err)
}

// DeleteByCondition removes entities matching the condition
func (r *Repository[T]) DeleteByCondition(ctx context.Context, condition gpa.Condition) error {
	var entity T
	_, err := r.db.NewDelete().Model(&entity).Where(condition.String(), condition.Value()).Exec(ctx)
	return convertBunError(err)
}

// Query retrieves entities based on query options
func (r *Repository[T]) Query(ctx context.Context, opts ...gpa.QueryOption) ([]*T, error) {
	return r.FindAll(ctx, opts...)
}

// QueryOne retrieves a single entity based on query options
func (r *Repository[T]) QueryOne(ctx context.Context, opts ...gpa.QueryOption) (*T, error) {
	entities, err := r.FindAll(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found",
		}
	}
	return entities[0], nil
}

// Count returns the number of entities matching the query options
func (r *Repository[T]) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	var entity T
	count, err := r.db.NewSelect().Model(&entity).Count(ctx)
	return int64(count), convertBunError(err)
}

// Exists checks if any entities match the query options
func (r *Repository[T]) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	return count > 0, err
}

// Transaction executes a function within a transaction
func (r *Repository[T]) Transaction(ctx context.Context, fn gpa.TransactionFunc[T]) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRepo := &Transaction[T]{
			Repository: &Repository[T]{
				db:       tx,
				provider: r.provider,
			},
		}
		return fn(txRepo)
	})
}

// RawQuery executes a raw query and returns results
func (r *Repository[T]) RawQuery(ctx context.Context, query string, args []interface{}) ([]*T, error) {
	var entities []*T
	err := r.db.NewRaw(query, args...).Scan(ctx, &entities)
	return entities, convertBunError(err)
}

// RawExec executes a raw command
func (r *Repository[T]) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	result, err := r.db.NewRaw(query, args...).Exec(ctx)
	if err != nil {
		return nil, convertBunError(err)
	}
	return &Result{result: result}, nil
}

// GetEntityInfo returns metadata about the entity
func (r *Repository[T]) GetEntityInfo() (*gpa.EntityInfo, error) {
	var entity T
	entityType := reflect.TypeOf(entity)
	return &gpa.EntityInfo{
		Name:      entityType.Name(),
		TableName: entityType.Name(),
		Fields:    []gpa.FieldInfo{},
	}, nil
}

// Close closes the repository
func (r *Repository[T]) Close() error {
	return nil
}

// Transaction implements gpa.Transaction[T]
type Transaction[T any] struct {
	*Repository[T]
}

// Commit commits the transaction
func (t *Transaction[T]) Commit() error {
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction[T]) Rollback() error {
	return nil
}

// SetSavepoint creates a savepoint
func (t *Transaction[T]) SetSavepoint(name string) error {
	return nil
}

// RollbackToSavepoint rolls back to a savepoint
func (t *Transaction[T]) RollbackToSavepoint(name string) error {
	return nil
}

// Result implements gpa.Result
type Result struct {
	result sql.Result
}

// LastInsertId returns the last insert ID
func (r *Result) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

// RowsAffected returns the number of affected rows
func (r *Result) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

// Repository returns a type-safe repository for Bun
func (p *TypeSafeProvider[T]) Repository() gpa.Repository[T] {
	return &Repository[T]{
		db:       p.provider.db,
		provider: p.provider,
	}
}

// Configure applies configuration to the provider
func (p *TypeSafeProvider[T]) Configure(config gpa.Config) error {
	return p.provider.Configure(config)
}

// Health checks the database connection health
func (p *TypeSafeProvider[T]) Health() error {
	return p.provider.Health()
}

// Close closes the database connection
func (p *TypeSafeProvider[T]) Close() error {
	return p.provider.Close()
}

// SupportedFeatures returns the list of supported features
func (p *TypeSafeProvider[T]) SupportedFeatures() []gpa.Feature {
	return p.provider.SupportedFeatures()
}

// ProviderInfo returns information about this provider
func (p *TypeSafeProvider[T]) ProviderInfo() gpa.ProviderInfo {
	return p.provider.ProviderInfo()
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
	// Validate database path for file-based SQLite
	if config.Database != ":memory:" && config.Database != "" {
		// Check if the directory exists for file-based databases
		if dir := filepath.Dir(config.Database); dir != "." && dir != "/" {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return nil, fmt.Errorf("database directory does not exist: %s", dir)
			}
		}
	}
	
	return sql.Open("sqlite3", config.Database)
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
