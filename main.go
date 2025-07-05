// Package gpa provides a unified persistence API for Go applications
// supporting both SQL and NoSQL databases through adapter implementations.
package gpa

import (
	"context"
	"reflect"
	"time"
)

// =====================================
// Core Interfaces
// =====================================

// Repository is the main interface for database operations.
// This is the universal interface implemented by all database adapters (SQL, Document, KV, etc.)
// It provides CRUD operations, querying, and metadata access in a database-agnostic way.
type Repository interface {
	// ===============================
	// Basic CRUD Operations
	// ===============================
	
	// Create inserts a new entity into the database.
	// The entity must have an ID field that will be used as the primary key.
	// Returns error if the entity already exists or validation fails.
	Create(ctx context.Context, entity interface{}) error
	
	// CreateBatch inserts multiple entities in a single operation for better performance.
	// Expects a slice of entities. May not be atomic depending on the database.
	// Returns error if any entity fails to insert.
	CreateBatch(ctx context.Context, entities interface{}) error
	
	// FindByID retrieves a single entity by its primary key.
	// The dest parameter must be a pointer to the entity type.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	FindByID(ctx context.Context, id interface{}, dest interface{}) error
	
	// FindAll retrieves all entities of a type, optionally filtered by query options.
	// The dest parameter must be a pointer to a slice of the entity type.
	// Use QueryOptions like Where(), Limit(), OrderBy() to filter and sort results.
	FindAll(ctx context.Context, dest interface{}, opts ...QueryOption) error
	
	// Update modifies an existing entity. The entity must have an ID field.
	// Replaces the entire entity with the new values.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	Update(ctx context.Context, entity interface{}) error
	
	// UpdatePartial modifies specific fields of an entity without replacing the whole entity.
	// The updates map contains field names as keys and new values as values.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error
	
	// Delete removes an entity by its primary key.
	// Returns ErrorTypeNotFound if the entity doesn't exist.
	Delete(ctx context.Context, id interface{}) error
	
	// DeleteByCondition removes all entities matching the given condition.
	// Use Where() conditions to specify which entities to delete.
	// Returns the number of deleted entities through the database's specific mechanisms.
	DeleteByCondition(ctx context.Context, condition Condition) error

	// ===============================
	// Query Operations
	// ===============================
	
	// Query retrieves entities based on the provided query options.
	// More flexible than FindAll, supports complex conditions, joins, subqueries.
	// The dest parameter must be a pointer to a slice of the entity type.
	Query(ctx context.Context, dest interface{}, opts ...QueryOption) error
	
	// QueryOne retrieves a single entity based on query options.
	// Equivalent to Query() with Limit(1) but returns ErrorTypeNotFound if no match.
	// The dest parameter must be a pointer to the entity type.
	QueryOne(ctx context.Context, dest interface{}, opts ...QueryOption) error
	
	// Count returns the number of entities matching the query options.
	// Does not retrieve the actual entities, just counts them.
	// Useful for pagination and analytics.
	Count(ctx context.Context, opts ...QueryOption) (int64, error)
	
	// Exists checks if any entities match the query options.
	// More efficient than Count() when you only need to know if matches exist.
	// Returns true if at least one entity matches, false otherwise.
	Exists(ctx context.Context, opts ...QueryOption) (bool, error)

	// ===============================
	// Advanced Operations
	// ===============================
	
	// Transaction executes a function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// If the function completes successfully, the transaction is committed.
	// Not all databases support transactions (e.g., some NoSQL databases).
	Transaction(ctx context.Context, fn TransactionFunc) error
	
	// RawQuery executes a database-specific query and returns results.
	// The query string and args format depend on the database type:
	// - SQL: "SELECT * FROM users WHERE age > ?" with args [18]
	// - MongoDB: Pipeline stages with aggregation
	// - Redis: Redis commands
	RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error
	
	// RawExec executes a database-specific command that doesn't return data.
	// Used for database-specific operations like creating indexes, triggers, etc.
	// Returns a Result object with information about rows affected, etc.
	RawExec(ctx context.Context, query string, args []interface{}) (Result, error)

	// ===============================
	// Metadata Operations
	// ===============================
	
	// GetEntityInfo returns metadata about an entity type.
	// Includes field information, primary keys, indexes, and relationships.
	// Useful for reflection, validation, and building dynamic UIs.
	GetEntityInfo(entity interface{}) (*EntityInfo, error)
	
	// Close closes the repository and releases any resources.
	// Should be called when the repository is no longer needed.
	// May close database connections, file handles, etc.
	Close() error
}

// TransactionFunc represents a function that runs within a transaction
type TransactionFunc func(tx Transaction) error

// Transaction interface for transactional operations.
// Extends Repository with commit/rollback functionality for ACID operations.
// Not all databases support transactions (e.g., Redis, some NoSQL databases).
type Transaction interface {
	Repository
	
	// Commit permanently saves all changes made within this transaction.
	// Once committed, changes cannot be rolled back.
	// Returns error if the commit fails (e.g., constraint violations).
	Commit() error
	
	// Rollback discards all changes made within this transaction.
	// Returns the database to the state before the transaction began.
	// Returns error if the rollback fails (rare, usually indicates serious issues).
	Rollback() error
}

// Result represents the result of a database operation that doesn't return data.
// Provides information about what happened during INSERT, UPDATE, DELETE operations.
// Similar to sql.Result in the standard library but database-agnostic.
type Result interface {
	// LastInsertId returns the ID of the last inserted record.
	// Only meaningful for databases that auto-generate IDs (SQL with auto-increment).
	// May return 0 for databases that don't support this concept.
	LastInsertId() (int64, error)
	
	// RowsAffected returns the number of rows affected by the operation.
	// Useful for UPDATE and DELETE operations to see how many records changed.
	// For batch operations, returns the total number of affected rows.
	RowsAffected() (int64, error)
}

