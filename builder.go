package gpa

import "context"

// =====================================
// Query Builder
// =====================================

// QueryBuilder provides a fluent interface for building type-safe database queries.
// Supports method chaining for convenient query construction.
type QueryBuilder[T any] struct {
	conditions []Condition
	orders     []Order
	limit      *int
	offset     *int
	fields     []string
	joins      []JoinClause
	groups     []string
	having     []Condition
	preloads   []string
	distinct   bool
	lock       LockType
}

// NewQueryBuilder creates a new type-safe query builder for entity type T.
// Example: qb := NewQueryBuilder[User]()
func NewQueryBuilder[T any]() *QueryBuilder[T] {
	return &QueryBuilder[T]{
		conditions: make([]Condition, 0),
		orders:     make([]Order, 0),
		fields:     make([]string, 0),
		joins:      make([]JoinClause, 0),
		groups:     make([]string, 0),
		having:     make([]Condition, 0),
		preloads:   make([]string, 0),
		lock:       LockNone,
	}
}

// Where adds a WHERE condition to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Where("age", ">", 18).Where("status", "=", "active")
func (qb *QueryBuilder[T]) Where(field string, operator Operator, value interface{}) *QueryBuilder[T] {
	qb.conditions = append(qb.conditions, BasicCondition{
		FieldName: field,
		Op:        operator,
		Val:       value,
	})
	return qb
}

// WhereCondition adds a custom condition to the query.
// Useful for complex conditions that can't be expressed with simple field/operator/value.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.WhereCondition(customCondition)
func (qb *QueryBuilder[T]) WhereCondition(condition Condition) *QueryBuilder[T] {
	qb.conditions = append(qb.conditions, condition)
	return qb
}

// OrderBy adds an ORDER BY clause to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.OrderBy("name", OrderAsc).OrderBy("created_at", OrderDesc)
func (qb *QueryBuilder[T]) OrderBy(field string, direction OrderDirection) *QueryBuilder[T] {
	qb.orders = append(qb.orders, Order{
		Field:     field,
		Direction: direction,
	})
	return qb
}

// Limit sets the maximum number of results to return.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Limit(10)
func (qb *QueryBuilder[T]) Limit(count int) *QueryBuilder[T] {
	qb.limit = &count
	return qb
}

// Offset sets the number of results to skip.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Offset(20)
func (qb *QueryBuilder[T]) Offset(count int) *QueryBuilder[T] {
	qb.offset = &count
	return qb
}

// Select specifies which fields to include in the results.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Select("id", "name", "email")
func (qb *QueryBuilder[T]) Select(fields ...string) *QueryBuilder[T] {
	qb.fields = append(qb.fields, fields...)
	return qb
}

// Join adds a JOIN clause to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Join(JoinLeft, "orders", "users.id = orders.user_id")
func (qb *QueryBuilder[T]) Join(joinType JoinType, table string, condition string) *QueryBuilder[T] {
	qb.joins = append(qb.joins, JoinClause{
		Type:      joinType,
		Table:     table,
		Condition: condition,
	})
	return qb
}

// GroupBy adds a GROUP BY clause to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.GroupBy("status", "department")
func (qb *QueryBuilder[T]) GroupBy(fields ...string) *QueryBuilder[T] {
	qb.groups = append(qb.groups, fields...)
	return qb
}

// Having adds a HAVING condition to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Having("COUNT(*)", ">", 5)
func (qb *QueryBuilder[T]) Having(field string, operator Operator, value interface{}) *QueryBuilder[T] {
	qb.having = append(qb.having, BasicCondition{
		FieldName: field,
		Op:        operator,
		Val:       value,
	})
	return qb
}

// Distinct adds a DISTINCT clause to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Distinct()
func (qb *QueryBuilder[T]) Distinct() *QueryBuilder[T] {
	qb.distinct = true
	return qb
}

// Lock adds a locking clause to the query.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Lock(LockExclusive)
func (qb *QueryBuilder[T]) Lock(lockType LockType) *QueryBuilder[T] {
	qb.lock = lockType
	return qb
}

// Preload specifies relationships to eagerly load.
// Returns the same QueryBuilder instance for method chaining.
// Example: qb.Preload("Orders", "Profile")
func (qb *QueryBuilder[T]) Preload(relations ...string) *QueryBuilder[T] {
	qb.preloads = append(qb.preloads, relations...)
	return qb
}

// Build converts the QueryBuilder to a Query struct.
// Returns a Query that can be used with repository methods.
// Example: query := qb.Build()
func (qb *QueryBuilder[T]) Build() *Query {
	return &Query{
		Conditions: qb.conditions,
		Orders:     qb.orders,
		Limit:      qb.limit,
		Offset:     qb.offset,
		Fields:     qb.fields,
		Joins:      qb.joins,
		Groups:     qb.groups,
		Having:     qb.having,
		Distinct:   qb.distinct,
		Lock:       qb.lock,
		Preloads:   qb.preloads,
		SubQueries: make([]SubQuery, 0),
	}
}

