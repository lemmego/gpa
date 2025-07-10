package gpa

import (
	"fmt"
	"strings"
)

// =====================================
// Query Building
// =====================================

// QueryOption interface for building database queries
type QueryOption interface {
	Apply(query *Query)
}

// Query represents a database query
type Query struct {
	Conditions []Condition
	Orders     []Order
	Limit      *int
	Offset     *int
	Fields     []string
	Joins      []JoinClause
	Groups     []string
	Having     []Condition
	Distinct   bool
	Lock       LockType
	Preloads   []string
	SubQueries []SubQuery
}

// Condition represents a query condition
type Condition interface {
	Field() string
	Operator() Operator
	Value() interface{}
	String() string
}

// BasicCondition implements Condition
type BasicCondition struct {
	FieldName string
	Op        Operator
	Val       interface{}
}

func (c BasicCondition) Field() string      { return c.FieldName }
func (c BasicCondition) Operator() Operator { return c.Op }
func (c BasicCondition) Value() interface{} { return c.Val }
func (c BasicCondition) String() string {
	return c.FieldName + " " + string(c.Op) + " ?"
}

// CompositeCondition for AND/OR operations
type CompositeCondition struct {
	Conditions []Condition
	Logic      LogicOperator
}

func (c CompositeCondition) Field() string      { return "" }
func (c CompositeCondition) Operator() Operator { return "" }
func (c CompositeCondition) Value() interface{} { return nil }
func (c CompositeCondition) String() string {
	if len(c.Conditions) == 0 {
		return ""
	}
	
	var parts []string
	for _, cond := range c.Conditions {
		parts = append(parts, cond.String())
	}
	
	return "(" + strings.Join(parts, " "+string(c.Logic)+" ") + ")"
}

// =====================================
// SubQuery Support
// =====================================

// SubQuery represents a subquery in a larger query
type SubQuery struct {
	Query    *Query
	Type     SubQueryType
	Field    string
	Operator Operator
	Args     []interface{}
}

// SubQueryCondition implements Condition for subqueries
type SubQueryCondition struct {
	SubQuery  SubQuery
	FieldName string
}

func (c SubQueryCondition) Field() string      { return c.FieldName }
func (c SubQueryCondition) Operator() Operator { return c.SubQuery.Operator }
func (c SubQueryCondition) Value() interface{} { return c.SubQuery }
func (c SubQueryCondition) String() string {
	switch c.SubQuery.Type {
	case SubQueryExists:
		return fmt.Sprintf("EXISTS (%s)", c.SubQuery.Query)
	case SubQueryNotExists:
		return fmt.Sprintf("NOT EXISTS (%s)", c.SubQuery.Query)
	case SubQueryIn:
		return fmt.Sprintf("%s IN (%s)", c.FieldName, c.SubQuery.Query)
	case SubQueryNotIn:
		return fmt.Sprintf("%s NOT IN (%s)", c.FieldName, c.SubQuery.Query)
	default:
		return fmt.Sprintf("%s %s (%s)", c.FieldName, c.SubQuery.Operator, c.SubQuery.Query)
	}
}

// =====================================
// Query Option Implementations
// =====================================

// ConditionOption implements QueryOption for basic conditions
type ConditionOption struct {
	Condition Condition
}

func (o ConditionOption) Apply(query *Query) {
	query.Conditions = append(query.Conditions, o.Condition)
}

// CompositeConditionOption implements QueryOption for composite conditions
type CompositeConditionOption struct {
	Conditions []Condition
	Logic      LogicOperator
}

func (o CompositeConditionOption) Apply(query *Query) {
	composite := CompositeCondition{
		Conditions: o.Conditions,
		Logic:      o.Logic,
	}
	query.Conditions = append(query.Conditions, composite)
}

// OrderOption implements QueryOption for ordering
type OrderOption struct {
	Order Order
}

func (o OrderOption) Apply(query *Query) {
	query.Orders = append(query.Orders, o.Order)
}

// LimitOption implements QueryOption for limiting results
type LimitOption struct {
	Count int
}

func (o LimitOption) Apply(query *Query) {
	query.Limit = &o.Count
}

// OffsetOption implements QueryOption for result offset
type OffsetOption struct {
	Count int
}

func (o OffsetOption) Apply(query *Query) {
	query.Offset = &o.Count
}

// FieldsOption implements QueryOption for field selection
type FieldsOption struct {
	Fields []string
}

func (o FieldsOption) Apply(query *Query) {
	query.Fields = append(query.Fields, o.Fields...)
}

