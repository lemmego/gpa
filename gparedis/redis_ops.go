package gparedis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lemmego/gpa"
)

// =====================================
// Redis-Specific Operations
// =====================================

// RedisRepository extends the basic repository with Redis-specific operations
type RedisRepository interface {
	gpa.KeyValueRepository
	
	// List operations
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPush(ctx context.Context, key string, values ...interface{}) error
	LPop(ctx context.Context, key string, dest interface{}) error
	RPop(ctx context.Context, key string, dest interface{}) error
	LRange(ctx context.Context, key string, start, stop int64, dest interface{}) error
	LLen(ctx context.Context, key string) (int64, error)
	
	// Set operations
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SRem(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string, dest interface{}) error
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)
	
	// Hash operations
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGet(ctx context.Context, key, field string, dest interface{}) error
	HGetAll(ctx context.Context, key string, dest interface{}) error
	HDel(ctx context.Context, key string, fields ...string) error
	HExists(ctx context.Context, key, field string) (bool, error)
	HLen(ctx context.Context, key string) (int64, error)
	
	// Sorted Set operations
	ZAdd(ctx context.Context, key string, members ...redis.Z) error
	ZRange(ctx context.Context, key string, start, stop int64, dest interface{}) error
	ZRangeByScore(ctx context.Context, key string, min, max string, dest interface{}) error
	ZRem(ctx context.Context, key string, members ...interface{}) error
	ZScore(ctx context.Context, key string, member interface{}) (float64, error)
	ZCard(ctx context.Context, key string) (int64, error)
	
	// Pub/Sub operations
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) (*redis.PubSub, error)
	
	// Stream operations
	XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error)
	XRead(ctx context.Context, streams map[string]string, count int64, block time.Duration) ([]redis.XStream, error)
}

// =====================================
// List Operations
// =====================================

// LPush pushes elements to the left of a list
func (r *Repository) LPush(ctx context.Context, key string, values ...interface{}) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.LPush(ctx, fullKey, values...).Err())
}

// RPush pushes elements to the right of a list
func (r *Repository) RPush(ctx context.Context, key string, values ...interface{}) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.RPush(ctx, fullKey, values...).Err())
}

// LPop pops an element from the left of a list
func (r *Repository) LPop(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.buildKey(key)
	result, err := r.client.LPop(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeNotFound,
				Message: "list is empty or key does not exist",
			}
		}
		return convertRedisError(err)
	}
	
	return r.parseResult(result, dest)
}

// RPop pops an element from the right of a list
func (r *Repository) RPop(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.buildKey(key)
	result, err := r.client.RPop(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeNotFound,
				Message: "list is empty or key does not exist",
			}
		}
		return convertRedisError(err)
	}
	
	return r.parseResult(result, dest)
}

// LRange returns a range of elements from a list
func (r *Repository) LRange(ctx context.Context, key string, start, stop int64, dest interface{}) error {
	fullKey := r.buildKey(key)
	results, err := r.client.LRange(ctx, fullKey, start, stop).Result()
	if err != nil {
		return convertRedisError(err)
	}
	
	return r.parseResults(results, dest)
}

// LLen returns the length of a list
func (r *Repository) LLen(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)
	length, err := r.client.LLen(ctx, fullKey).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return length, nil
}

// =====================================
// Set Operations
// =====================================

// SAdd adds members to a set
func (r *Repository) SAdd(ctx context.Context, key string, members ...interface{}) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.SAdd(ctx, fullKey, members...).Err())
}

// SRem removes members from a set
func (r *Repository) SRem(ctx context.Context, key string, members ...interface{}) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.SRem(ctx, fullKey, members...).Err())
}

// SMembers returns all members of a set
func (r *Repository) SMembers(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.buildKey(key)
	members, err := r.client.SMembers(ctx, fullKey).Result()
	if err != nil {
		return convertRedisError(err)
	}
	
	return r.parseResults(members, dest)
}

