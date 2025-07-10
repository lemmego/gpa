package gpa

import "context"

// =====================================
// Document Database Repository Interface
// =====================================

// DocumentRepository extends Repository with document database-specific operations.
// Designed for NoSQL document stores like MongoDB, CouchDB, etc.
type DocumentRepository[T any] interface {
	Repository[T]

	// FindByDocument retrieves entities matching a document-style query.
	// The query is a map representing the document structure to match.
	// Returns a slice of entity pointers with compile-time type safety.
	// Example: users, err := FindByDocument(ctx, map[string]interface{}{"status": "active", "age": map[string]interface{}{"$gte": 18}})
	FindByDocument(ctx context.Context, query map[string]interface{}) ([]*T, error)

	// UpdateDocument updates an entity using document-style operations.
	// The update parameter contains the update operations (e.g., $set, $unset, $inc).
	// Returns the number of documents modified.
	// Example: count, err := UpdateDocument(ctx, userID, map[string]interface{}{"$set": map[string]interface{}{"status": "inactive"}})
	UpdateDocument(ctx context.Context, id interface{}, update map[string]interface{}) (int64, error)

	// UpdateManyDocuments updates multiple entities using document-style operations.
	// Combines a query filter with update operations.
	// Returns the number of documents modified.
	// Example: count, err := UpdateManyDocuments(ctx, filter, update)
	UpdateManyDocuments(ctx context.Context, filter map[string]interface{}, update map[string]interface{}) (int64, error)

	// Aggregate performs aggregation operations using database-specific pipelines.
	// For MongoDB, this uses the aggregation pipeline. Other databases may use similar concepts.
	// Returns aggregated results as a slice of maps.
	// Example: results, err := Aggregate(ctx, []map[string]interface{}{{"$group": {"_id": "$status", "count": {"$sum": 1}}}})
	Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error)

	// CreateIndex creates an index on the specified fields.
	// The keys parameter maps field names to index direction (1 for ascending, -1 for descending).
	// For document databases, this might include text indexes, geospatial indexes, etc.
	// Example: err := CreateIndex(ctx, map[string]interface{}{"email": 1, "status": 1}, false)
	CreateIndex(ctx context.Context, keys map[string]interface{}, unique bool) error

	// DropIndex removes an index by name.
	// Example: err := DropIndex(ctx, "email_1_status_1")
	DropIndex(ctx context.Context, indexName string) error

	// TextSearch performs full-text search on indexed text fields.
	// The query string contains the search terms.
	// Returns entities matching the search criteria with compile-time type safety.
	// Example: users, err := TextSearch(ctx, "john developer", opts...)
	TextSearch(ctx context.Context, query string, opts ...QueryOption) ([]*T, error)

	// FindNear finds entities near a geographical point.
	// Requires geospatial indexes on the queried fields.
	// Returns entities sorted by distance with compile-time type safety.
	// Example: places, err := FindNear(ctx, "location", []float64{-73.9857, 40.7484}, 1000)
	FindNear(ctx context.Context, field string, point []float64, maxDistance float64) ([]*T, error)

	// FindWithinPolygon finds entities within a geographical polygon.
	// The polygon is defined by an array of coordinate points.
	// Returns matching entities with compile-time type safety.
	// Example: places, err := FindWithinPolygon(ctx, "location", polygon)
	FindWithinPolygon(ctx context.Context, field string, polygon [][]float64) ([]*T, error)
}

// =====================================
// Wide Column Store Interface
// =====================================

// WideColumnRepository represents wide column stores like Cassandra, HBase.
// Optimized for time-series data and large-scale analytics.
type WideColumnRepository[T any] interface {
	Repository[T]

	// FindByPartitionKey retrieves entities by partition key.
	// Partition keys determine data distribution across nodes.
	// Returns entities within the same partition with compile-time type safety.
	// Example: events, err := FindByPartitionKey(ctx, partitionKey, opts...)
	FindByPartitionKey(ctx context.Context, partitionKey interface{}, opts ...QueryOption) ([]*T, error)

	// FindByRange retrieves entities within a clustering key range.
	// Clustering keys determine sort order within partitions.
	// Returns entities within the specified range with compile-time type safety.
	// Example: events, err := FindByRange(ctx, partitionKey, startKey, endKey)
	FindByRange(ctx context.Context, partitionKey, startKey, endKey interface{}) ([]*T, error)

	// FindByTimeRange retrieves entities within a time range.
	// Optimized for time-series data queries.
	// Returns time-ordered entities with compile-time type safety.
	// Example: metrics, err := FindByTimeRange(ctx, startTime, endTime, opts...)
	FindByTimeRange(ctx context.Context, startTime, endTime interface{}, opts ...QueryOption) ([]*T, error)

	// Compact triggers compaction of the underlying storage.
	// Improves read performance by merging and organizing data files.
	// Returns when compaction is complete or fails.
	// Example: err := Compact(ctx)
	Compact(ctx context.Context) error
}

// =====================================
// Graph Database Interface
// =====================================

// GraphRepository represents graph databases like Neo4j, ArangoDB.
// Optimized for relationship-heavy data and graph traversals.
type GraphRepository[T any] interface {
	Repository[T]

	// FindConnected finds entities connected to the given entity.
	// Follows relationships based on the specified relationship type and direction.
	// Returns connected entities with compile-time type safety.
	// Example: friends, err := FindConnected(ctx, userID, "FRIEND", "outgoing", 1)
	FindConnected(ctx context.Context, entityID interface{}, relationshipType string, direction string, depth int) ([]*T, error)

	// FindPath finds the shortest path between two entities.
	// Returns a list of entities representing the path with compile-time type safety.
	// Example: path, err := FindPath(ctx, startID, endID, "CONNECTED")
	FindPath(ctx context.Context, startID, endID interface{}, relationshipType string) ([]*T, error)

	// TraverseGraph performs a custom graph traversal.
	// Uses a traversal specification to define the traversal pattern.
	// Returns entities found during traversal with compile-time type safety.
	// Example: results, err := TraverseGraph(ctx, startID, traversal)
	TraverseGraph(ctx context.Context, startID interface{}, traversal GraphTraversal) ([]*T, error)
}

// GraphTraversal represents a graph traversal specification
type GraphTraversal struct {
	StartingPoints []interface{}
	MaxDepth       int
	Relationships  []string
	Direction      string
	Filters        []Condition
}
