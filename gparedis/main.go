// Package gparedis provides a Redis adapter for the Go Persistence API (GPA)
package gparedis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lemmego/gpa"
)

// =====================================
// Provider Implementation
// =====================================

// Provider implements gpa.Provider using Redis
type Provider struct {
	client *redis.Client
	config gpa.Config
}

// Factory implements gpa.ProviderFactory
type Factory struct{}

// Create creates a new Redis provider instance
func (f *Factory) Create(config gpa.Config) (gpa.Provider, error) {
	provider := &Provider{config: config}

	// Build Redis connection options
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       0, // Default database
	}

	// Parse database number if provided
	if config.Database != "" {
		if db, err := strconv.Atoi(config.Database); err == nil {
			opts.DB = db
		}
	}

	// Configure connection pool
	if config.MaxOpenConns > 0 {
		opts.PoolSize = config.MaxOpenConns
	}
	if config.MaxIdleConns > 0 {
		opts.MinIdleConns = config.MaxIdleConns
	}
	// Note: Redis client doesn't have direct equivalents for ConnMaxLifetime and ConnMaxIdleTime
	// These are handled internally by the Redis client

	// Apply Redis-specific options
	if options, ok := config.Options["redis"]; ok {
		if redisOpts, ok := options.(map[string]interface{}); ok {
			if dialTimeout, ok := redisOpts["dial_timeout"].(time.Duration); ok {
				opts.DialTimeout = dialTimeout
			}
			if readTimeout, ok := redisOpts["read_timeout"].(time.Duration); ok {
				opts.ReadTimeout = readTimeout
			}
			if writeTimeout, ok := redisOpts["write_timeout"].(time.Duration); ok {
				opts.WriteTimeout = writeTimeout
			}
		}
	}

	// Create Redis client
	provider.client = redis.NewClient(opts)

	// Test connection
	if err := provider.Health(); err != nil {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeConnection,
			Message: "failed to connect to Redis",
			Cause:   err,
		}
	}

	return provider, nil
}

// SupportedDrivers returns the list of supported Redis drivers
func (f *Factory) SupportedDrivers() []string {
	return []string{"redis"}
}

// Repository returns a repository for the given entity type
func (p *Provider) Repository(entityType reflect.Type) gpa.Repository {
	return &Repository{
		provider:   p,
		client:     p.client,
		entityType: entityType,
		keyPrefix:  strings.ToLower(entityType.Name()),
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

// Configure applies configuration to the provider
func (p *Provider) Configure(config gpa.Config) error {
	p.config = config
	return nil
}

// Health checks the connection to Redis
func (p *Provider) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return p.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (p *Provider) Close() error {
	return p.client.Close()
}

// SupportedFeatures returns the features supported by Redis
func (p *Provider) SupportedFeatures() []gpa.Feature {
	return []gpa.Feature{
		gpa.FeaturePubSub,
		gpa.FeatureIndexing,
		gpa.FeatureStreaming,
		gpa.FeatureReplication,
	}
}

// ProviderInfo returns information about the Redis provider
func (p *Provider) ProviderInfo() gpa.ProviderInfo {
	return gpa.ProviderInfo{
		Name:         "Redis",
		Version:      "7.0",
		DatabaseType: gpa.DatabaseTypeKV,
		Features:     p.SupportedFeatures(),
	}
}

// =====================================
// Repository Implementation
// =====================================

// Repository implements gpa.Repository and gpa.KeyValueRepository using Redis
type Repository struct {
	provider   *Provider
	client     *redis.Client
	entityType reflect.Type
	keyPrefix  string
}

// =====================================
// Key-Value Operations (KeyValueRepository interface)
// =====================================

// Get retrieves a value by key
func (r *Repository) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.buildKey(key)
	result := r.client.Get(ctx, fullKey)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeNotFound,
				Message: fmt.Sprintf("key not found: %s", key),
			}
		}
		return convertRedisError(err)
	}

	data, err := result.Bytes()
	if err != nil {
		return convertRedisError(err)
	}

	return json.Unmarshal(data, dest)
}

// Set stores a value with a key (BasicKeyValueRepository interface)
func (r *Repository) Set(ctx context.Context, key string, value interface{}) error {
	return r.SetWithTTL(ctx, key, value, 0)
}

// SetWithTTL stores a value with a key and TTL (TTLKeyValueRepository interface)
func (r *Repository) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := r.buildKey(key)
	
	data, err := json.Marshal(value)
	if err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeSerialization,
			Message: "failed to serialize value",
			Cause:   err,
		}
	}

	return convertRedisError(r.client.Set(ctx, fullKey, data, ttl).Err())
}

// DeleteKey removes a key (KeyValueRepository-style method)
func (r *Repository) DeleteKey(ctx context.Context, key string) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.Del(ctx, fullKey).Err())
}

// ExistsKey checks if a key exists (KeyValueRepository-style method)
func (r *Repository) ExistsKey(ctx context.Context, key string) (bool, error) {
	fullKey := r.buildKey(key)
	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, convertRedisError(err)
	}
	return count > 0, nil
}

// MGet retrieves multiple values by keys
func (r *Repository) MGet(ctx context.Context, keys []string, dest interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	values, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return convertRedisError(err)
	}

	results := make([]interface{}, 0, len(values))
	for _, val := range values {
		if val != nil {
			var item interface{}
			if strVal, ok := val.(string); ok {
				if err := json.Unmarshal([]byte(strVal), &item); err == nil {
					results = append(results, item)
				}
			}
		}
	}

	// Set the results to the destination
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "dest must be a pointer to a slice",
		}
	}

	sliceValue := destValue.Elem()
	sliceType := sliceValue.Type().Elem()
	
	for _, result := range results {
		itemValue := reflect.ValueOf(result)
		if itemValue.Type().ConvertibleTo(sliceType) {
			sliceValue = reflect.Append(sliceValue, itemValue.Convert(sliceType))
		}
	}
	
	destValue.Elem().Set(sliceValue)
	return nil
}

