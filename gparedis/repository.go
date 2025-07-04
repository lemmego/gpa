package gparedis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/lemmego/gpa"
)

// =====================================
// Repository Interface Implementation (continued)
// =====================================

// FindAll retrieves all entities matching the given options
func (r *Repository) FindAll(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	// For Redis, we'll scan all keys with our prefix and return matching entities
	keys, err := r.Keys(ctx, "*")
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		// Set empty slice
		destValue := reflect.ValueOf(dest)
		if destValue.Kind() == reflect.Ptr && destValue.Elem().Kind() == reflect.Slice {
			destValue.Elem().Set(reflect.MakeSlice(destValue.Elem().Type(), 0, 0))
		}
		return nil
	}

	// Get all entities
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	values, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return convertRedisError(err)
	}

	// Parse query options
	query := &gpa.Query{}
	for _, opt := range opts {
		opt.Apply(query)
	}

	// Build results
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

	var results []interface{}
	for _, val := range values {
		if val != nil {
			if strVal, ok := val.(string); ok {
				// Create a new instance of the element type
				elem := reflect.New(elemType).Interface()
				if err := json.Unmarshal([]byte(strVal), elem); err == nil {
					// Apply basic filtering if conditions are present
					if r.matchesConditions(elem, query.Conditions) {
						results = append(results, elem)
					}
				}
			}
		}
	}

	// Apply limit and offset
	start := 0
	end := len(results)
	
	if query.Offset != nil {
		start = int(*query.Offset)
		if start > len(results) {
			start = len(results)
		}
	}
	
	if query.Limit != nil {
		requestedEnd := start + int(*query.Limit)
		if requestedEnd < end {
			end = requestedEnd
		}
	}
	
	if start < end {
		results = results[start:end]
	} else {
		results = nil
	}

	// Set the results
	newSlice := reflect.MakeSlice(sliceType, len(results), len(results))
	for i, result := range results {
		newSlice.Index(i).Set(reflect.ValueOf(result).Elem())
	}
	
	destValue.Elem().Set(newSlice)
	return nil
}

// Update updates an entity
func (r *Repository) Update(ctx context.Context, entity interface{}) error {
	id, err := r.extractID(entity)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%v", id)
	// Check if entity exists
	exists, err := r.ExistsKey(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: fmt.Sprintf("entity with ID %v not found", id),
		}
	}

	return r.Set(ctx, key, entity, 0)
}

// UpdatePartial updates specific fields of an entity
func (r *Repository) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	key := fmt.Sprintf("%v", id)
	
	// Get existing entity
	fullKey := r.buildKey(key)
	result := r.client.Get(ctx, fullKey)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return gpa.GPAError{
				Type:    gpa.ErrorTypeNotFound,
				Message: fmt.Sprintf("entity with ID %v not found", id),
			}
		}
		return convertRedisError(err)
	}

	// Parse existing entity as map
	var entityMap map[string]interface{}
	data, err := result.Bytes()
	if err != nil {
		return convertRedisError(err)
	}
	
	if err := json.Unmarshal(data, &entityMap); err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeSerialization,
			Message: "failed to parse existing entity",
			Cause:   err,
		}
	}

	// Apply updates
	for field, value := range updates {
		entityMap[field] = value
	}

	// Save updated entity
	updatedData, err := json.Marshal(entityMap)
	if err != nil {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeSerialization,
			Message: "failed to serialize updated entity",
			Cause:   err,
		}
	}

	return convertRedisError(r.client.Set(ctx, fullKey, updatedData, 0).Err())
}

// Delete removes an entity by ID (Repository interface)
func (r *Repository) Delete(ctx context.Context, id interface{}) error {
	key := fmt.Sprintf("%v", id)
	fullKey := r.buildKey(key)
	
	deleted, err := r.client.Del(ctx, fullKey).Result()
	if err != nil {
		return convertRedisError(err)
	}
	
	if deleted == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: fmt.Sprintf("entity with ID %v not found", id),
		}
	}
	
	return nil
}

