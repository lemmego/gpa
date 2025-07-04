package gpamongo

import (
	"fmt"
	"strings"

	"github.com/lemmego/gpa"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// =====================================
// Query Building
// =====================================

// buildQuery builds MongoDB filter and find options from GPA query options
func (r *Repository) buildQuery(opts ...gpa.QueryOption) (bson.M, *options.FindOptions) {
	query := &gpa.Query{}

	// Apply all options
	for _, opt := range opts {
		opt.Apply(query)
	}

	// Build filter from conditions
	filter := bson.M{}
	if len(query.Conditions) > 0 {
		filter = r.buildConditions(query.Conditions)
	}

	// Build find options
	findOpts := options.Find()

	// Apply field selection (projection)
	if len(query.Fields) > 0 {
		projection := bson.M{}
		for _, field := range query.Fields {
			projection[r.convertFieldName(field)] = 1
		}
		findOpts.SetProjection(projection)
	}

	// Apply sorting
	if len(query.Orders) > 0 {
		sort := bson.D{}
		for _, order := range query.Orders {
			direction := 1
			if order.Direction == gpa.OrderDesc {
				direction = -1
			}
			sort = append(sort, bson.E{Key: r.convertFieldName(order.Field), Value: direction})
		}
		findOpts.SetSort(sort)
	}

	// Apply limit
	if query.Limit != nil {
		findOpts.SetLimit(int64(*query.Limit))
	}

	// Apply skip (offset)
	if query.Offset != nil {
		findOpts.SetSkip(int64(*query.Offset))
	}

	return filter, findOpts
}

// buildConditions builds MongoDB filter from GPA conditions
func (r *Repository) buildConditions(conditions []gpa.Condition) bson.M {
	if len(conditions) == 0 {
		return bson.M{}
	}

	if len(conditions) == 1 {
		return r.buildCondition(conditions[0])
	}

	// Multiple conditions - combine with AND by default
	var filters []bson.M
	for _, condition := range conditions {
		filters = append(filters, r.buildCondition(condition))
	}

	return bson.M{"$and": filters}
}

// buildCondition builds MongoDB filter from a single GPA condition
func (r *Repository) buildCondition(condition gpa.Condition) bson.M {
	switch cond := condition.(type) {
	case gpa.BasicCondition:
		return r.buildBasicCondition(cond)
	case gpa.CompositeCondition:
		return r.buildCompositeCondition(cond)
	default:
		// Fallback - try to extract field, operator, and value
		field := r.convertFieldName(condition.Field())
		operator := condition.Operator()
		value := condition.Value()
		return r.buildOperatorCondition(field, operator, value)
	}
}

// buildBasicCondition builds MongoDB filter from a basic condition
func (r *Repository) buildBasicCondition(condition gpa.BasicCondition) bson.M {
	field := r.convertFieldName(condition.Field())
	operator := condition.Operator()
	value := condition.Value()

	return r.buildOperatorCondition(field, operator, value)
}

// buildCompositeCondition builds MongoDB filter from a composite condition
func (r *Repository) buildCompositeCondition(condition gpa.CompositeCondition) bson.M {
	if len(condition.Conditions) == 0 {
		return bson.M{}
	}

	var filters []bson.M
	for _, subCondition := range condition.Conditions {
		filters = append(filters, r.buildCondition(subCondition))
	}

	switch condition.Logic {
	case gpa.LogicOr:
		return bson.M{"$or": filters}
	case gpa.LogicAnd:
		return bson.M{"$and": filters}
	case gpa.LogicNot:
		if len(filters) == 1 {
			return bson.M{"$not": filters[0]}
		}
		return bson.M{"$nor": filters}
	default:
		// Default to AND
		return bson.M{"$and": filters}
	}
}

// buildOperatorCondition builds MongoDB filter for a specific operator
func (r *Repository) buildOperatorCondition(field string, operator gpa.Operator, value interface{}) bson.M {
	// Convert string ID values to ObjectID for _id field
	if field == "_id" {
		if strID, ok := value.(string); ok {
			if objID, err := primitive.ObjectIDFromHex(strID); err == nil {
				value = objID
			}
		}
	}

	switch operator {
	case gpa.OpEqual:
		return bson.M{field: value}
	case gpa.OpNotEqual:
		return bson.M{field: bson.M{"$ne": value}}
	case gpa.OpGreaterThan:
		return bson.M{field: bson.M{"$gt": value}}
	case gpa.OpGreaterThanOrEqual:
		return bson.M{field: bson.M{"$gte": value}}
	case gpa.OpLessThan:
		return bson.M{field: bson.M{"$lt": value}}
	case gpa.OpLessThanOrEqual:
		return bson.M{field: bson.M{"$lte": value}}
	case gpa.OpLike:
		// Convert SQL LIKE to MongoDB regex
		regexValue := strings.ReplaceAll(fmt.Sprintf("%v", value), "%", ".*")
		return bson.M{field: bson.M{"$regex": regexValue, "$options": "i"}}
	case gpa.OpNotLike:
		regexValue := strings.ReplaceAll(fmt.Sprintf("%v", value), "%", ".*")
		return bson.M{field: bson.M{"$not": bson.M{"$regex": regexValue, "$options": "i"}}}
	case gpa.OpIn:
		return bson.M{field: bson.M{"$in": value}}
	case gpa.OpNotIn:
		return bson.M{field: bson.M{"$nin": value}}
	case gpa.OpIsNull:
		return bson.M{field: nil}
	case gpa.OpIsNotNull:
		return bson.M{field: bson.M{"$ne": nil}}
	case gpa.OpBetween:
		if values, ok := value.([]interface{}); ok && len(values) == 2 {
			return bson.M{field: bson.M{"$gte": values[0], "$lte": values[1]}}
		}
		return bson.M{}
	case gpa.OpNotBetween:
		if values, ok := value.([]interface{}); ok && len(values) == 2 {
			return bson.M{"$or": []bson.M{
				{field: bson.M{"$lt": values[0]}},
				{field: bson.M{"$gt": values[1]}},
			}}
		}
		return bson.M{}
	case gpa.OpContains:
		return bson.M{field: bson.M{"$regex": fmt.Sprintf(".*%v.*", value), "$options": "i"}}
	case gpa.OpStartsWith:
		return bson.M{field: bson.M{"$regex": fmt.Sprintf("^%v", value), "$options": "i"}}
	case gpa.OpEndsWith:
		return bson.M{field: bson.M{"$regex": fmt.Sprintf("%v$", value), "$options": "i"}}
	case gpa.OpRegex:
		return bson.M{field: bson.M{"$regex": value}}
	default:
		// Fallback to equality
		return bson.M{field: value}
	}
}

// convertFieldName converts GPA field names to MongoDB field names
func (r *Repository) convertFieldName(fieldName string) string {
	// Convert common field mappings
	switch strings.ToLower(fieldName) {
	case "id":
		return "_id"
	default:
		return fieldName
	}
}

// =====================================
// Aggregation Pipeline Builders
// =====================================

// BuildMatchStage creates a $match stage for aggregation
func (r *Repository) BuildMatchStage(conditions []gpa.Condition) bson.M {
	filter := r.buildConditions(conditions)
	return bson.M{"$match": filter}
}

// BuildSortStage creates a $sort stage for aggregation
func (r *Repository) BuildSortStage(orders []gpa.Order) bson.M {
	sort := bson.M{}
	for _, order := range orders {
		direction := 1
		if order.Direction == gpa.OrderDesc {
			direction = -1
		}
		sort[r.convertFieldName(order.Field)] = direction
	}
	return bson.M{"$sort": sort}
}

// BuildLimitStage creates a $limit stage for aggregation
func (r *Repository) BuildLimitStage(limit int) bson.M {
	return bson.M{"$limit": limit}
}

// BuildSkipStage creates a $skip stage for aggregation
func (r *Repository) BuildSkipStage(skip int) bson.M {
	return bson.M{"$skip": skip}
}

// BuildProjectStage creates a $project stage for aggregation
func (r *Repository) BuildProjectStage(fields []string) bson.M {
	projection := bson.M{}
	for _, field := range fields {
		projection[r.convertFieldName(field)] = 1
	}
	return bson.M{"$project": projection}
}

// BuildGroupStage creates a $group stage for aggregation
func (r *Repository) BuildGroupStage(groupBy []string, aggregations map[string]interface{}) bson.M {
	group := bson.M{}
	
	// Build _id for grouping
	if len(groupBy) == 1 {
		group["_id"] = "$" + r.convertFieldName(groupBy[0])
	} else if len(groupBy) > 1 {
		idDoc := bson.M{}
		for _, field := range groupBy {
			idDoc[field] = "$" + r.convertFieldName(field)
		}
		group["_id"] = idDoc
	} else {
		group["_id"] = nil // Group all documents
	}
	
	// Add aggregation functions
	for key, value := range aggregations {
		group[key] = value
	}
	
	return bson.M{"$group": group}
}

// BuildLookupStage creates a $lookup stage for aggregation (joins)
func (r *Repository) BuildLookupStage(from, localField, foreignField, as string) bson.M {
	return bson.M{
		"$lookup": bson.M{
			"from":         from,
			"localField":   r.convertFieldName(localField),
			"foreignField": r.convertFieldName(foreignField),
			"as":           as,
		},
	}
}

// BuildUnwindStage creates an $unwind stage for aggregation
func (r *Repository) BuildUnwindStage(field string, preserveNullAndEmpty bool) bson.M {
	unwind := bson.M{
		"path": "$" + field,
	}
	if preserveNullAndEmpty {
		unwind["preserveNullAndEmptyArrays"] = true
	}
	return bson.M{"$unwind": unwind}
}