// MSet stores multiple key-value pairs (BatchKeyValueRepository interface)
func (r *Repository) MSet(ctx context.Context, pairs map[string]interface{}) error {
	return r.MSetWithTTL(ctx, pairs, 0)
}

// MSetWithTTL stores multiple key-value pairs with TTL
func (r *Repository) MSetWithTTL(ctx context.Context, pairs map[string]interface{}, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}

	// Use pipeline for better performance
	pipe := r.client.Pipeline()
	
	for key, value := range pairs {
		fullKey := r.buildKey(key)
		data, err := json.Marshal(value)
		if err != nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeSerialization,
				Message: fmt.Sprintf("failed to serialize value for key %s", key),
				Cause:   err,
			}
		}
		pipe.Set(ctx, fullKey, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	return convertRedisError(err)
}

// MDelete removes multiple keys
func (r *Repository) MDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	return convertRedisError(r.client.Del(ctx, fullKeys...).Err())
}

// Increment increments a numeric value
func (r *Repository) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := r.buildKey(key)
	result, err := r.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return result, nil
}

// Decrement decrements a numeric value
func (r *Repository) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := r.buildKey(key)
	result, err := r.client.DecrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return result, nil
}

// Expire sets TTL for a key
func (r *Repository) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.Expire(ctx, fullKey, ttl).Err())
}

// TTL returns the TTL of a key
func (r *Repository) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.buildKey(key)
	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return ttl, nil
}

// Keys returns keys matching a pattern
func (r *Repository) Keys(ctx context.Context, pattern string) ([]string, error) {
	fullPattern := r.buildKey(pattern)
	keys, err := r.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		return nil, convertRedisError(err)
	}

	// Remove prefix from returned keys
	result := make([]string, len(keys))
	prefix := r.keyPrefix + ":"
	for i, key := range keys {
		result[i] = strings.TrimPrefix(key, prefix)
	}
	
	return result, nil
}

// Scan scans keys matching a pattern with cursor-based pagination
func (r *Repository) Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	fullPattern := r.buildKey(pattern)
	keys, newCursor, err := r.client.Scan(ctx, cursor, fullPattern, count).Result()
	if err != nil {
		return nil, 0, convertRedisError(err)
	}

	// Remove prefix from returned keys
	result := make([]string, len(keys))
	prefix := r.keyPrefix + ":"
	for i, key := range keys {
		result[i] = strings.TrimPrefix(key, prefix)
	}
	
	return result, newCursor, nil
}

// =====================================
// KeyValueRepository Interface Adapter
// =====================================

// AsKeyValue returns the repository as a KeyValueRepository interface
func (r *Repository) AsKeyValue() gpa.KeyValueRepository {
	return &KeyValueAdapter{r}
}

// KeyValueAdapter adapts Repository to KeyValueRepository interface
type KeyValueAdapter struct {
	*Repository
}

// Delete implements KeyValueRepository.Delete with string key
func (kv *KeyValueAdapter) Delete(ctx context.Context, key string) error {
	return kv.Repository.DeleteKey(ctx, key)
}

// Exists implements KeyValueRepository.Exists with string key  
func (kv *KeyValueAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return kv.Repository.ExistsKey(ctx, key)
}

// =====================================
// Basic Repository Operations
// =====================================

// Create stores an entity using its ID as the key
func (r *Repository) Create(ctx context.Context, entity interface{}) error {
	id, err := r.extractID(entity)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%v", id)
	return r.Set(ctx, key, entity) // No TTL for created entities
}

// CreateBatch creates multiple entities
func (r *Repository) CreateBatch(ctx context.Context, entities interface{}) error {
	entitiesValue := reflect.ValueOf(entities)
	if entitiesValue.Kind() != reflect.Slice {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "entities must be a slice",
		}
	}

	pairs := make(map[string]interface{})
	for i := 0; i < entitiesValue.Len(); i++ {
		entity := entitiesValue.Index(i).Interface()
		id, err := r.extractID(entity)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%v", id)
		pairs[key] = entity
	}

	return r.MSet(ctx, pairs)
}

// FindByID retrieves an entity by its ID
func (r *Repository) FindByID(ctx context.Context, id interface{}, dest interface{}) error {
	key := fmt.Sprintf("%v", id)
	return r.Get(ctx, key, dest)
}

// buildKey constructs the full Redis key with prefix
func (r *Repository) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", r.keyPrefix, key)
}

// extractID extracts the ID field from an entity
func (r *Repository) extractID(entity interface{}) (interface{}, error) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if entityValue.Kind() != reflect.Struct {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "entity must be a struct",
		}
	}

	// Look for ID field (case-insensitive)
	entityType := entityValue.Type()
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if strings.ToLower(field.Name) == "id" {
			return entityValue.Field(i).Interface(), nil
		}
	}

	return nil, gpa.GPAError{
		Type:    gpa.ErrorTypeInvalidArgument,
		Message: "entity must have an ID field",
	}
}

// =====================================
// Error Conversion
// =====================================

// convertRedisError converts Redis errors to GPA errors
func convertRedisError(err error) error {
	if err == nil {
		return nil
	}

	if err == redis.Nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "key not found",
		}
	}

	return gpa.GPAError{
		Type:    gpa.ErrorTypeDatabase,
		Message: "Redis operation failed",
		Cause:   err,
	}
}