// =====================================
// Provider and Configuration
// =====================================

// Provider is the main interface for creating and managing database repositories.
// Acts as a factory for repositories and manages database connections and configuration.
// Each database adapter (GORM, Bun, MongoDB, Redis) implements this interface.
type Provider interface {
	// ===============================
	// Repository Creation
	// ===============================
	
	// Repository creates a new repository for the given entity type.
	// The entityType should be a reflect.Type representing your entity struct.
	// Example: provider.Repository(reflect.TypeOf(User{}))
	Repository(entityType reflect.Type) Repository
	
	// RepositoryFor creates a new repository for the given entity instance.
	// More convenient than Repository() when you have an entity instance.
	// Example: provider.RepositoryFor(&User{})
	RepositoryFor(entity interface{}) Repository

	// ===============================
	// Configuration and Lifecycle
	// ===============================
	
	// Configure applies new configuration to the provider.
	// Can be used to change connection settings, pool sizes, etc. at runtime.
	// May require reconnection depending on what settings changed.
	Configure(config Config) error
	
	// Health checks if the database connection is healthy and responsive.
	// Returns error if the database is unreachable or not functioning properly.
	// Useful for health check endpoints and monitoring.
	Health() error
	
	// Close shuts down the provider and releases all resources.
	// Closes database connections, stops background tasks, etc.
	// Should be called during application shutdown.
	Close() error

	// ===============================
	// Metadata and Capabilities
	// ===============================
	
	// SupportedFeatures returns a list of features this provider supports.
	// Features include things like transactions, full-text search, pub/sub, etc.
	// Use this to check capabilities before using advanced features.
	SupportedFeatures() []Feature
	
	// ProviderInfo returns metadata about this provider.
	// Includes provider name, version, database type, and supported features.
	// Useful for debugging, logging, and feature detection.
	ProviderInfo() ProviderInfo
}