// SIsMember checks if a value is a member of a set
func (r *Repository) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	fullKey := r.buildKey(key)
	memberStr := fmt.Sprintf("%v", member)
	isMember, err := r.client.SIsMember(ctx, fullKey, memberStr).Result()
	if err != nil {
		return false, convertRedisError(err)
	}
	return isMember, nil
}

// SCard returns the cardinality (number of elements) of a set
func (r *Repository) SCard(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)
	cardinality, err := r.client.SCard(ctx, fullKey).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return cardinality, nil
}

// =====================================
// Hash Operations
// =====================================

// HSet sets field-value pairs in a hash
func (r *Repository) HSet(ctx context.Context, key string, values ...interface{}) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.HSet(ctx, fullKey, values...).Err())
}

// HGet gets a field value from a hash
func (r *Repository) HGet(ctx context.Context, key, field string, dest interface{}) error {
	fullKey := r.buildKey(key)
	result, err := r.client.HGet(ctx, fullKey, field).Result()
	if err != nil {
		if err == redis.Nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeNotFound,
				Message: fmt.Sprintf("field %s not found in hash %s", field, key),
			}
		}
		return convertRedisError(err)
	}
	
	return r.parseResult(result, dest)
}

// HGetAll gets all field-value pairs from a hash
func (r *Repository) HGetAll(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.buildKey(key)
	result, err := r.client.HGetAll(ctx, fullKey).Result()
	if err != nil {
		return convertRedisError(err)
	}
	
	return r.parseMapResult(result, dest)
}

// HDel deletes fields from a hash
func (r *Repository) HDel(ctx context.Context, key string, fields ...string) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.HDel(ctx, fullKey, fields...).Err())
}

// HExists checks if a field exists in a hash
func (r *Repository) HExists(ctx context.Context, key, field string) (bool, error) {
	fullKey := r.buildKey(key)
	exists, err := r.client.HExists(ctx, fullKey, field).Result()
	if err != nil {
		return false, convertRedisError(err)
	}
	return exists, nil
}

// HLen returns the number of fields in a hash
func (r *Repository) HLen(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)
	length, err := r.client.HLen(ctx, fullKey).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return length, nil
}

// =====================================
// Sorted Set Operations
// =====================================

// ZAdd adds members with scores to a sorted set
func (r *Repository) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	fullKey := r.buildKey(key)
	// Use ZAddArgs for the newer Redis client interface
	args := &redis.ZAddArgs{
		Members: members,
	}
	return convertRedisError(r.client.ZAddArgs(ctx, fullKey, *args).Err())
}

// ZRange returns a range of members from a sorted set by index
func (r *Repository) ZRange(ctx context.Context, key string, start, stop int64, dest interface{}) error {
	fullKey := r.buildKey(key)
	results, err := r.client.ZRange(ctx, fullKey, start, stop).Result()
	if err != nil {
		return convertRedisError(err)
	}
	
	return r.parseResults(results, dest)
}

// ZRangeByScore returns a range of members from a sorted set by score
func (r *Repository) ZRangeByScore(ctx context.Context, key string, min, max string, dest interface{}) error {
	fullKey := r.buildKey(key)
	results, err := r.client.ZRangeByScore(ctx, fullKey, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
	if err != nil {
		return convertRedisError(err)
	}
	
	return r.parseResults(results, dest)
}

// ZRem removes members from a sorted set
func (r *Repository) ZRem(ctx context.Context, key string, members ...interface{}) error {
	fullKey := r.buildKey(key)
	return convertRedisError(r.client.ZRem(ctx, fullKey, members...).Err())
}

// ZScore returns the score of a member in a sorted set
func (r *Repository) ZScore(ctx context.Context, key string, member interface{}) (float64, error) {
	fullKey := r.buildKey(key)
	memberStr := fmt.Sprintf("%v", member)
	score, err := r.client.ZScore(ctx, fullKey, memberStr).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, gpa.GPAError{
				Type:    gpa.ErrorTypeNotFound,
				Message: fmt.Sprintf("member %v not found in sorted set %s", member, key),
			}
		}
		return 0, convertRedisError(err)
	}
	return score, nil
}