// JoinOption implements QueryOption for joins
type JoinOption struct {
	Join JoinClause
}

func (o JoinOption) Apply(query *Query) {
	query.Joins = append(query.Joins, o.Join)
}

// GroupByOption implements QueryOption for grouping
type GroupByOption struct {
	Fields []string
}

func (o GroupByOption) Apply(query *Query) {
	query.Groups = append(query.Groups, o.Fields...)
}

// HavingOption implements QueryOption for having conditions
type HavingOption struct {
	Condition Condition
}

func (o HavingOption) Apply(query *Query) {
	query.Having = append(query.Having, o.Condition)
}

// DistinctOption implements QueryOption for distinct results
type DistinctOption struct{}

func (o DistinctOption) Apply(query *Query) {
	query.Distinct = true
}

// LockOption implements QueryOption for row locking
type LockOption struct {
	Type LockType
}

func (o LockOption) Apply(query *Query) {
	query.Lock = o.Type
}

// PreloadOption implements QueryOption for eager loading
type PreloadOption struct {
	Relations []string
}

func (o PreloadOption) Apply(query *Query) {
	query.Preloads = append(query.Preloads, o.Relations...)
}

// SubQueryOption implements QueryOption for subqueries
type SubQueryOption struct {
	SubQuery SubQuery
}

func (o SubQueryOption) Apply(query *Query) {
	query.SubQueries = append(query.SubQueries, o.SubQuery)
	// Also add as a condition
	condition := SubQueryCondition{
		SubQuery:  o.SubQuery,
		FieldName: o.SubQuery.Field,
	}
	query.Conditions = append(query.Conditions, condition)
}

// =====================================
// Query Builder Functions
// =====================================

// Where creates a basic WHERE condition
func Where(field string, operator Operator, value interface{}) QueryOption {
	return ConditionOption{
		Condition: BasicCondition{
			FieldName: field,
			Op:        operator,
			Val:       value,
		},
	}
}

// And creates an AND composite condition
func And(conditions ...Condition) QueryOption {
	return CompositeConditionOption{
		Conditions: conditions,
		Logic:      LogicAnd,
	}
}

// Or creates an OR composite condition
func Or(conditions ...Condition) QueryOption {
	return CompositeConditionOption{
		Conditions: conditions,
		Logic:      LogicOr,
	}
}

// AndOption creates an AND composite condition
func AndOption(conditions ...Condition) QueryOption {
	return CompositeConditionOption{
		Conditions: conditions,
		Logic:      LogicAnd,
	}
}

// OrOption creates an OR composite condition
func OrOption(conditions ...Condition) QueryOption {
	return CompositeConditionOption{
		Conditions: conditions,
		Logic:      LogicOr,
	}
}

// WhereCondition creates a where condition from a basic condition
func WhereCondition(field string, operator Operator, value interface{}) Condition {
	return BasicCondition{
		FieldName: field,
		Op:        operator,
		Val:       value,
	}
}

// WhereIn creates a WHERE IN condition
func WhereIn(field string, values []interface{}) QueryOption {
	return ConditionOption{
		Condition: BasicCondition{
			FieldName: field,
			Op:        OpIn,
			Val:       values,
		},
	}
}

// WhereLike creates a WHERE LIKE condition
func WhereLike(field string, value string) QueryOption {
	return ConditionOption{
		Condition: BasicCondition{
			FieldName: field,
			Op:        OpLike,
			Val:       value,
		},
	}
}

// WhereNull creates a WHERE IS NULL condition
func WhereNull(field string) QueryOption {
	return ConditionOption{
		Condition: BasicCondition{
			FieldName: field,
			Op:        OpIsNull,
			Val:       nil,
		},
	}
}

// WhereNotNull creates a WHERE IS NOT NULL condition
func WhereNotNull(field string) QueryOption {
	return ConditionOption{
		Condition: BasicCondition{
			FieldName: field,
			Op:        OpIsNotNull,
			Val:       nil,
		},
	}
}

// OrderBy creates an ordering option
func OrderBy(field string, direction OrderDirection) QueryOption {
	return OrderOption{
		Order: Order{
			Field:     field,
			Direction: direction,
		},
	}
}

// Limit creates a limit option
func Limit(count int) QueryOption {
	return LimitOption{Count: count}
}

// Offset creates an offset option
func Offset(count int) QueryOption {
	return OffsetOption{Count: count}
}

// Fields creates a field selection option
func Fields(fields ...string) QueryOption {
	return FieldsOption{Fields: fields}
}

// Select creates a field selection option (alias for Fields)
func Select(fields ...string) QueryOption {
	return FieldsOption{Fields: fields}
}

