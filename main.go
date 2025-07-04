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

// Repository is the main interface for database operations
type Repository interface {
	// Basic CRUD operations
	Create(ctx context.Context, entity interface{}) error
	CreateBatch(ctx context.Context, entities interface{}) error
	FindByID(ctx context.Context, id interface{}, dest interface{}) error
	FindAll(ctx context.Context, dest interface{}, opts ...QueryOption) error
	Update(ctx context.Context, entity interface{}) error
	UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error
	Delete(ctx context.Context, id interface{}) error
	DeleteByCondition(ctx context.Context, condition Condition) error

	// Query operations
	Query(ctx context.Context, dest interface{}, opts ...QueryOption) error
	QueryOne(ctx context.Context, dest interface{}, opts ...QueryOption) error
	Count(ctx context.Context, opts ...QueryOption) (int64, error)
	Exists(ctx context.Context, opts ...QueryOption) (bool, error)

	// Advanced operations
	Transaction(ctx context.Context, fn TransactionFunc) error
	RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error
	RawExec(ctx context.Context, query string, args []interface{}) (Result, error)

	// Metadata
	GetEntityInfo(entity interface{}) (*EntityInfo, error)
	Close() error
}

// TransactionFunc represents a function that runs within a transaction
type TransactionFunc func(tx Transaction) error

// Transaction interface for transactional operations
type Transaction interface {
	Repository
	Commit() error
	Rollback() error
}

// Result represents the result of a database operation
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// =====================================
// Provider and Configuration
// =====================================

// Provider is the main interface for creating repositories
type Provider interface {
	// Repository creation
	Repository(entityType reflect.Type) Repository
	RepositoryFor(entity interface{}) Repository

	// Configuration and lifecycle
	Configure(config Config) error
	Health() error
	Close() error

	// Metadata
	SupportedFeatures() []Feature
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
	Query       string
	Args        []interface{}
	Type        SubQueryType
	Field       string // Field for subquery conditions
	Operator    Operator
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

// SQLRepository extends Repository with SQL-specific operations
type SQLRepository interface {
	Repository

	// SQL-specific operations
	FindBySQL(ctx context.Context, sql string, args []interface{}, dest interface{}) error
	ExecSQL(ctx context.Context, sql string, args ...interface{}) (Result, error)

	// Schema operations
	CreateTable(ctx context.Context, entity interface{}) error
	DropTable(ctx context.Context, entity interface{}) error
	MigrateTable(ctx context.Context, entity interface{}) error

	// Index operations
	CreateIndex(ctx context.Context, entity interface{}, fields []string, unique bool) error
	DropIndex(ctx context.Context, entity interface{}, indexName string) error
}

// NoSQLRepository extends Repository with NoSQL-specific operations
type NoSQLRepository interface {
	Repository

	// Document operations
	FindByDocument(ctx context.Context, document map[string]interface{}, dest interface{}) error
	UpdateDocument(ctx context.Context, id interface{}, document map[string]interface{}) error

	// Collection operations
	CreateCollection(ctx context.Context, name string) error
	DropCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]string, error)

	// Aggregation
	Aggregate(ctx context.Context, pipeline []map[string]interface{}, dest interface{}) error
}

// KeyValueRepository for key-value stores
type KeyValueRepository interface {
	// Basic KV operations
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Batch operations
	MGet(ctx context.Context, keys []string, dest interface{}) error
	MSet(ctx context.Context, pairs map[string]interface{}, ttl time.Duration) error
	MDelete(ctx context.Context, keys []string) error

	// Advanced operations
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Pattern operations
	Keys(ctx context.Context, pattern string) ([]string, error)
	Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error)
}

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
	ErrorTypeConnection     ErrorType = "connection"
	ErrorTypeConstraint     ErrorType = "constraint"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeDuplicate      ErrorType = "duplicate"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeTransaction    ErrorType = "transaction"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeUnsupported    ErrorType = "unsupported"
	ErrorTypeDatabase       ErrorType = "database"
	ErrorTypeSerialization  ErrorType = "serialization"
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
