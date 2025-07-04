// Package gpamongo provides a MongoDB adapter for the Go Persistence API (GPA)
package gpamongo

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/lemmego/gpa"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// =====================================
// Provider Implementation
// =====================================

// Provider implements gpa.Provider using MongoDB
type Provider struct {
	client   *mongo.Client
	database *mongo.Database
	config   gpa.Config
}

// Factory implements gpa.ProviderFactory
type Factory struct{}

// Create creates a new MongoDB provider instance
func (f *Factory) Create(config gpa.Config) (gpa.Provider, error) {
	provider := &Provider{config: config}

	// Build connection string
	connectionURI := f.buildConnectionURI(config)

	// Create client options
	clientOpts := options.Client().ApplyURI(connectionURI)

	// Apply additional options
	if options, ok := config.Options["mongo"]; ok {
		if mongoOpts, ok := options.(map[string]interface{}); ok {
			f.applyClientOptions(clientOpts, mongoOpts)
		}
	}

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to connect to MongoDB",
			Cause:   err,
		}
	}

	// Test the connection
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to ping MongoDB",
			Cause:   err,
		}
	}

	provider.client = client
	provider.database = client.Database(config.Database)

	return provider, nil
}

// SupportedDrivers returns the list of supported database drivers
func (f *Factory) SupportedDrivers() []string {
	return []string{"mongodb", "mongo"}
}

// buildConnectionURI builds MongoDB connection URI
func (f *Factory) buildConnectionURI(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	uri := "mongodb://"

	// Add credentials if provided
	if config.Username != "" {
		uri += config.Username
		if config.Password != "" {
			uri += ":" + config.Password
		}
		uri += "@"
	}

	// Add host and port
	host := config.Host
	if host == "" {
		host = "localhost"
	}
	port := config.Port
	if port == 0 {
		port = 27017
	}

	uri += fmt.Sprintf("%s:%d", host, port)

	// Add database
	if config.Database != "" {
		uri += "/" + config.Database
	}

	// Add SSL options
	if config.SSL.Enabled {
		uri += "?ssl=true"
		if config.SSL.CAFile != "" {
			uri += "&sslCAFile=" + config.SSL.CAFile
		}
		if config.SSL.CertFile != "" {
			uri += "&sslCertificateKeyFile=" + config.SSL.CertFile
		}
	}

	return uri
}

// applyClientOptions applies MongoDB-specific client options
func (f *Factory) applyClientOptions(clientOpts *options.ClientOptions, mongoOpts map[string]interface{}) {
	if maxPoolSize, ok := mongoOpts["max_pool_size"].(int); ok {
		clientOpts.SetMaxPoolSize(uint64(maxPoolSize))
	}
	if minPoolSize, ok := mongoOpts["min_pool_size"].(int); ok {
		clientOpts.SetMinPoolSize(uint64(minPoolSize))
	}
	if maxIdleTime, ok := mongoOpts["max_idle_time"].(time.Duration); ok {
		clientOpts.SetMaxConnIdleTime(maxIdleTime)
	}
}

// Repository returns a repository for the given entity type
func (p *Provider) Repository(entityType reflect.Type) gpa.Repository {
	return &Repository{
		database:   p.database,
		client:     p.client,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.client.Ping(ctx, readpref.Primary())
}

// Close closes the database connection
func (p *Provider) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.client.Disconnect(ctx)
}

// SupportedFeatures returns the list of supported features
func (p *Provider) SupportedFeatures() []gpa.Feature {
	return []gpa.Feature{
		gpa.FeatureTransactions,
		gpa.FeatureJSONQueries,
		gpa.FeatureIndexing,
		gpa.FeatureAggregation,
		gpa.FeatureFullTextSearch,
		gpa.FeatureGeospatial,
		gpa.FeatureSharding,
		gpa.FeatureReplication,
	}
}

// ProviderInfo returns information about this provider
func (p *Provider) ProviderInfo() gpa.ProviderInfo {
	return gpa.ProviderInfo{
		Name:         "MongoDB",
		Version:      "1.0.0",
		DatabaseType: gpa.DatabaseTypeDocument,
		Features:     p.SupportedFeatures(),
	}
}

