package gpa

import "time"

// =====================================
// Core Types and Constants
// =====================================

// Config represents database connection configuration
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
	FeatureIndexes        Feature = "indexes"
	FeatureTTL            Feature = "ttl"
	FeatureAtomicOps      Feature = "atomic_ops"
	FeatureFullText       Feature = "full_text_search"
	FeatureGeoSpatial     Feature = "geospatial"
	FeatureGeospatial     Feature = "geospatial"
	FeatureSubQueries     Feature = "subqueries"
	FeatureJoins          Feature = "joins"
	FeatureJSONQueries    Feature = "json_queries"
	FeatureIndexing       Feature = "indexing"
	FeatureAggregation    Feature = "aggregation"
	FeatureFullTextSearch Feature = "full_text_search"
	FeaturePubSub         Feature = "pub_sub"
	FeatureStreaming      Feature = "streaming"
	FeatureReplication    Feature = "replication"
	FeatureSharding       Feature = "sharding"
	FeatureMigration      Feature = "migration"
	FeatureRawSQL         Feature = "raw_sql"
)

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

// LogicOperator represents logic operators for combining conditions
type LogicOperator string

const (
	LogicAnd LogicOperator = "AND"
	LogicOr  LogicOperator = "OR"
	LogicNot LogicOperator = "NOT"
)

// Order represents sorting order
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

// JoinType represents types of table joins
type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinFull  JoinType = "FULL"
)

// LockType represents database lock types
type LockType string

const (
	LockNone         LockType = "NONE"
	LockShared       LockType = "SHARED"
	LockExclusive    LockType = "EXCLUSIVE"
	LockUpdateNoWait LockType = "UPDATE_NOWAIT"
	LockForUpdate    LockType = "FOR_UPDATE"
	LockForShare     LockType = "FOR_SHARE"
)

// IndexType represents different types of database indexes
type IndexType string

const (
	IndexTypePrimary    IndexType = "primary"
	IndexTypeUnique     IndexType = "unique"
	IndexTypeStandard   IndexType = "standard"
	IndexTypeFullText   IndexType = "fulltext"
	IndexTypeGeoSpatial IndexType = "geospatial"
	IndexTypeComposite  IndexType = "composite"
)

// RelationType represents different types of entity relationships
type RelationType string

const (
	RelationOneToOne   RelationType = "one_to_one"
	RelationOneToMany  RelationType = "one_to_many"
	RelationManyToOne  RelationType = "many_to_one"
	RelationManyToMany RelationType = "many_to_many"
)

// SubQueryType represents the type of subquery operation
type SubQueryType string

const (
	SubQueryExists         SubQueryType = "EXISTS"
	SubQueryNotExists      SubQueryType = "NOT EXISTS"
	SubQueryIn             SubQueryType = "IN"
	SubQueryNotIn          SubQueryType = "NOT IN"
	SubQueryScalar         SubQueryType = "SCALAR"
	SubQueryAny            SubQueryType = "ANY"
	SubQueryAll            SubQueryType = "ALL"
	SubQueryTypeExists     SubQueryType = "EXISTS"
	SubQueryTypeIn         SubQueryType = "IN"
	SubQueryTypeScalar     SubQueryType = "SCALAR"
	SubQueryTypeCorrelated SubQueryType = "CORRELATED"
)

// ErrorType represents different types of errors that can occur
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeDuplicate    ErrorType = "duplicate"
	ErrorTypeConnection   ErrorType = "connection"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypePermission   ErrorType = "permission"
	ErrorTypeConstraint      ErrorType = "constraint"
	ErrorTypeTransaction     ErrorType = "transaction"
	ErrorTypeUnsupported     ErrorType = "unsupported"
	ErrorTypeInternal        ErrorType = "internal"
	ErrorTypeSerialization   ErrorType = "serialization"
	ErrorTypeInvalidArgument ErrorType = "invalid_argument"
	ErrorTypeDatabase        ErrorType = "database"
)