// DeleteByCondition removes entities matching the given condition
func (r *Repository) DeleteByCondition(ctx context.Context, condition gpa.Condition) error {
	// Get all entities first
	keys, err := r.Keys(ctx, "*")
	if err != nil {
		return err
	}

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

	var keysToDelete []string
	for i, val := range values {
		if val != nil {
			if strVal, ok := val.(string); ok {
				var entityMap map[string]interface{}
				if err := json.Unmarshal([]byte(strVal), &entityMap); err == nil {
					if r.matchesCondition(entityMap, condition) {
						keysToDelete = append(keysToDelete, fullKeys[i])
					}
				}
			}
		}
	}

	if len(keysToDelete) > 0 {
		return convertRedisError(r.client.Del(ctx, keysToDelete...).Err())
	}

	return nil
}

// Query performs a query with the given options
func (r *Repository) Query(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	return r.FindAll(ctx, dest, opts...)
}

// QueryOne retrieves a single entity matching the query
func (r *Repository) QueryOne(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	// Create a slice to hold results
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "dest must be a pointer",
		}
	}

	elemType := destType.Elem()
	sliceType := reflect.SliceOf(elemType)
	slice := reflect.New(sliceType).Interface()

	// Add limit of 1 to options
	queryOpts := append(opts, gpa.Limit(1))
	
	if err := r.FindAll(ctx, slice, queryOpts...); err != nil {
		return err
	}

	// Check if we found any results
	sliceValue := reflect.ValueOf(slice).Elem()
	if sliceValue.Len() == 0 {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeNotFound,
			Message: "no entity found matching the query",
		}
	}

	// Set the first result to dest
	reflect.ValueOf(dest).Elem().Set(sliceValue.Index(0))
	return nil
}

// Count returns the number of entities matching the query
func (r *Repository) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	keys, err := r.Keys(ctx, "*")
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Parse query options
	query := &gpa.Query{}
	for _, opt := range opts {
		opt.Apply(query)
	}

	// If no conditions, return total count
	if len(query.Conditions) == 0 {
		return int64(len(keys)), nil
	}

	// Get all entities and count matches
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	values, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return 0, convertRedisError(err)
	}

	var count int64
	for _, val := range values {
		if val != nil {
			if strVal, ok := val.(string); ok {
				var entityMap map[string]interface{}
				if err := json.Unmarshal([]byte(strVal), &entityMap); err == nil {
					if r.matchesConditions(entityMap, query.Conditions) {
						count++
					}
				}
			}
		}
	}

	return count, nil
}

// Exists checks if any entity matches the query (Repository interface)
func (r *Repository) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Transaction is not supported by Redis in the traditional sense
func (r *Repository) Transaction(ctx context.Context, fn gpa.TransactionFunc) error {
	return gpa.GPAError{
		Type:    gpa.ErrorTypeUnsupported,
		Message: "transactions are not supported by Redis adapter",
	}
}

// RawQuery executes a raw Redis command
func (r *Repository) RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error {
	// Parse Redis command
	parts := []interface{}{query}
	parts = append(parts, args...)
	
	result := r.client.Do(ctx, parts...)
	if err := result.Err(); err != nil {
		return convertRedisError(err)
	}

	// Try to set the result
	val, err := result.Result()
	if err != nil {
		return convertRedisError(err)
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return gpa.GPAError{
			Type:    gpa.ErrorTypeInvalidArgument,
			Message: "dest must be a pointer",
		}
	}

	// Convert result based on type
	resultValue := reflect.ValueOf(val)
	if resultValue.Type().AssignableTo(destValue.Elem().Type()) {
		destValue.Elem().Set(resultValue)
		return nil
	}

	return gpa.GPAError{
		Type:    gpa.ErrorTypeInvalidArgument,
		Message: "result type not compatible with destination",
	}
}

// RawExec executes a raw Redis command without returning data
func (r *Repository) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	parts := []interface{}{query}
	parts = append(parts, args...)
	
	result := r.client.Do(ctx, parts...)
	if err := result.Err(); err != nil {
		return nil, convertRedisError(err)
	}

	return &RedisResult{result: result}, nil
}

// GetEntityInfo returns metadata about the entity
func (r *Repository) GetEntityInfo(entity interface{}) (*gpa.EntityInfo, error) {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	return &gpa.EntityInfo{
		Name:       entityType.Name(),
		TableName:  entityType.Name(),
		Fields:     extractFields(entityType),
		PrimaryKey: []string{"ID"}, // Assume ID field
		Indexes:    []gpa.IndexInfo{}, // Redis doesn't have traditional indexes
		Relations:  []gpa.RelationInfo{}, // Redis doesn't support relations
	}, nil
}