// Execute executes the query using the provided repository.
// Returns a slice of entity pointers with compile-time type safety.
// Example: users, err := qb.Execute(ctx, repo)
func (qb *QueryBuilder[T]) Execute(ctx context.Context, repo Repository[T]) ([]*T, error) {
	query := qb.Build()
	
	// Convert Query to QueryOptions
	var options []QueryOption
	
	// Add conditions
	for _, condition := range query.Conditions {
		options = append(options, ConditionOption{Condition: condition})
	}
	
	// Add orders
	for _, order := range query.Orders {
		options = append(options, OrderOption{Order: order})
	}
	
	// Add limit
	if query.Limit != nil {
		options = append(options, LimitOption{Count: *query.Limit})
	}
	
	// Add offset
	if query.Offset != nil {
		options = append(options, OffsetOption{Count: *query.Offset})
	}
	
	// Add fields
	if len(query.Fields) > 0 {
		options = append(options, FieldsOption{Fields: query.Fields})
	}
	
	// Add joins
	for _, join := range query.Joins {
		options = append(options, JoinOption{Join: join})
	}
	
	// Add group by
	if len(query.Groups) > 0 {
		options = append(options, GroupByOption{Fields: query.Groups})
	}
	
	// Add having
	for _, having := range query.Having {
		options = append(options, HavingOption{Condition: having})
	}
	
	// Add distinct
	if query.Distinct {
		options = append(options, DistinctOption{})
	}
	
	// Add lock
	if query.Lock != LockNone {
		options = append(options, LockOption{Type: query.Lock})
	}
	
	// Add preloads
	if len(query.Preloads) > 0 {
		options = append(options, PreloadOption{Relations: query.Preloads})
	}
	
	return repo.Query(ctx, options...)
}

// ExecuteOne executes the query and returns a single result.
// Returns the entity directly with compile-time type safety.
// Returns ErrorTypeNotFound if no entity matches the query.
// Example: user, err := qb.ExecuteOne(ctx, repo)
func (qb *QueryBuilder[T]) ExecuteOne(ctx context.Context, repo Repository[T]) (*T, error) {
	query := qb.Build()
	
	// Convert Query to QueryOptions (same as Execute)
	var options []QueryOption
	
	// Add conditions
	for _, condition := range query.Conditions {
		options = append(options, ConditionOption{Condition: condition})
	}
	
	// Add orders
	for _, order := range query.Orders {
		options = append(options, OrderOption{Order: order})
	}
	
	// Add limit (force to 1 for single result)
	options = append(options, LimitOption{Count: 1})
	
	// Add offset
	if query.Offset != nil {
		options = append(options, OffsetOption{Count: *query.Offset})
	}
	
	// Add fields
	if len(query.Fields) > 0 {
		options = append(options, FieldsOption{Fields: query.Fields})
	}
	
	// Add joins
	for _, join := range query.Joins {
		options = append(options, JoinOption{Join: join})
	}
	
	// Add group by
	if len(query.Groups) > 0 {
		options = append(options, GroupByOption{Fields: query.Groups})
	}
	
	// Add having
	for _, having := range query.Having {
		options = append(options, HavingOption{Condition: having})
	}
	
	// Add distinct
	if query.Distinct {
		options = append(options, DistinctOption{})
	}
	
	// Add lock
	if query.Lock != LockNone {
		options = append(options, LockOption{Type: query.Lock})
	}
	
	// Add preloads
	if len(query.Preloads) > 0 {
		options = append(options, PreloadOption{Relations: query.Preloads})
	}
	
	return repo.QueryOne(ctx, options...)
}

// Count executes the query and returns the count of matching entities.
// Returns the count as an int64.
// Example: count, err := qb.Count(ctx, repo)
func (qb *QueryBuilder[T]) Count(ctx context.Context, repo Repository[T]) (int64, error) {
	query := qb.Build()
	
	// Convert Query to QueryOptions (only conditions matter for count)
	var options []QueryOption
	
	// Add conditions
	for _, condition := range query.Conditions {
		options = append(options, ConditionOption{Condition: condition})
	}
	
	// Add joins if they affect the count
	for _, join := range query.Joins {
		options = append(options, JoinOption{Join: join})
	}
	
	// Add group by if it affects the count
	if len(query.Groups) > 0 {
		options = append(options, GroupByOption{Fields: query.Groups})
	}
	
	// Add having if it affects the count
	for _, having := range query.Having {
		options = append(options, HavingOption{Condition: having})
	}
	
	return repo.Count(ctx, options...)
}