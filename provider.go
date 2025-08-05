package gpa

import (
	"context"
	"time"
)

type ProviderFunc func(...string) Provider

// =====================================
// Provider Interface
// =====================================

// Provider is the main interface for database provider implementations.
// Provides unified access to database operations with type safety through
// repository creation functions.
//
// This interface is implemented by all provider packages:
// • gpagorm.Provider (GORM SQL adapter)
// • gpabun.Provider (Bun SQL adapter)
// • gpamongo.Provider (MongoDB adapter)
// • gparedis.Provider (Redis adapter)
//
// Usage:
//
//	provider, err := gpagorm.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Create type-safe repositories
//	userRepo := gpagorm.GetRepository[User](provider)
//	postRepo := gpagorm.GetRepository[Post](provider)
type Provider interface {
	// Configure applies new configuration to the provider.
	// Can be used to change connection settings, pool sizes, etc. at runtime.
	// May require reconnection depending on what settings changed.
	Configure(config Config) error

	// Health checks if the database connection is healthy and responsive.
	// Returns error if the database is unreachable or not functioning properly.
	// Useful for health check endpoints and monitoring.
	Health() error

	// Close shuts down the provider and releases all resources.
	// Closes database providers, stops background tasks, etc.
	// Should be called during application shutdown.
	Close() error

	// SupportedFeatures returns a list of features this provider supports.
	// Features include things like transactions, full-text search, pub/sub, etc.
	// Use this to check capabilities before using advanced features.
	SupportedFeatures() []Feature

	// ProviderInfo returns metadata about this provider.
	// Includes provider name, version, database type, and supported features.
	// Useful for debugging, logging, and feature detection.
	ProviderInfo() ProviderInfo
}

// =====================================
// Specialized Provider Interfaces
// =====================================

// SQLProvider extends Provider with SQL-specific functionality
// Implemented by providers that support SQL databases (GORM, Bun, etc.)
type SQLProvider interface {
	Provider

	// DB returns the underlying database/sql.DB instance
	// This allows direct access to the native SQL connection for advanced operations
	DB() interface{}

	// BeginTx starts a transaction with specific isolation level
	BeginTx(ctx context.Context, opts *TxOptions) (interface{}, error)

	// Migrate runs database migrations
	Migrate(models ...interface{}) error

	// RawQuery executes raw SQL and returns results
	RawQuery(ctx context.Context, query string, args ...interface{}) (interface{}, error)

	// RawExec executes raw SQL without returning results
	RawExec(ctx context.Context, query string, args ...interface{}) (Result, error)
}