// Close closes the repository (no-op for Redis)
func (r *Repository) Close() error {
	return nil
}

// =====================================
// Helper Functions
// =====================================

// matchesConditions checks if an entity matches all conditions
func (r *Repository) matchesConditions(entity interface{}, conditions []gpa.Condition) bool {
	for _, condition := range conditions {
		if !r.matchesCondition(entity, condition) {
			return false
		}
	}
	return true
}

// matchesCondition checks if an entity matches a single condition
func (r *Repository) matchesCondition(entity interface{}, condition gpa.Condition) bool {
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		return r.matchesBasicCondition(entity, cond)
	case gpa.CompositeCondition:
		return r.matchesCompositeCondition(entity, cond)
	default:
		// For complex conditions, we'll just return true for now
		return true
	}
}

// matchesBasicCondition checks if an entity matches a basic condition
func (r *Repository) matchesBasicCondition(entity interface{}, condition gpa.BasicCondition) bool {
	var entityMap map[string]interface{}
	
	// Convert entity to map if it's not already
	if m, ok := entity.(map[string]interface{}); ok {
		entityMap = m
	} else {
		// Convert struct to map via JSON
		data, err := json.Marshal(entity)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(data, &entityMap); err != nil {
			return false
		}
	}

	fieldValue, exists := entityMap[condition.Field()]
	if !exists {
		return false
	}

	return compareValues(fieldValue, condition.Operator(), condition.Value())
}

// matchesCompositeCondition checks if an entity matches a composite condition
func (r *Repository) matchesCompositeCondition(entity interface{}, condition gpa.CompositeCondition) bool {
	switch condition.Logic {
	case gpa.LogicAnd:
		for _, subCondition := range condition.Conditions {
			if !r.matchesCondition(entity, subCondition) {
				return false
			}
		}
		return true
	case gpa.LogicOr:
		for _, subCondition := range condition.Conditions {
			if r.matchesCondition(entity, subCondition) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// compareValues compares two values using the given operator
func compareValues(fieldValue interface{}, operator gpa.Operator, targetValue interface{}) bool {
	switch operator {
	case gpa.OpEqual:
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", targetValue)
	case gpa.OpNotEqual:
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", targetValue)
	case gpa.OpLike:
		fieldStr := fmt.Sprintf("%v", fieldValue)
		targetStr := fmt.Sprintf("%v", targetValue)
		// Simple contains check for LIKE
		return contains(fieldStr, strings.Trim(targetStr, "%"))
	default:
		// For other operators, we'll do string comparison for simplicity
		fieldStr := fmt.Sprintf("%v", fieldValue)
		targetStr := fmt.Sprintf("%v", targetValue)
		
		switch operator {
		case gpa.OpGreaterThan:
			return fieldStr > targetStr
		case gpa.OpLessThan:
			return fieldStr < targetStr
		case gpa.OpGreaterThanOrEqual:
			return fieldStr >= targetStr
		case gpa.OpLessThanOrEqual:
			return fieldStr <= targetStr
		}
	}
	
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// extractFields extracts field information from a struct type
func extractFields(entityType reflect.Type) []gpa.FieldInfo {
	var fields []gpa.FieldInfo
	
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		
		fieldInfo := gpa.FieldInfo{
			Name:         field.Name,
			Type:         field.Type,
			Tag:          string(field.Tag),
			IsPrimaryKey: strings.ToLower(field.Name) == "id",
			IsNullable:   true, // Redis doesn't enforce constraints
		}
		
		fields = append(fields, fieldInfo)
	}
	
	return fields
}

// =====================================
// Redis Result Implementation
// =====================================

// RedisResult implements gpa.Result for Redis operations
type RedisResult struct {
	result *redis.Cmd
}

// LastInsertId returns 0 (not applicable for Redis)
func (r *RedisResult) LastInsertId() (int64, error) {
	return 0, nil
}

// RowsAffected returns the number of affected rows (when applicable)
func (r *RedisResult) RowsAffected() (int64, error) {
	val, err := r.result.Result()
	if err != nil {
		return 0, err
	}
	
	// Try to convert to int64
	if intVal, ok := val.(int64); ok {
		return intVal, nil
	}
	if intVal, ok := val.(int); ok {
		return int64(intVal), nil
	}
	
	return 1, nil // Default to 1 for successful operations
}