// =====================================
// Repository Implementation
// =====================================

// Repository implements gpa.Repository and gpa.NoSQLRepository using MongoDB
type Repository struct {
	database   *mongo.Database
	client     *mongo.Client
	entityType reflect.Type
	provider   *Provider
}

// getCollectionName returns the collection name for the entity
func (r *Repository) getCollectionName() string {
	// Check if entity has a CollectionName method
	entityPtr := reflect.New(r.entityType)
	if method := entityPtr.MethodByName("CollectionName"); method.IsValid() {
		results := method.Call(nil)
		if len(results) > 0 && results[0].Kind() == reflect.String {
			return results[0].String()
		}
	}

	// Default to lowercase struct name with 's' suffix
	name := strings.ToLower(r.entityType.Name())
	if !strings.HasSuffix(name, "s") {
		name += "s"
	}
	return name
}

// getCollection returns the MongoDB collection for this repository
func (r *Repository) getCollection() *mongo.Collection {
	return r.database.Collection(r.getCollectionName())
}

// Create creates a new entity
func (r *Repository) Create(ctx context.Context, entity interface{}) error {
	collection := r.getCollection()

	// Set ID if not provided
	r.ensureID(entity)

	result, err := collection.InsertOne(ctx, entity)
	if err != nil {
		return convertMongoError(err)
	}

	// Set the inserted ID back to the entity if it was generated
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		r.setFieldValue(entity, "_id", oid)
		r.setFieldValue(entity, "id", oid.Hex())
	}

	return nil
}

// CreateBatch creates multiple entities in a batch
func (r *Repository) CreateBatch(ctx context.Context, entities interface{}) error {
	collection := r.getCollection()

	// Convert entities to []interface{}
	entitiesSlice, err := r.convertToSlice(entities)
	if err != nil {
		return err
	}

	// Ensure IDs for all entities
	for _, entity := range entitiesSlice {
		r.ensureID(entity)
	}

	results, err := collection.InsertMany(ctx, entitiesSlice)
	if err != nil {
		return convertMongoError(err)
	}

	// Set the inserted IDs back to the entities
	for i, insertedID := range results.InsertedIDs {
		if oid, ok := insertedID.(primitive.ObjectID); ok {
			r.setFieldValue(entitiesSlice[i], "_id", oid)
			r.setFieldValue(entitiesSlice[i], "id", oid.Hex())
		}
	}

	return nil
}

// FindByID finds an entity by its ID
func (r *Repository) FindByID(ctx context.Context, id interface{}, dest interface{}) error {
	collection := r.getCollection()

	// Convert ID to ObjectID if it's a string
	mongoID, err := r.convertToObjectID(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": mongoID}
	result := collection.FindOne(ctx, filter)

	err = result.Decode(dest)
	if err != nil {
		return convertMongoError(err)
	}

	return nil
}

// FindAll finds all entities matching the given options
func (r *Repository) FindAll(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	collection := r.getCollection()

	// Build query
	filter, findOpts := r.buildQuery(opts...)

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return convertMongoError(err)
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, dest)
	if err != nil {
		return convertMongoError(err)
	}

	return nil
}

// Update updates an entity
func (r *Repository) Update(ctx context.Context, entity interface{}) error {
	collection := r.getCollection()

	// Get ID from entity
	id := r.getFieldValue(entity, "_id")
	if id == nil {
		id = r.getFieldValue(entity, "id")
		if id != nil {
			// Convert string ID to ObjectID
			if strID, ok := id.(string); ok {
				objID, err := primitive.ObjectIDFromHex(strID)
				if err != nil {
					return gpa.GPAError{
						Type:    gpa.ErrorTypeValidation,
						Message: "invalid ID format",
						Cause:   err,
					}
				}
				id = objID
			}
		}
	}

	if id == nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "entity must have an ID for update",
		}
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$set": entity}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return convertMongoError(err)
	}

	if result.MatchedCount == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found",
		}
	}

	return nil
}

// UpdatePartial updates specific fields of an entity
func (r *Repository) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	collection := r.getCollection()

	// Convert ID to ObjectID if it's a string
	mongoID, err := r.convertToObjectID(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": mongoID}
	update := bson.M{"$set": updates}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return convertMongoError(err)
	}

	if result.MatchedCount == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found",
		}
	}

	return nil
}

