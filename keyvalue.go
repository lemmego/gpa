package gpa

import (
	"context"
	"time"
)

// =====================================
// Key-Value Repository Interfaces
// =====================================

// BasicKeyValueRepository provides basic key-value storage operations.
// Suitable for simple caching and key-value storage scenarios.
type BasicKeyValueRepository[T any] interface {
	Repository[T]

	// Set stores a value with the given key.
	// Overwrites any existing value at that key.
	// Example: err := Set(ctx, "user:123", user)
	Set(ctx context.Context, key string, value *T) error

	// Get retrieves a value by key.
	// Returns the value with compile-time type safety.
	// Returns ErrorTypeNotFound if the key doesn't exist.
	// Example: user, err := Get(ctx, "user:123")
	Get(ctx context.Context, key string) (*T, error)

	// DeleteKey removes a key-value pair.
	// Returns ErrorTypeNotFound if the key doesn't exist.
	// Example: err := DeleteKey(ctx, "user:123")
	DeleteKey(ctx context.Context, key string) error

	// KeyExists checks if a key exists.
	// Returns true if the key exists, false otherwise.
	// Example: exists, err := KeyExists(ctx, "user:123")
	KeyExists(ctx context.Context, key string) (bool, error)
}

// BatchKeyValueRepository extends BasicKeyValueRepository with batch operations.
// Provides efficient bulk operations for better performance.
type BatchKeyValueRepository[T any] interface {
	BasicKeyValueRepository[T]

	// MSet sets multiple key-value pairs in a single operation.
	// More efficient than multiple individual Set calls.
	// Example: err := MSet(ctx, map[string]*T{"user:1": user1, "user:2": user2})
	MSet(ctx context.Context, pairs map[string]*T) error

	// MGet retrieves multiple values by their keys.
	// Returns a map of key-value pairs with compile-time type safety.
	// Missing keys are omitted from the result map.
	// Example: users, err := MGet(ctx, []string{"user:1", "user:2", "user:3"})
	MGet(ctx context.Context, keys []string) (map[string]*T, error)

	// MDelete removes multiple keys in a single operation.
	// More efficient than multiple individual DeleteKey calls.
	// Returns the number of keys that were actually deleted.
	// Example: deleted, err := MDelete(ctx, []string{"user:1", "user:2"})
	MDelete(ctx context.Context, keys []string) (int64, error)
}

// TTLKeyValueRepository extends BasicKeyValueRepository with TTL (Time To Live) support.
// Values automatically expire after a specified duration.
type TTLKeyValueRepository[T any] interface {
	BasicKeyValueRepository[T]

	// SetWithTTL stores a value with an expiration time.
	// The value will be automatically deleted after the TTL expires.
	// Example: err := SetWithTTL(ctx, "session:abc123", session, 30*time.Minute)
	SetWithTTL(ctx context.Context, key string, value *T, ttl time.Duration) error

	// GetTTL returns the remaining time-to-live for a key.
	// Returns 0 if the key doesn't exist or has no TTL.
	// Example: ttl, err := GetTTL(ctx, "session:abc123")
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// SetTTL sets or updates the TTL for an existing key.
	// Returns ErrorTypeNotFound if the key doesn't exist.
	// Example: err := SetTTL(ctx, "session:abc123", 15*time.Minute)
	SetTTL(ctx context.Context, key string, ttl time.Duration) error

	// RemoveTTL removes the TTL from a key, making it persistent.
	// The key will no longer expire automatically.
	// Example: err := RemoveTTL(ctx, "permanent:key")
	RemoveTTL(ctx context.Context, key string) error
}

// IncrementKeyValueRepository provides atomic increment/decrement operations.
// Useful for counters, statistics, and rate limiting.
type IncrementKeyValueRepository interface {
	// Increment atomically increments a numeric value.
	// Creates the key with value 0 if it doesn't exist, then adds the delta.
	// Returns the new value after incrementing.
	// Example: newValue, err := Increment(ctx, "counter:visits", 1)
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Decrement atomically decrements a numeric value.
	// Creates the key with value 0 if it doesn't exist, then subtracts the delta.
	// Returns the new value after decrementing.
	// Example: newValue, err := Decrement(ctx, "counter:items", 1)
	Decrement(ctx context.Context, key string, delta int64) (int64, error)
}

// PatternKeyValueRepository provides pattern-based key operations.
// Useful for finding keys that match certain patterns.
type PatternKeyValueRepository interface {
	// Keys returns all keys matching the given pattern.
	// Uses glob-style patterns (*, ?, [abc], etc.).
	// WARNING: Can be slow with large datasets - use with caution.
	// Example: keys, err := Keys(ctx, "user:*")
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Scan iterates over keys matching a pattern using cursor-based pagination.
	// More efficient than Keys() for large datasets as it doesn't load all keys at once.
	// Returns matching keys and a cursor for the next iteration.
	// Example: keys, cursor, err := Scan(ctx, 0, "user:*", 10)
	Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error)
}

// AdvancedKeyValueRepository combines all key-value capabilities.
// Provides the full spectrum of key-value operations including batching, TTL, atomics, and patterns.
// Only the most advanced KV stores (Redis, Hazelcast) implement this complete interface.
type AdvancedKeyValueRepository[T any] interface {
	BasicKeyValueRepository[T]
	BatchKeyValueRepository[T]
	TTLKeyValueRepository[T]
	IncrementKeyValueRepository
	PatternKeyValueRepository
}