// KeyValueProvider extends Provider with key-value store functionality
// Implemented by providers like Redis, memcached, etc.
type KeyValueProvider interface {
	Provider

	// Client returns the underlying client instance
	Client() interface{}

	// Set stores a key-value pair with optional TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a value by key
	Get(ctx context.Context, key string) (interface{}, error)

	// Delete removes a key
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// Keys returns all keys matching a pattern
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Expire sets TTL for a key
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// TTL returns the remaining TTL for a key
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// DocumentProvider extends Provider with document database functionality
// Implemented by providers like MongoDB, CouchDB, etc.
type DocumentProvider interface {
	Provider

	// Database returns the underlying database instance
	Database() interface{}

	// Collection returns a collection instance
	Collection(name string) interface{}

	// CreateIndex creates an index on a collection
	CreateIndex(ctx context.Context, collection string, keys interface{}, options *IndexOptions) error

	// DropIndex drops an index from a collection
	DropIndex(ctx context.Context, collection string, name string) error

	// ListIndexes lists all indexes for a collection
	ListIndexes(ctx context.Context, collection string) ([]IndexInfo, error)

	// Aggregate runs an aggregation pipeline
	Aggregate(ctx context.Context, collection string, pipeline interface{}) (interface{}, error)

	// Watch starts a change stream
	Watch(ctx context.Context, collection string, pipeline interface{}) (interface{}, error)
}

// WideColumnProvider extends Provider with wide-column store functionality
// Implemented by providers like Cassandra, HBase, etc.
type WideColumnProvider interface {
	Provider

	// Session returns the underlying session instance
	Session() interface{}

	// CreateKeyspace creates a keyspace
	CreateKeyspace(ctx context.Context, name string, options *KeyspaceOptions) error

	// CreateTable creates a table in a keyspace
	CreateTable(ctx context.Context, keyspace string, table string, schema interface{}) error

	// PrepareStatement prepares a statement for execution
	PrepareStatement(ctx context.Context, query string) (interface{}, error)

	// BatchExecute executes multiple statements as a batch
	BatchExecute(ctx context.Context, statements []interface{}) error
}

// GraphProvider extends Provider with graph database functionality
// Implemented by providers like Neo4j, ArangoDB, etc.
type GraphProvider interface {
	Provider

	// Driver returns the underlying driver instance
	Driver() interface{}

	// Session returns a session for executing queries
	Session(ctx context.Context) (interface{}, error)

	// CypherQuery executes a Cypher query (for Neo4j-compatible databases)
	CypherQuery(ctx context.Context, query string, params map[string]interface{}) (interface{}, error)

	// CreateNode creates a node in the graph
	CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (interface{}, error)

	// CreateRelationship creates a relationship between two nodes
	CreateRelationship(ctx context.Context, fromNode, toNode interface{}, relType string, properties map[string]interface{}) (interface{}, error)

	// FindPath finds paths between nodes
	FindPath(ctx context.Context, from, to interface{}, maxDepth int) (interface{}, error)
}

// =====================================
// Supporting Types
// =====================================

// TxOptions represents transaction options
type TxOptions struct {
	IsolationLevel IsolationLevel
	ReadOnly       bool
	Timeout        time.Duration
}

// IsolationLevel represents transaction isolation levels
type IsolationLevel string

const (
	IsolationDefault         IsolationLevel = "DEFAULT"
	IsolationReadUncommitted IsolationLevel = "READ_UNCOMMITTED"
	IsolationReadCommitted   IsolationLevel = "READ_COMMITTED"
	IsolationRepeatableRead  IsolationLevel = "REPEATABLE_READ"
	IsolationSerializable    IsolationLevel = "SERIALIZABLE"
)

// IndexOptions represents options for creating indexes
type IndexOptions struct {
	Unique     bool
	Sparse     bool
	Background bool
	TTL        time.Duration
	Name       string
}

// KeyspaceOptions represents options for creating keyspaces
type KeyspaceOptions struct {
	ReplicationFactor int
	ReplicationClass  string
	DurableWrites     bool
}

// =====================================
// Type Assertion Helpers
// =====================================

// AsSQLProvider safely casts a Provider to SQLProvider
// Returns the SQLProvider instance and true if successful, nil and false otherwise
func AsSQLProvider(provider Provider) (SQLProvider, bool) {
	if sqlProvider, ok := provider.(SQLProvider); ok {
		return sqlProvider, true
	}
	return nil, false
}

// AsKeyValueProvider safely casts a Provider to KeyValueProvider
// Returns the KeyValueProvider instance and true if successful, nil and false otherwise
func AsKeyValueProvider(provider Provider) (KeyValueProvider, bool) {
	if kvProvider, ok := provider.(KeyValueProvider); ok {
		return kvProvider, true
	}
	return nil, false
}

// AsDocumentProvider safely casts a Provider to DocumentProvider
// Returns the DocumentProvider instance and true if successful, nil and false otherwise
func AsDocumentProvider(provider Provider) (DocumentProvider, bool) {
	if docProvider, ok := provider.(DocumentProvider); ok {
		return docProvider, true
	}
	return nil, false
}

// AsWideColumnProvider safely casts a Provider to WideColumnProvider
// Returns the WideColumnProvider instance and true if successful, nil and false otherwise
func AsWideColumnProvider(provider Provider) (WideColumnProvider, bool) {
	if wcProvider, ok := provider.(WideColumnProvider); ok {
		return wcProvider, true
	}
	return nil, false
}

// AsGraphProvider safely casts a Provider to GraphProvider
// Returns the GraphProvider instance and true if successful, nil and false otherwise
func AsGraphProvider(provider Provider) (GraphProvider, bool) {
	if graphProvider, ok := provider.(GraphProvider); ok {
		return graphProvider, true
	}
	return nil, false
}

// =====================================
// Provider Type Checks
// =====================================

// IsSQLProvider checks if a provider implements SQLProvider interface
func IsSQLProvider(provider Provider) bool {
	_, ok := provider.(SQLProvider)
	return ok
}

// IsKeyValueProvider checks if a provider implements KeyValueProvider interface
func IsKeyValueProvider(provider Provider) bool {
	_, ok := provider.(KeyValueProvider)
	return ok
}

// IsDocumentProvider checks if a provider implements DocumentProvider interface
func IsDocumentProvider(provider Provider) bool {
	_, ok := provider.(DocumentProvider)
	return ok
}

// IsWideColumnProvider checks if a provider implements WideColumnProvider interface
func IsWideColumnProvider(provider Provider) bool {
	_, ok := provider.(WideColumnProvider)
	return ok
}

// IsGraphProvider checks if a provider implements GraphProvider interface
func IsGraphProvider(provider Provider) bool {
	_, ok := provider.(GraphProvider)
	return ok
}