// Config represents database configuration
type Config struct {
	// Connection details
	Driver        string `json:"driver" yaml:"driver"`
	ConnectionURL string `json:"connection_url" yaml:"connection_url"`
	Host          string `json:"host" yaml:"host"`
	Port          int    `json:"port" yaml:"port"`
	Database      string `json:"database" yaml:"database"`
	Username      string `json:"username" yaml:"username"`
	Password      string `json:"password" yaml:"password"`

	// Connection pool settings
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`

	// Additional options
	Options map[string]interface{} `json:"options" yaml:"options"`

	// SSL/TLS configuration
	SSL SSLConfig `json:"ssl" yaml:"ssl"`
}

// SSLConfig represents SSL/TLS configuration
type SSLConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Mode     string `json:"mode" yaml:"mode"`
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
	CAFile   string `json:"ca_file" yaml:"ca_file"`
}

// ProviderInfo contains information about the provider
type ProviderInfo struct {
	Name         string
	Version      string
	DatabaseType DatabaseType
	Features     []Feature
}

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypeSQL      DatabaseType = "sql"
	DatabaseTypeDocument DatabaseType = "document"
	DatabaseTypeKV       DatabaseType = "key-value"
	DatabaseTypeGraph    DatabaseType = "graph"
	DatabaseTypeMemory   DatabaseType = "memory"
)

// Feature represents a database feature
type Feature string

const (
	FeatureTransactions   Feature = "transactions"
	FeatureFullTextSearch Feature = "full_text_search"
	FeatureJSONQueries    Feature = "json_queries"
	FeatureGeospatial     Feature = "geospatial"
	FeaturePubSub         Feature = "pub_sub"
	FeatureStreaming      Feature = "streaming"
	FeatureSharding       Feature = "sharding"
	FeatureReplication    Feature = "replication"
	FeatureIndexing       Feature = "indexing"
	FeatureAggregation    Feature = "aggregation"
)

// =====================================
// Query Building and Conditions
// =====================================

// QueryOption represents options for queries
type QueryOption interface {
	Apply(query *Query)
}

// Query represents a database query
type Query struct {
	Conditions []Condition
	Orders     []Order
	Limit      *int
	Offset     *int
	Fields     []string
	Joins      []JoinClause
	Groups     []string
	Having     []Condition
	Distinct   bool
	Lock       LockType
	Preloads   []string
	SubQueries []SubQuery
}

// Condition represents a query condition
type Condition interface {
	Field() string
	Operator() Operator
	Value() interface{}
	String() string
}

// BasicCondition implements Condition
type BasicCondition struct {
	FieldName string
	Op        Operator
	Val       interface{}
}

func (c BasicCondition) Field() string      { return c.FieldName }
func (c BasicCondition) Operator() Operator { return c.Op }
func (c BasicCondition) Value() interface{} { return c.Val }
func (c BasicCondition) String() string {
	return c.FieldName + " " + string(c.Op) + " ?"
}

// CompositeCondition for AND/OR operations
type CompositeCondition struct {
	Conditions []Condition
	Logic      LogicOperator
}

func (c CompositeCondition) Field() string      { return "" }
func (c CompositeCondition) Operator() Operator { return "" }
func (c CompositeCondition) Value() interface{} { return nil }
func (c CompositeCondition) String() string {
	// Implementation would build composite condition string
	return ""
}

// Operator represents query operators
type Operator string

const (
	OpEqual              Operator = "="
	OpNotEqual           Operator = "!="
	OpGreaterThan        Operator = ">"
	OpGreaterThanOrEqual Operator = ">="
	OpLessThan           Operator = "<"
	OpLessThanOrEqual    Operator = "<="
	OpLike               Operator = "LIKE"
	OpNotLike            Operator = "NOT LIKE"
	OpIn                 Operator = "IN"
	OpNotIn              Operator = "NOT IN"
	OpIsNull             Operator = "IS NULL"
	OpIsNotNull          Operator = "IS NOT NULL"
	OpBetween            Operator = "BETWEEN"
	OpNotBetween         Operator = "NOT BETWEEN"
	OpContains           Operator = "CONTAINS"
	OpStartsWith         Operator = "STARTS_WITH"
	OpEndsWith           Operator = "ENDS_WITH"
	OpRegex              Operator = "REGEX"
	OpExists             Operator = "EXISTS"
	OpNotExists          Operator = "NOT EXISTS"
	OpInSubQuery         Operator = "IN_SUBQUERY"
	OpNotInSubQuery      Operator = "NOT_IN_SUBQUERY"
)

// LogicOperator represents logical operators
type LogicOperator string

const (
	LogicAnd LogicOperator = "AND"
	LogicOr  LogicOperator = "OR"
	LogicNot LogicOperator = "NOT"
)

// Order represents query ordering
type Order struct {
	Field     string
	Direction OrderDirection
}

// OrderDirection represents sort direction
type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// JoinClause represents a table join
type JoinClause struct {
	Type      JoinType
	Table     string
	Condition string
	Alias     string
}

// JoinType represents join types
type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinFull  JoinType = "FULL"
)

// LockType represents lock types for queries
type LockType string

const (
	LockNone      LockType = ""
	LockForUpdate LockType = "FOR UPDATE"
	LockForShare  LockType = "FOR SHARE"
)

// =====================================
// SubQuery Support
// =====================================

// SubQuery represents a subquery
type SubQuery struct {
	Query        string
	Args         []interface{}
	Type         SubQueryType
	Field        string // Field for subquery conditions
	Operator     Operator
	IsCorrelated bool
}

// SubQueryType represents the type of subquery
type SubQueryType string

const (
	SubQueryTypeExists     SubQueryType = "EXISTS"
	SubQueryTypeIn         SubQueryType = "IN"
	SubQueryTypeScalar     SubQueryType = "SCALAR"
	SubQueryTypeCorrelated SubQueryType = "CORRELATED"
)

// SubQueryCondition implements Condition for subqueries
type SubQueryCondition struct {
	SubQuery SubQuery
}

func (s SubQueryCondition) Field() string      { return s.SubQuery.Field }
func (s SubQueryCondition) Operator() Operator { return s.SubQuery.Operator }
func (s SubQueryCondition) Value() interface{} { return s.SubQuery }
func (s SubQueryCondition) String() string {
	switch s.SubQuery.Type {
	case SubQueryTypeExists:
		return "EXISTS (" + s.SubQuery.Query + ")"
	case SubQueryTypeIn:
		return s.SubQuery.Field + " IN (" + s.SubQuery.Query + ")"
	default:
		return s.SubQuery.Field + " " + string(s.SubQuery.Operator) + " (" + s.SubQuery.Query + ")"
	}
}

// =====================================
// Query Builder Functions
// =====================================

// Where creates a WHERE condition that can be used both as QueryOption and Condition
func Where(field string, operator Operator, value interface{}) ConditionOption {
	return ConditionOption{BasicCondition{FieldName: field, Op: operator, Val: value}}
}

// And creates an AND condition
func And(conditions ...Condition) Condition {
	return CompositeCondition{Conditions: conditions, Logic: LogicAnd}
}

// Or creates an OR condition
func Or(conditions ...Condition) Condition {
	return CompositeCondition{Conditions: conditions, Logic: LogicOr}
}

// WhereCondition creates a pure Condition (for use in And/Or)
func WhereCondition(field string, operator Operator, value interface{}) Condition {
	return BasicCondition{FieldName: field, Op: operator, Val: value}
}

// AndOption creates an AND condition that can be used as QueryOption
func AndOption(conditions ...Condition) QueryOption {
	return CompositeConditionOption{CompositeCondition{Conditions: conditions, Logic: LogicAnd}}
}

// OrOption creates an OR condition that can be used as QueryOption
func OrOption(conditions ...Condition) QueryOption {
	return CompositeConditionOption{CompositeCondition{Conditions: conditions, Logic: LogicOr}}
}

// OrderBy creates an ORDER BY clause
func OrderBy(field string, direction OrderDirection) QueryOption {
	return orderOption{Order{Field: field, Direction: direction}}
}

// Limit creates a LIMIT clause
func Limit(limit int) QueryOption {
	return limitOption{limit}
}

// Offset creates an OFFSET clause
func Offset(offset int) QueryOption {
	return offsetOption{offset}
}

// Select specifies which fields to select
func Select(fields ...string) QueryOption {
	return selectOption{fields}
}

// GroupBy creates a GROUP BY clause
func GroupBy(fields ...string) QueryOption {
	return groupByOption{fields}
}

// Having creates a HAVING clause
func Having(condition Condition) QueryOption {
	return havingOption{condition}
}

// Preload specifies which relationships to preload
func Preload(relations ...string) QueryOption {
	return preloadOption{relations}
}

// Join creates a JOIN clause
func Join(joinType JoinType, table string, condition string) QueryOption {
	return joinOption{join: JoinClause{Type: joinType, Table: table, Condition: condition}}
}

// JoinWithAlias creates a JOIN clause with table alias
func JoinWithAlias(joinType JoinType, table string, alias string, condition string) QueryOption {
	return joinOption{join: JoinClause{Type: joinType, Table: table, Alias: alias, Condition: condition}}
}

// =====================================
// SubQuery Builder Functions
// =====================================

// NewSubQuery creates a basic subquery
func NewSubQuery(query string, args ...interface{}) SubQuery {
	return SubQuery{
		Query: query,
		Args:  args,
		Type:  SubQueryTypeScalar,
	}
}

// ExistsSubQuery creates an EXISTS subquery condition
func ExistsSubQuery(query string, args ...interface{}) QueryOption {
	subQuery := SubQuery{
		Query:    query,
		Args:     args,
		Type:     SubQueryTypeExists,
		Operator: OpExists,
	}
	return subQueryOption{SubQueryCondition{SubQuery: subQuery}}
}

// NotExistsSubQuery creates a NOT EXISTS subquery condition
func NotExistsSubQuery(query string, args ...interface{}) QueryOption {
	subQuery := SubQuery{
		Query:    query,
		Args:     args,
		Type:     SubQueryTypeExists,
		Operator: OpNotExists,
	}
	return subQueryOption{SubQueryCondition{SubQuery: subQuery}}
}

// InSubQuery creates an IN subquery condition
func InSubQuery(field string, query string, args ...interface{}) QueryOption {
	subQuery := SubQuery{
		Query:    query,
		Args:     args,
		Type:     SubQueryTypeIn,
		Field:    field,
		Operator: OpInSubQuery,
	}
	return subQueryOption{SubQueryCondition{SubQuery: subQuery}}
}

// NotInSubQuery creates a NOT IN subquery condition
func NotInSubQuery(field string, query string, args ...interface{}) QueryOption {
	subQuery := SubQuery{
		Query:    query,
		Args:     args,
		Type:     SubQueryTypeIn,
		Field:    field,
		Operator: OpNotInSubQuery,
	}
	return subQueryOption{SubQueryCondition{SubQuery: subQuery}}
}

// WhereSubQuery creates a subquery condition with custom operator
func WhereSubQuery(field string, operator Operator, query string, args ...interface{}) QueryOption {
	subQuery := SubQuery{
		Query:    query,
		Args:     args,
		Type:     SubQueryTypeScalar,
		Field:    field,
		Operator: operator,
	}
	return subQueryOption{SubQueryCondition{SubQuery: subQuery}}
}

// CorrelatedSubQuery creates a correlated subquery
func CorrelatedSubQuery(field string, operator Operator, query string, args ...interface{}) QueryOption {
	subQuery := SubQuery{
		Query:        query,
		Args:         args,
		Type:         SubQueryTypeCorrelated,
		Field:        field,
		Operator:     operator,
		IsCorrelated: true,
	}
	return subQueryOption{SubQueryCondition{SubQuery: subQuery}}
}

// =====================================
// Query Option Implementations
// =====================================

// ConditionOption implements both QueryOption and Condition interfaces
type ConditionOption struct {
	BasicCondition
}

// Apply implements QueryOption
func (c ConditionOption) Apply(q *Query) { q.Conditions = append(q.Conditions, c.BasicCondition) }

// Field implements Condition
func (c ConditionOption) Field() string { return c.BasicCondition.Field() }

// Operator implements Condition
func (c ConditionOption) Operator() Operator { return c.BasicCondition.Operator() }

// Value implements Condition
func (c ConditionOption) Value() interface{} { return c.BasicCondition.Value() }

// String implements Condition
func (c ConditionOption) String() string { return c.BasicCondition.String() }

// CompositeConditionOption for complex conditions that can also be used as QueryOption
type CompositeConditionOption struct {
	CompositeCondition
}

// Apply implements QueryOption
func (c CompositeConditionOption) Apply(q *Query) {
	q.Conditions = append(q.Conditions, c.CompositeCondition)
}

type orderOption struct{ order Order }

func (o orderOption) Apply(q *Query) { q.Orders = append(q.Orders, o.order) }

type limitOption struct{ limit int }

func (l limitOption) Apply(q *Query) { q.Limit = &l.limit }

type offsetOption struct{ offset int }

func (o offsetOption) Apply(q *Query) { q.Offset = &o.offset }

type selectOption struct{ fields []string }

func (s selectOption) Apply(q *Query) { q.Fields = s.fields }

type groupByOption struct{ fields []string }

func (g groupByOption) Apply(q *Query) { q.Groups = g.fields }

type havingOption struct{ condition Condition }

func (h havingOption) Apply(q *Query) { q.Having = append(q.Having, h.condition) }

type preloadOption struct{ relations []string }

func (p preloadOption) Apply(q *Query) { q.Preloads = p.relations }

type joinOption struct{ join JoinClause }

func (j joinOption) Apply(q *Query) { q.Joins = append(q.Joins, j.join) }

type subQueryOption struct{ condition SubQueryCondition }

func (s subQueryOption) Apply(q *Query) {
	q.Conditions = append(q.Conditions, s.condition)
	q.SubQueries = append(q.SubQueries, s.condition.SubQuery)
}

// =====================================
// Entity Metadata
// =====================================

// EntityInfo contains metadata about an entity
type EntityInfo struct {
	Name       string
	TableName  string
	Fields     []FieldInfo
	PrimaryKey []string
	Indexes    []IndexInfo
	Relations  []RelationInfo
}

// FieldInfo contains metadata about a field
type FieldInfo struct {
	Name            string
	Type            reflect.Type
	DatabaseType    string
	Tag             string
	IsPrimaryKey    bool
	IsNullable      bool
	IsAutoIncrement bool
	DefaultValue    interface{}
	MaxLength       int
	Precision       int
	Scale           int
}

// IndexInfo contains metadata about an index
type IndexInfo struct {
	Name     string
	Fields   []string
	IsUnique bool
	Type     IndexType
}

// IndexType represents index types
type IndexType string

const (
	IndexTypeBTree    IndexType = "btree"
	IndexTypeHash     IndexType = "hash"
	IndexTypeGIN      IndexType = "gin"
	IndexTypeGiST     IndexType = "gist"
	IndexTypeFullText IndexType = "fulltext"
)

// RelationInfo contains metadata about relationships
type RelationInfo struct {
	Name         string
	Type         RelationType
	TargetEntity string
	ForeignKey   string
	References   string
}

// RelationType represents relationship types
type RelationType string

const (
	RelationOneToOne   RelationType = "one_to_one"
	RelationOneToMany  RelationType = "one_to_many"
	RelationManyToOne  RelationType = "many_to_one"
	RelationManyToMany RelationType = "many_to_many"
)

// =====================================
// Specialized Interfaces for Different Database Types
// =====================================

// SQLRepository extends Repository with SQL-specific operations.
// Implemented by SQL database adapters like GORM, Bun, etc.
// Provides raw SQL access, relationship loading, and schema management.
type SQLRepository interface {
	Repository

	// ===============================
	// Raw SQL Operations
	// ===============================
	
	// FindBySQL executes a raw SQL SELECT query and maps results to entities.
	// The dest parameter must be a pointer to a slice of the entity type.
	// Example: FindBySQL(ctx, "SELECT * FROM users WHERE age > ?", []interface{}{18}, &users)
	FindBySQL(ctx context.Context, sql string, args []interface{}, dest interface{}) error
	
	// ExecSQL executes a raw SQL command that doesn't return entities (INSERT, UPDATE, DELETE, DDL).
	// Returns a Result with information about rows affected, last insert ID, etc.
	// Example: ExecSQL(ctx, "UPDATE users SET status = ? WHERE active = ?", "inactive", false)
	ExecSQL(ctx context.Context, sql string, args ...interface{}) (Result, error)

	// ===============================
	// Relationship Operations (Convenience)
	// ===============================
	
	// FindWithRelations retrieves entities with their related entities preloaded.
	// More convenient than using Query() with Preload() options.
	// The relations slice specifies which relationships to load.
	// Example: FindWithRelations(ctx, &users, []string{"Posts", "Profile"}, Where("active", "=", true))
	FindWithRelations(ctx context.Context, dest interface{}, relations []string, opts ...QueryOption) error
	
	// FindByIDWithRelations retrieves a single entity by ID with relationships preloaded.
	// Combines FindByID() with relationship loading for convenience.
	// Example: FindByIDWithRelations(ctx, userID, &user, []string{"Posts", "Comments"})
	FindByIDWithRelations(ctx context.Context, id interface{}, dest interface{}, relations []string) error

	// ===============================
	// Schema Management
	// ===============================
	
	// CreateTable creates a new table based on the entity structure.
	// Analyzes the entity's fields, tags, and relationships to generate appropriate SQL.
	// May create foreign key constraints, indexes, and other database objects.
	CreateTable(ctx context.Context, entity interface{}) error
	
	// DropTable removes a table from the database.
	// WARNING: This permanently deletes all data in the table.
	// May fail if there are foreign key constraints pointing to this table.
	DropTable(ctx context.Context, entity interface{}) error

	// ===============================
	// Index Management
	// ===============================
	
	// CreateIndex creates a database index on the specified fields.
	// Improves query performance but may slow down write operations.
	// The unique parameter determines if the index enforces uniqueness.
	CreateIndex(ctx context.Context, entity interface{}, fields []string, unique bool) error
	
	// DropIndex removes an existing index from the database.
	// The indexName should match the name used when the index was created.
	// May improve write performance but will slow down relevant queries.
	DropIndex(ctx context.Context, entity interface{}, indexName string) error
}

// =====================================
// Optional SQL Extensions
// =====================================

// MigratableRepository is an optional interface for SQL databases that support schema migration.
// Currently only implemented by GORM adapter. Use type assertion to check availability.
// Example: if migrator, ok := repo.(MigratableRepository); ok { migrator.MigrateTable(...) }
type MigratableRepository interface {
	// MigrateTable updates an existing table to match the entity structure.
	// Adds missing columns, indexes, constraints, etc. Usually doesn't drop existing columns.
	// Use this for schema evolution during application updates.
	// Only available in adapters that support automatic schema migration (e.g., GORM).
	MigrateTable(ctx context.Context, entity interface{}) error
}

// DocumentRepository extends Repository with document database operations.
// Implemented by document store adapters like MongoDB, CouchDB, etc.
// Works with JSON-like documents, collections, and aggregation pipelines.
type DocumentRepository interface {
	Repository

	// ===============================
	// Document Operations
	// ===============================
	
	// FindByDocument finds documents that match the given document structure.
	// The document parameter acts as a query template - fields with values become filters.
	// Example: FindByDocument(ctx, map[string]interface{}{"status": "active", "age": map[string]interface{}{"$gt": 18}}, &users)
	FindByDocument(ctx context.Context, document map[string]interface{}, dest interface{}) error
	
	// UpdateDocument updates a document by merging the provided fields.
	// Only the fields present in the document map are updated; others remain unchanged.
	// Example: UpdateDocument(ctx, userID, map[string]interface{}{"lastLogin": time.Now(), "status": "online"})
	UpdateDocument(ctx context.Context, id interface{}, document map[string]interface{}) error

	// ===============================
	// Collection Management
	// ===============================
	
	// CreateCollection creates a new collection (equivalent to a table in SQL).
	// Some document databases automatically create collections when documents are inserted.
	// May accept options like capped collections, validation rules, etc.
	CreateCollection(ctx context.Context, name string) error
	
	// DropCollection removes a collection and all its documents.
	// WARNING: This permanently deletes all data in the collection.
	// Use with caution as this operation cannot be undone.
	DropCollection(ctx context.Context, name string) error
	
	// ListCollections returns the names of all collections in the database.
	// Useful for discovery, administration, and dynamic collection management.
	// May exclude system collections depending on the implementation.
	ListCollections(ctx context.Context) ([]string, error)

	// ===============================
	// Aggregation and Analytics
	// ===============================
	
	// Aggregate executes an aggregation pipeline for complex data processing.
	// The pipeline is a series of stages that transform, filter, group, and analyze documents.
	// Example: Aggregate(ctx, []map[string]interface{}{{"$group": {"_id": "$status", "count": {"$sum": 1}}}}, &results)
	Aggregate(ctx context.Context, pipeline []map[string]interface{}, dest interface{}) error
}

// WideColumnRepository extends Repository with wide-column store operations.
// Implemented by wide-column adapters like Cassandra, HBase, ScyllaDB, DynamoDB.
// Organizes data in column families with flexible schemas and high scalability.
type WideColumnRepository interface {
	Repository

	// ===============================
	// Column Family Management
	// ===============================
	
	// CreateColumnFamily creates a new column family (similar to a table).
	// Options may include replication factor, consistency levels, compaction strategy.
	// Example: CreateColumnFamily(ctx, "user_posts", map[string]interface{}{"replication_factor": 3})
	CreateColumnFamily(ctx context.Context, name string, options map[string]interface{}) error
	
	// DropColumnFamily removes a column family and all its data.
	// WARNING: This permanently deletes all data in the column family.
	DropColumnFamily(ctx context.Context, name string) error
	
	// ListColumnFamilies returns all column families in the keyspace/database.
	ListColumnFamilies(ctx context.Context) ([]string, error)

	// ===============================
	// Row and Column Operations
	// ===============================
	
	// InsertRow inserts or updates a row with the specified columns.
	// Wide-column stores allow different rows to have different columns (flexible schema).
	// Example: InsertRow(ctx, "users", "user123", map[string]interface{}{"name": "John", "email": "john@example.com"})
	InsertRow(ctx context.Context, columnFamily string, rowKey string, columns map[string]interface{}) error
	
	// GetRow retrieves all columns for a specific row.
	// Returns all columns and their values for the given row key.
	GetRow(ctx context.Context, columnFamily string, rowKey string, dest interface{}) error
	
	// GetColumn retrieves a specific column value from a row.
	// More efficient than GetRow when you only need one column.
	GetColumn(ctx context.Context, columnFamily string, rowKey string, columnName string, dest interface{}) error
	DeleteRow(ctx context.Context, columnFamily string, rowKey string) error
	DeleteColumn(ctx context.Context, columnFamily string, rowKey string, columnName string) error

	// Range queries
	GetRowRange(ctx context.Context, columnFamily string, startKey, endKey string, dest interface{}) error
	GetColumnRange(ctx context.Context, columnFamily string, rowKey string, startColumn, endColumn string, dest interface{}) error

	// Batch operations
	BatchInsert(ctx context.Context, columnFamily string, rows map[string]map[string]interface{}) error
	BatchDelete(ctx context.Context, columnFamily string, rowKeys []string) error

	// Consistency and timestamps
	InsertWithTimestamp(ctx context.Context, columnFamily string, rowKey string, columns map[string]interface{}, timestamp int64) error
	GetWithConsistency(ctx context.Context, columnFamily string, rowKey string, consistency ConsistencyLevel, dest interface{}) error
}

// GraphRepository extends Repository with graph database operations
type GraphRepository interface {
	Repository

	// Vertex operations
	CreateVertex(ctx context.Context, label string, properties map[string]interface{}) (string, error)
	GetVertex(ctx context.Context, id string, dest interface{}) error
	UpdateVertex(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteVertex(ctx context.Context, id string) error
	FindVertices(ctx context.Context, label string, properties map[string]interface{}, dest interface{}) error

	// Edge operations
	CreateEdge(ctx context.Context, fromVertexID, toVertexID string, label string, properties map[string]interface{}) (string, error)
	GetEdge(ctx context.Context, id string, dest interface{}) error
	UpdateEdge(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteEdge(ctx context.Context, id string) error
	FindEdges(ctx context.Context, label string, properties map[string]interface{}, dest interface{}) error

	// Relationship queries
	GetVertexEdges(ctx context.Context, vertexID string, direction EdgeDirection, edgeLabel string, dest interface{}) error
	GetNeighbors(ctx context.Context, vertexID string, direction EdgeDirection, edgeLabel string, dest interface{}) error

	// Path and traversal operations
	FindShortestPath(ctx context.Context, fromVertexID, toVertexID string, maxDepth int, dest interface{}) error
	TraverseGraph(ctx context.Context, startVertexID string, traversal GraphTraversal, dest interface{}) error

	// Graph algorithms
	PageRank(ctx context.Context, iterations int, dampingFactor float64, dest interface{}) error
	ConnectedComponents(ctx context.Context, dest interface{}) error

	// Cypher/Gremlin query support
	ExecuteGraphQuery(ctx context.Context, query string, parameters map[string]interface{}, dest interface{}) error
}

// ConsistencyLevel represents consistency levels for wide-column stores
type ConsistencyLevel string

const (
	ConsistencyOne    ConsistencyLevel = "ONE"
	ConsistencyQuorum ConsistencyLevel = "QUORUM"
	ConsistencyAll    ConsistencyLevel = "ALL"
	ConsistencyLocal  ConsistencyLevel = "LOCAL"
	ConsistencyAny    ConsistencyLevel = "ANY"
)

// EdgeDirection represents edge direction in graph queries
type EdgeDirection string

const (
	EdgeDirectionIn   EdgeDirection = "IN"
	EdgeDirectionOut  EdgeDirection = "OUT"
	EdgeDirectionBoth EdgeDirection = "BOTH"
)

// GraphTraversal represents graph traversal configuration
type GraphTraversal struct {
	MaxDepth     int
	Direction    EdgeDirection
	EdgeLabels   []string
	VertexFilter map[string]interface{}
	EdgeFilter   map[string]interface{}
	Unique       bool
}

// =====================================
// Hierarchical KV Interfaces
// =====================================

// BasicKeyValueRepository provides the minimal interface supported by ALL key-value stores.
// This is the foundation interface that every KV database can implement, from simple
// caches like Memcached to complex systems like Redis. Designed for maximum compatibility.
type BasicKeyValueRepository interface {
	// ===============================
	// Core Key-Value Operations
	// ===============================
	
	// Get retrieves a value by its key.
	// The dest parameter must be a pointer to the target type.
	// Returns ErrorTypeNotFound if the key doesn't exist.
	// Example: Get(ctx, "user:123", &user)
	Get(ctx context.Context, key string, dest interface{}) error
	
	// Set stores a value with the given key.
	// Overwrites existing values. No TTL support in basic interface.
	// Example: Set(ctx, "user:123", user)
	Set(ctx context.Context, key string, value interface{}) error
	
	// Delete removes a key-value pair.
	// Returns no error if the key doesn't exist (idempotent operation).
	// Example: Delete(ctx, "user:123")
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists in the store.
	// More efficient than Get when you only need to check existence.
	// Example: exists, err := Exists(ctx, "user:123")
	Exists(ctx context.Context, key string) (bool, error)
}

// BatchKeyValueRepository extends BasicKeyValueRepository with batch operations.
// Provides multi-key operations for better performance when working with multiple keys.
// Supported by most KV stores except very basic ones like simple Memcached.
type BatchKeyValueRepository interface {
	BasicKeyValueRepository

	// ===============================
	// Batch Operations
	// ===============================
	
	// MGet retrieves multiple values by their keys in a single operation.
	// Much more efficient than multiple Get() calls, especially over networks.
	// The dest parameter must be a pointer to a slice to hold the results.
	// Example: MGet(ctx, []string{"user:1", "user:2"}, &users)
	MGet(ctx context.Context, keys []string, dest interface{}) error
	
	// MSet stores multiple key-value pairs in a single operation.
	// The pairs map contains keys as map keys and values as map values.
	// Example: MSet(ctx, map[string]interface{}{"user:1": user1, "user:2": user2})
	MSet(ctx context.Context, pairs map[string]interface{}) error
	
	// MDelete removes multiple keys in a single operation.
	// More efficient than multiple Delete() calls.
	// Example: MDelete(ctx, []string{"user:1", "user:2", "user:3"})
	MDelete(ctx context.Context, keys []string) error
}

// TTLKeyValueRepository extends BasicKeyValueRepository with Time-To-Live support.
// Allows setting expiration times on keys for automatic cleanup.
// Supported by Redis, ElastiCache, but not basic Memcached or embedded stores like RocksDB.
type TTLKeyValueRepository interface {
	BasicKeyValueRepository

	// ===============================
	// TTL Operations
	// ===============================
	// SetWithTTL stores a value with an expiration time.
	// The key will be automatically deleted after the TTL expires.
	// TTL of 0 means no expiration (equivalent to basic Set).
	// Example: SetWithTTL(ctx, "session:abc", session, 30*time.Minute)
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Expire sets or updates the TTL for an existing key.
	// Useful for extending session timeouts or implementing cache refresh.
	// Example: Expire(ctx, "session:abc", 15*time.Minute)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	
	// TTL returns the remaining time until the key expires.
	// Returns -1 if the key has no expiration, -2 if the key doesn't exist.
	// Example: remaining, err := TTL(ctx, "session:abc")
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// IncrementKeyValueRepository extends BasicKeyValueRepository with atomic numeric operations.
// Provides thread-safe counters and numeric operations without race conditions.
// Essential for counters, metrics, and distributed coordination. Supported by Redis, not by basic stores.
type IncrementKeyValueRepository interface {
	BasicKeyValueRepository

	// ===============================
	// Atomic Numeric Operations
	// ===============================
	// Increment atomically adds delta to a numeric value.
	// Creates the key with value 0 if it doesn't exist, then adds delta.
	// Returns the new value after incrementing.
	// Example: newCount, err := Increment(ctx, "page:views", 1)
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	
	// Decrement atomically subtracts delta from a numeric value.
	// Creates the key with value 0 if it doesn't exist, then subtracts delta.
	// Returns the new value after decrementing.
	// Example: remaining, err := Decrement(ctx, "quota:user123", 1)
	Decrement(ctx context.Context, key string, delta int64) (int64, error)
}

// PatternKeyValueRepository extends BasicKeyValueRepository with pattern-based operations.
// Allows key discovery and bulk operations based on patterns.
// Very powerful but can be expensive on large datasets. Primarily supported by Redis.
type PatternKeyValueRepository interface {
	BasicKeyValueRepository

	// ===============================
	// Pattern and Discovery Operations
	// ===============================
	// Keys returns all keys matching the given pattern.
	// Use '*' for wildcards, '?' for single characters.
	// WARNING: Can be slow on large datasets, use Scan for production.
	// Example: Keys(ctx, "user:*") returns ["user:1", "user:2", ...]
	Keys(ctx context.Context, pattern string) ([]string, error)
	
	// Scan iterates through keys matching a pattern using cursor-based pagination.
	// More efficient than Keys() for large datasets as it doesn't block the database.
	// Returns keys and a new cursor for the next iteration (0 when done).
	// Example: keys, cursor, err := Scan(ctx, 0, "user:*", 100)
	Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error)
}

// AdvancedKeyValueRepository combines all KV capabilities for feature-rich stores like Redis.
// Provides the full spectrum of key-value operations including batching, TTL, atomics, and patterns.
// Only the most advanced KV stores (Redis, Hazelcast) implement this complete interface.
type AdvancedKeyValueRepository interface {
	BatchKeyValueRepository
	TTLKeyValueRepository
	IncrementKeyValueRepository
	PatternKeyValueRepository
}

// KeyValueRepository provides backward compatibility and defaults to the advanced interface.
// Legacy applications can continue using this while new code should use specific capability interfaces.
// Enables gradual migration and type assertions for capability detection.
type KeyValueRepository interface {
	AdvancedKeyValueRepository
}

// =====================================
// Database Classification Examples
// =====================================

/*
Database capabilities by type:

**SQL Databases:**
- PostgreSQL, MySQL, SQLite: SQLRepository
- SQL Server, Oracle: SQLRepository

**Document Stores:**
- MongoDB: DocumentRepository
- CouchDB: DocumentRepository
- Amazon DocumentDB: DocumentRepository
- ArangoDB: DocumentRepository + GraphRepository

**Key-Value Stores:**
- Simple KV: Memcached → BasicKeyValueRepository
- Advanced Memory: Redis → AdvancedKeyValueRepository + specialized data structures
- Embedded KV: RocksDB, LevelDB → BasicKeyValueRepository + BatchKeyValueRepository
- Cloud KV: Amazon ElastiCache → BasicKeyValueRepository + TTLKeyValueRepository

**Wide-Column Stores:**
- Cassandra: WideColumnRepository
- HBase: WideColumnRepository  
- ScyllaDB: WideColumnRepository
- Amazon DynamoDB: WideColumnRepository (with KV access via BasicKeyValueRepository)
- Google Bigtable: WideColumnRepository

**Graph Databases:**
- Neo4j: GraphRepository
- Amazon Neptune: GraphRepository
- ArangoDB: DocumentRepository + GraphRepository
- OrientDB: DocumentRepository + GraphRepository
- JanusGraph: GraphRepository

**Multi-Model Databases:**
- ArangoDB: DocumentRepository + GraphRepository
- CosmosDB: DocumentRepository + WideColumnRepository + GraphRepository
- OrientDB: DocumentRepository + GraphRepository

**Usage Recommendations:**
- Use the most specific interface for your database type (SQLRepository vs DocumentRepository)
- For KV stores: Use BasicKeyValueRepository for maximum compatibility
- Check capabilities via type assertions: repo.(TTLKeyValueRepository)
- Use Preload() QueryOption for relationships in base Repository interface
- Use FindWithRelations() methods via SQLRepository interface for convenience
- For schema migration: Check if repo implements MigratableRepository (currently GORM only)
  Example: if migrator, ok := repo.(MigratableRepository); ok { migrator.MigrateTable(ctx, entity) }
- Multi-model databases can implement multiple interfaces (e.g., ArangoDB: DocumentRepository + GraphRepository)
*/

// =====================================
// Events and Hooks
// =====================================

// EventHook represents hooks for database events
type EventHook interface {
	BeforeCreate(ctx context.Context, entity interface{}) error
	AfterCreate(ctx context.Context, entity interface{}) error
	BeforeUpdate(ctx context.Context, entity interface{}) error
	AfterUpdate(ctx context.Context, entity interface{}) error
	BeforeDelete(ctx context.Context, entity interface{}) error
	AfterDelete(ctx context.Context, entity interface{}) error
}

// =====================================
// Error Types
// =====================================

// Error types for different database errors
type ErrorType string

const (
	ErrorTypeConnection      ErrorType = "connection"
	ErrorTypeConstraint      ErrorType = "constraint"
	ErrorTypeNotFound        ErrorType = "not_found"
	ErrorTypeDuplicate       ErrorType = "duplicate"
	ErrorTypeTimeout         ErrorType = "timeout"
	ErrorTypeTransaction     ErrorType = "transaction"
	ErrorTypeValidation      ErrorType = "validation"
	ErrorTypeUnsupported     ErrorType = "unsupported"
	ErrorTypeDatabase        ErrorType = "database"
	ErrorTypeSerialization   ErrorType = "serialization"
	ErrorTypeInvalidArgument ErrorType = "invalid_argument"
)

// GPAError represents a GPA-specific error
type GPAError struct {
	Type    ErrorType
	Message string
	Cause   error
	Code    string
}

func (e GPAError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e GPAError) Unwrap() error {
	return e.Cause
}

// =====================================
// Registry for Provider Management
// =====================================

// ProviderRegistry manages registered providers
type ProviderRegistry interface {
	Register(name string, factory ProviderFactory) error
	Get(name string) (ProviderFactory, error)
	List() []string
	Unregister(name string) error
}

// ProviderFactory creates new provider instances
type ProviderFactory interface {
	Create(config Config) (Provider, error)
	SupportedDrivers() []string
}

// DefaultRegistry is the default provider registry
var DefaultRegistry ProviderRegistry = NewRegistry()

// NewRegistry creates a new provider registry
func NewRegistry() ProviderRegistry {
	return &registry{
		providers: make(map[string]ProviderFactory),
	}
}

type registry struct {
	providers map[string]ProviderFactory
}

func (r *registry) Register(name string, factory ProviderFactory) error {
	r.providers[name] = factory
	return nil
}

func (r *registry) Get(name string) (ProviderFactory, error) {
	factory, exists := r.providers[name]
	if !exists {
		return nil, GPAError{
			Type:    ErrorTypeNotFound,
			Message: "provider not found: " + name,
		}
	}
	return factory, nil
}

func (r *registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

func (r *registry) Unregister(name string) error {
	delete(r.providers, name)
	return nil
}

// =====================================
// Utility Functions
// =====================================

// NewProvider creates a new provider instance
func NewProvider(driverName string, config Config) (Provider, error) {
	factory, err := DefaultRegistry.Get(driverName)
	if err != nil {
		return nil, err
	}
	return factory.Create(config)
}

// RegisterProvider registers a new provider factory
func RegisterProvider(name string, factory ProviderFactory) error {
	return DefaultRegistry.Register(name, factory)
}

// ListProviders returns all registered provider names
func ListProviders() []string {
	return DefaultRegistry.List()
}
