package gpamongo

import (
	"context"
	"reflect"
	"strings"

	"github.com/lemmego/gpa"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// =====================================
// NoSQL Repository Implementation
// =====================================

// FindByDocument finds entities matching a document
func (r *Repository) FindByDocument(ctx context.Context, document map[string]interface{}, dest interface{}) error {
	collection := r.getCollection()

	cursor, err := collection.Find(ctx, document)
	if err != nil {
		return convertMongoError(err)
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, dest)
}

// UpdateDocument updates an entity with a document
func (r *Repository) UpdateDocument(ctx context.Context, id interface{}, document map[string]interface{}) error {
	collection := r.getCollection()

	// Convert ID to ObjectID if it's a string
	mongoID, err := r.convertToObjectID(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": mongoID}
	update := bson.M{"$set": document}

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

// CreateCollection creates a new collection
func (r *Repository) CreateCollection(ctx context.Context, name string) error {
	return r.database.CreateCollection(ctx, name)
}

// DropCollection drops a collection
func (r *Repository) DropCollection(ctx context.Context, name string) error {
	return r.database.Collection(name).Drop(ctx)
}

// ListCollections lists all collections in the database
func (r *Repository) ListCollections(ctx context.Context) ([]string, error) {
	cursor, err := r.database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, convertMongoError(err)
	}
	return cursor, nil
}

// Aggregate executes an aggregation pipeline
func (r *Repository) Aggregate(ctx context.Context, pipeline []map[string]interface{}, dest interface{}) error {
	collection := r.getCollection()

	// Convert pipeline to mongo.Pipeline
	mongoPipeline := make(mongo.Pipeline, len(pipeline))
	for i, stage := range pipeline {
		// Convert map[string]interface{} to bson.D
		stageDoc := bson.D{}
		for key, value := range stage {
			stageDoc = append(stageDoc, bson.E{Key: key, Value: value})
		}
		mongoPipeline[i] = stageDoc
	}

	cursor, err := collection.Aggregate(ctx, mongoPipeline)
	if err != nil {
		return convertMongoError(err)
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, dest)
}

// =====================================
// MongoDB-Specific Extensions
// =====================================

// FindOneAndUpdate finds and updates a document atomically
func (r *Repository) FindOneAndUpdate(ctx context.Context, filter, update interface{}, dest interface{}) error {
	collection := r.getCollection()
	
	result := collection.FindOneAndUpdate(ctx, filter, update)
	return result.Decode(dest)
}

// FindOneAndDelete finds and deletes a document atomically
func (r *Repository) FindOneAndDelete(ctx context.Context, filter interface{}, dest interface{}) error {
	collection := r.getCollection()
	
	result := collection.FindOneAndDelete(ctx, filter)
	return result.Decode(dest)
}

// CreateIndex creates an index on the collection
func (r *Repository) CreateIndex(ctx context.Context, keys bson.D, unique bool) error {
	collection := r.getCollection()
	
	indexModel := mongo.IndexModel{
		Keys: keys,
	}
	
	if unique {
		indexModel.Options = options.Index().SetUnique(true)
	}
	
	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	return convertMongoError(err)
}

// DropIndex drops an index from the collection
func (r *Repository) DropIndex(ctx context.Context, name string) error {
	collection := r.getCollection()
	_, err := collection.Indexes().DropOne(ctx, name)
	return convertMongoError(err)
}

// ListIndexes lists all indexes on the collection
func (r *Repository) ListIndexes(ctx context.Context) ([]bson.M, error) {
	collection := r.getCollection()
	
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, convertMongoError(err)
	}
	defer cursor.Close(ctx)
	
	var indexes []bson.M
	err = cursor.All(ctx, &indexes)
	return indexes, convertMongoError(err)
}

// BulkWrite performs bulk write operations
func (r *Repository) BulkWrite(ctx context.Context, operations []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	collection := r.getCollection()
	return collection.BulkWrite(ctx, operations)
}

// Watch returns a change stream for the collection
func (r *Repository) Watch(ctx context.Context, pipeline mongo.Pipeline) (*mongo.ChangeStream, error) {
	collection := r.getCollection()
	return collection.Watch(ctx, pipeline)
}

// =====================================
// Utility Methods
// =====================================

// ensureID ensures the entity has an ID
func (r *Repository) ensureID(entity interface{}) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Check for ObjectID field tagged with "_id"
	if idField := r.getFieldByBSONTag(entityValue, "_id"); idField.IsValid() && idField.CanSet() {
		if idField.Type() == reflect.TypeOf(primitive.ObjectID{}) {
			if idField.Interface() == primitive.NilObjectID {
				idField.Set(reflect.ValueOf(primitive.NewObjectID()))
			}
		}
	} else {
		// Fallback: check for ID field by name
		if idField := entityValue.FieldByName("ID"); idField.IsValid() && idField.CanSet() {
			if idField.Type() == reflect.TypeOf(primitive.ObjectID{}) {
				if idField.Interface() == primitive.NilObjectID {
					idField.Set(reflect.ValueOf(primitive.NewObjectID()))
				}
			} else if idField.Kind() == reflect.String && idField.String() == "" {
				// Generate new ObjectID and set as hex string
				newID := primitive.NewObjectID()
				idField.SetString(newID.Hex())
			}
		}
	}
}

// getFieldByBSONTag finds a field by its BSON tag
func (r *Repository) getFieldByBSONTag(entityValue reflect.Value, tag string) reflect.Value {
	entityType := entityValue.Type()
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
			// Split tag by comma to handle ",omitempty" etc.
			tagParts := strings.Split(bsonTag, ",")
			if len(tagParts) > 0 && tagParts[0] == tag {
				return entityValue.Field(i)
			}
		}
	}
	return reflect.Value{}
}

// getFieldValue gets a field value by name or BSON tag
func (r *Repository) getFieldValue(entity interface{}, fieldName string) interface{} {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// First try by field name
	if field := entityValue.FieldByName(fieldName); field.IsValid() {
		return field.Interface()
	}

	// Then try by BSON tag
	if field := r.getFieldByBSONTag(entityValue, fieldName); field.IsValid() {
		return field.Interface()
	}

	return nil
}

// setFieldValue sets a field value by name or BSON tag
func (r *Repository) setFieldValue(entity interface{}, fieldName string, value interface{}) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// First try by field name
	if field := entityValue.FieldByName(fieldName); field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
		return
	}

	// Then try by BSON tag
	if field := r.getFieldByBSONTag(entityValue, fieldName); field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
	}
}

// convertToObjectID converts various ID types to ObjectID
func (r *Repository) convertToObjectID(id interface{}) (primitive.ObjectID, error) {
	switch v := id.(type) {
	case primitive.ObjectID:
		return v, nil
	case string:
		return primitive.ObjectIDFromHex(v)
	default:
		return primitive.NilObjectID, gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "invalid ID type, expected ObjectID or string",
		}
	}
}

// convertToSlice converts interface{} to []interface{}
func (r *Repository) convertToSlice(entities interface{}) ([]interface{}, error) {
	entitiesValue := reflect.ValueOf(entities)
	
	// Handle pointer to slice
	if entitiesValue.Kind() == reflect.Ptr {
		entitiesValue = entitiesValue.Elem()
	}

	if entitiesValue.Kind() != reflect.Slice {
		return nil, gpa.GPAError{
			Type:    gpa.ErrorTypeValidation,
			Message: "entities must be a slice",
		}
	}

	result := make([]interface{}, entitiesValue.Len())
	for i := 0; i < entitiesValue.Len(); i++ {
		result[i] = entitiesValue.Index(i).Interface()
	}

	return result, nil
}