// Join creates a join option
func Join(joinType JoinType, table string, condition string, alias ...string) QueryOption {
	join := JoinClause{
		Type:      joinType,
		Table:     table,
		Condition: condition,
	}
	if len(alias) > 0 {
		join.Alias = alias[0]
	}
	return JoinOption{Join: join}
}

// InnerJoin creates an INNER JOIN
func InnerJoin(table string, condition string) QueryOption {
	return Join(JoinInner, table, condition)
}

// LeftJoin creates a LEFT JOIN
func LeftJoin(table string, condition string) QueryOption {
	return Join(JoinLeft, table, condition)
}

// GroupBy creates a group by option
func GroupBy(fields ...string) QueryOption {
	return GroupByOption{Fields: fields}
}

// Having creates a having condition option
func Having(field string, operator Operator, value interface{}) QueryOption {
	return HavingOption{
		Condition: BasicCondition{
			FieldName: field,
			Op:        operator,
			Val:       value,
		},
	}
}

// Distinct creates a distinct option
func Distinct() QueryOption {
	return DistinctOption{}
}

// Lock creates a lock option
func Lock(lockType LockType) QueryOption {
	return LockOption{Type: lockType}
}

// Preload creates a preload option for eager loading
func Preload(relations ...string) QueryOption {
	return PreloadOption{Relations: relations}
}

// =====================================
// SubQuery Builder Functions
// =====================================

// ExistsSubQuery creates an EXISTS subquery condition
func ExistsSubQuery(subQuery *Query) QueryOption {
	return SubQueryOption{
		SubQuery: SubQuery{
			Query:    subQuery,
			Type:     SubQueryExists,
			Operator: OpExists,
		},
	}
}

// NotExistsSubQuery creates a NOT EXISTS subquery condition
func NotExistsSubQuery(subQuery *Query) QueryOption {
	return SubQueryOption{
		SubQuery: SubQuery{
			Query:    subQuery,
			Type:     SubQueryNotExists,
			Operator: OpNotExists,
		},
	}
}

// InSubQuery creates an IN subquery condition
func InSubQuery(field string, subQuery *Query) QueryOption {
	return SubQueryOption{
		SubQuery: SubQuery{
			Query:    subQuery,
			Type:     SubQueryIn,
			Field:    field,
			Operator: OpIn,
		},
	}
}

// NotInSubQuery creates a NOT IN subquery condition
func NotInSubQuery(field string, subQuery *Query) QueryOption {
	return SubQueryOption{
		SubQuery: SubQuery{
			Query:    subQuery,
			Type:     SubQueryNotIn,
			Field:    field,
			Operator: OpNotIn,
		},
	}
}

// ScalarSubQuery creates a scalar subquery condition
func ScalarSubQuery(field string, operator Operator, subQuery *Query) QueryOption {
	return SubQueryOption{
		SubQuery: SubQuery{
			Query:    subQuery,
			Type:     SubQueryScalar,
			Field:    field,
			Operator: operator,
		},
	}
}

// WhereSubQuery creates a general subquery condition
func WhereSubQuery(field string, operator Operator, subQuery *Query) QueryOption {
	return ConditionOption{
		Condition: SubQueryCondition{
			SubQuery: SubQuery{
				Query:    subQuery,
				Type:     SubQueryScalar,
				Field:    field,
				Operator: operator,
			},
			FieldName: field,
		},
	}
}

// CorrelatedSubQuery creates a correlated subquery
func CorrelatedSubQuery(field string, operator Operator, subQuery *Query, correlationField string) QueryOption {
	// Add correlation condition to subquery
	subQuery.Conditions = append(subQuery.Conditions, BasicCondition{
		FieldName: correlationField,
		Op:        OpEqual,
		Val:       fmt.Sprintf("{{PARENT.%s}}", field), // Placeholder for parent field
	})
	
	return WhereSubQuery(field, operator, subQuery)
}

// NewQuery creates a new empty query
func NewQuery() *Query {
	return &Query{
		Conditions: make([]Condition, 0),
		Orders:     make([]Order, 0),
		Fields:     make([]string, 0),
		Joins:      make([]JoinClause, 0),
		Groups:     make([]string, 0),
		Having:     make([]Condition, 0),
		Preloads:   make([]string, 0),
		SubQueries: make([]SubQuery, 0),
		Lock:       LockNone,
	}
}

// String returns a string representation of the query
func (q *Query) String() string {
	if q == nil {
		return ""
	}
	return "<Query>" // Placeholder - specific providers should implement proper SQL generation
}