// Delete deletes an entity by ID
func (r *Repository) Delete(ctx context.Context, id interface{}) error {
	collection := r.getCollection()

	// Convert ID to ObjectID if it's a string
	mongoID, err := r.convertToObjectID(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": mongoID}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return convertMongoError(err)
	}

	if result.DeletedCount == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "entity not found",
		}
	}

	return nil
}

// DeleteByCondition deletes entities matching the given condition
func (r *Repository) DeleteByCondition(ctx context.Context, condition gpa.Condition) error {
	collection := r.getCollection()

	filter := r.buildCondition(condition)
	_, err := collection.DeleteMany(ctx, filter)
	return convertMongoError(err)
}

// Query executes a query with the given options
func (r *Repository) Query(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	return r.FindAll(ctx, dest, opts...)
}

// QueryOne executes a query and returns a single result
func (r *Repository) QueryOne(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	collection := r.getCollection()

	// Build query
	filter, findOpts := r.buildQuery(opts...)

	// Convert FindOptions to FindOneOptions
	findOneOpts := options.FindOne()
	if findOpts.Projection != nil {
		findOneOpts.SetProjection(findOpts.Projection)
	}
	if findOpts.Sort != nil {
		findOneOpts.SetSort(findOpts.Sort)
	}
	if findOpts.Skip != nil {
		findOneOpts.SetSkip(*findOpts.Skip)
	}

	result := collection.FindOne(ctx, filter, findOneOpts)
	err := result.Decode(dest)
	if err != nil {
		return convertMongoError(err)
	}

	return nil
}

// Count counts entities matching the given options
func (r *Repository) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	collection := r.getCollection()

	// Build query (only need filter, not find options)
	filter, _ := r.buildQuery(opts...)

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, convertMongoError(err)
	}

	return count, nil
}

// Exists checks if any entity matches the given options
func (r *Repository) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	return count > 0, err
}

// Transaction executes a function within a transaction
func (r *Repository) Transaction(ctx context.Context, fn gpa.TransactionFunc) error {
	session, err := r.client.StartSession()
	if err != nil {
		return convertMongoError(err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		err := session.StartTransaction()
		if err != nil {
			return convertMongoError(err)
		}

		txRepo := &Transaction{
			Repository: &Repository{
				database:   r.database,
				client:     r.client,
				entityType: r.entityType,
				provider:   r.provider,
			},
			session: session,
		}

		err = fn(txRepo)
		if err != nil {
			session.AbortTransaction(sc)
			return err
		}

		return session.CommitTransaction(sc)
	})
}

// RawQuery executes a raw MongoDB query
func (r *Repository) RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error {
	// For MongoDB, we'll interpret the query as a BSON filter
	// This is a simplified implementation
	collection := r.getCollection()

	// Parse query as BSON
	var filter bson.M
	err := bson.UnmarshalExtJSON([]byte(query), true, &filter)
	if err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "invalid MongoDB query",
			Cause:   err,
		}
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return convertMongoError(err)
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, dest)
}

// RawExec executes a raw MongoDB operation
func (r *Repository) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	// For MongoDB, this could be used for aggregations or other operations
	// This is a simplified implementation
	return &Result{rowsAffected: 0}, nil
}

// GetEntityInfo returns metadata about the entity
func (r *Repository) GetEntityInfo(entity interface{}) (*gpa.EntityInfo, error) {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	info := &gpa.EntityInfo{
		Name:       entityType.Name(),
		TableName:  r.getCollectionName(),
		Fields:     make([]gpa.FieldInfo, 0),
		PrimaryKey: []string{"_id"},
	}

	// Analyze struct fields
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		fieldInfo := gpa.FieldInfo{
			Name:         field.Name,
			Type:         field.Type,
			DatabaseType: "bson",
			Tag:          string(field.Tag),
		}

		// Check for BSON tags
		if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
			if strings.Contains(bsonTag, "_id") {
				fieldInfo.IsPrimaryKey = true
			}
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	return info, nil
}

// Close closes the repository (no-op for MongoDB)
func (r *Repository) Close() error {
	return nil
}