// ZCard returns the cardinality of a sorted set
func (r *Repository) ZCard(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)
	cardinality, err := r.client.ZCard(ctx, fullKey).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}
	return cardinality, nil
}

// =====================================
// Pub/Sub Operations
// =====================================

// Publish publishes a message to a channel
func (r *Repository) Publish(ctx context.Context, channel string, message interface{}) error {
	return convertRedisError(r.client.Publish(ctx, channel, message).Err())
}

// Subscribe subscribes to channels
func (r *Repository) Subscribe(ctx context.Context, channels ...string) (*redis.PubSub, error) {
	pubsub := r.client.Subscribe(ctx, channels...)
	
	// Test the subscription
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return nil, convertRedisError(err)
	}
	
	return pubsub, nil
}

// =====================================
// Stream Operations
// =====================================

// XAdd adds an entry to a stream
func (r *Repository) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	fullKey := r.buildKey(stream)
	
	// Convert values to the format expected by Redis
	args := &redis.XAddArgs{
		Stream: fullKey,
		Values: values,
	}
	
	id, err := r.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", convertRedisError(err)
	}
	
	return id, nil
}

// XRead reads entries from streams
func (r *Repository) XRead(ctx context.Context, streams map[string]string, count int64, block time.Duration) ([]redis.XStream, error) {
	// Build full stream names
	fullStreams := make(map[string]string)
	for stream, id := range streams {
		fullStreams[r.buildKey(stream)] = id
	}
	
	// Convert map to slice format expected by XReadArgs
	streamSlice := make([]string, 0, len(fullStreams)*2)
	for stream, id := range fullStreams {
		streamSlice = append(streamSlice, stream, id)
	}
	
	args := &redis.XReadArgs{
		Streams: streamSlice,
		Count:   count,
		Block:   block,
	}
	
	results, err := r.client.XRead(ctx, args).Result()
	if err != nil {
		return nil, convertRedisError(err)
	}
	
	return results, nil
}

// =====================================
// Helper Functions for Result Parsing
// =====================================

// parseResult parses a single Redis result into the destination
func (r *Repository) parseResult(result string, dest interface{}) error {
	if dest == nil {
		return nil
	}
	
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "dest must be a pointer",
		}
	}
	
	destElem := destValue.Elem()
	switch destElem.Kind() {
	case reflect.String:
		destElem.SetString(result)
	case reflect.Interface:
		destElem.Set(reflect.ValueOf(result))
	default:
		// Try JSON unmarshaling for complex types
		if err := json.Unmarshal([]byte(result), dest); err != nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeSerialization,
				Message: "failed to parse result",
				Cause:   err,
			}
		}
	}
	
	return nil
}

// parseResults parses multiple Redis results into the destination slice
func (r *Repository) parseResults(results []string, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "dest must be a pointer to a slice",
		}
	}
	
	sliceValue := destValue.Elem()
	sliceType := sliceValue.Type()
	elemType := sliceType.Elem()
	
	newSlice := reflect.MakeSlice(sliceType, len(results), len(results))
	
	for i, result := range results {
		elem := reflect.New(elemType).Interface()
		if err := r.parseResult(result, elem); err != nil {
			return err
		}
		newSlice.Index(i).Set(reflect.ValueOf(elem).Elem())
	}
	
	destValue.Elem().Set(newSlice)
	return nil
}

// parseMapResult parses a Redis hash result into the destination
func (r *Repository) parseMapResult(result map[string]string, dest interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeSerialization,
			Message: "failed to serialize hash result",
			Cause:   err,
		}
	}
	
	if err := json.Unmarshal(data, dest); err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeSerialization,
			Message: "failed to parse hash result",
			Cause:   err,
		}
	}
	
	return nil
}