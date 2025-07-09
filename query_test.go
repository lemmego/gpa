package gpa

import (
	"reflect"
	"testing"
)

func TestBasicCondition(t *testing.T) {
	condition := BasicCondition{
		FieldName: "age",
		Op:        OpGreaterThan,
		Val:       18,
	}

	if condition.Field() != "age" {
		t.Errorf("Expected field 'age', got '%s'", condition.Field())
	}
	if condition.Operator() != OpGreaterThan {
		t.Errorf("Expected operator '>', got '%s'", condition.Operator())
	}
	if condition.Value() != 18 {
		t.Errorf("Expected value 18, got %v", condition.Value())
	}
}

func TestCompositeCondition(t *testing.T) {
	left := BasicCondition{FieldName: "age", Op: OpGreaterThan, Val: 18}
	right := BasicCondition{FieldName: "status", Op: OpEqual, Val: "active"}
	
	condition := CompositeCondition{
		Conditions: []Condition{left, right},
		Logic:      LogicAnd,
	}

	if condition.Logic != LogicAnd {
		t.Errorf("Expected logic operator AND, got %s", condition.Logic)
	}
	if len(condition.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(condition.Conditions))
	}
	if condition.Conditions[0] != left {
		t.Error("Expected first condition to match")
	}
	if condition.Conditions[1] != right {
		t.Error("Expected second condition to match")
	}
}

func TestConditionString(t *testing.T) {
	condition := BasicCondition{FieldName: "deleted", Op: OpEqual, Val: true}
	
	expected := "deleted = ?"
	if condition.String() != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, condition.String())
	}
}

func TestSubQuery(t *testing.T) {
	innerQuery := &Query{
		Conditions: []Condition{
			BasicCondition{FieldName: "user_id", Op: OpEqual, Val: "users.id"},
		},
	}
	
	subQuery := SubQuery{
		Type:     SubQueryExists,
		Query:    innerQuery,
		Field:    "posts",
		Operator: OpExists,
		Args:     []interface{}{},
	}

	if subQuery.Type != SubQueryExists {
		t.Errorf("Expected subquery type EXISTS, got %s", subQuery.Type)
	}
	if subQuery.Query == nil {
		t.Error("Expected query to be set")
	}
	if len(subQuery.Args) != 0 {
		t.Errorf("Expected 0 arguments, got %d", len(subQuery.Args))
	}
}

func TestSubQueryCondition(t *testing.T) {
	innerQuery := &Query{
		Conditions: []Condition{
			BasicCondition{FieldName: "active", Op: OpEqual, Val: true},
		},
	}
	
	subQuery := SubQuery{
		Type:     SubQueryIn,
		Query:    innerQuery,
		Field:    "user_id",
		Operator: OpIn,
		Args:     []interface{}{},
	}
	
	condition := SubQueryCondition{
		FieldName: "id",
		SubQuery:  subQuery,
	}

	if condition.Field() != "id" {
		t.Errorf("Expected field 'id', got '%s'", condition.Field())
	}
	if condition.SubQuery.Type != subQuery.Type {
		t.Error("Expected subquery type to match")
	}
}

func TestQuery(t *testing.T) {
	query := &Query{}

	// Test initial state
	if len(query.Conditions) != 0 {
		t.Error("Expected empty conditions initially")
	}
	if len(query.Orders) != 0 {
		t.Error("Expected empty orders initially")
	}
	if query.Limit != nil {
		t.Error("Expected nil limit initially")
	}
	if query.Offset != nil {
		t.Error("Expected nil offset initially")
	}
}

func TestWhere(t *testing.T) {
	option := Where("name", OpEqual, "John")
	query := &Query{}
	option.Apply(query)

	if len(query.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(query.Conditions))
	}

	condition, ok := query.Conditions[0].(BasicCondition)
	if !ok {
		t.Fatal("Expected BasicCondition")
	}

	if condition.Field() != "name" {
		t.Errorf("Expected field 'name', got '%s'", condition.Field())
	}
	if condition.Operator() != OpEqual {
		t.Errorf("Expected operator '=', got '%s'", condition.Operator())
	}
	if condition.Value() != "John" {
		t.Errorf("Expected value 'John', got '%v'", condition.Value())
	}
}

func TestWhereIn(t *testing.T) {
	values := []interface{}{1, 2, 3}
	option := WhereIn("id", values)
	query := &Query{}
	option.Apply(query)

	if len(query.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(query.Conditions))
	}

	condition, ok := query.Conditions[0].(BasicCondition)
	if !ok {
		t.Fatal("Expected BasicCondition")
	}

	if condition.Operator() != OpIn {
		t.Errorf("Expected operator IN, got %s", condition.Operator())
	}
	if !reflect.DeepEqual(condition.Value(), values) {
		t.Errorf("Expected values %v, got %v", values, condition.Value())
	}
}

func TestWhereLike(t *testing.T) {
	option := WhereLike("name", "%John%")
	query := &Query{}
	option.Apply(query)

	condition, ok := query.Conditions[0].(BasicCondition)
	if !ok {
		t.Fatal("Expected BasicCondition")
	}

	if condition.Operator() != OpLike {
		t.Errorf("Expected operator LIKE, got %s", condition.Operator())
	}
	if condition.Value() != "%John%" {
		t.Errorf("Expected value '%%John%%', got '%v'", condition.Value())
	}
}

func TestWhereNull(t *testing.T) {
	option := WhereNull("deleted_at")
	query := &Query{}
	option.Apply(query)

	condition, ok := query.Conditions[0].(BasicCondition)
	if !ok {
		t.Fatal("Expected BasicCondition")
	}

	if condition.Operator() != OpIsNull {
		t.Errorf("Expected operator IS NULL, got %s", condition.Operator())
	}
	if condition.Field() != "deleted_at" {
		t.Errorf("Expected field 'deleted_at', got '%s'", condition.Field())
	}
}

func TestWhereNotNull(t *testing.T) {
	option := WhereNotNull("email")
	query := &Query{}
	option.Apply(query)

	condition, ok := query.Conditions[0].(BasicCondition)
	if !ok {
		t.Fatal("Expected BasicCondition")
	}

	if condition.Operator() != OpIsNotNull {
		t.Errorf("Expected operator IS NOT NULL, got %s", condition.Operator())
	}
	if condition.Field() != "email" {
		t.Errorf("Expected field 'email', got '%s'", condition.Field())
	}
}

func TestOrderBy(t *testing.T) {
	option := OrderBy("name", OrderAsc)
	query := &Query{}
	option.Apply(query)

	if len(query.Orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(query.Orders))
	}

	order := query.Orders[0]
	if order.Field != "name" {
		t.Errorf("Expected field 'name', got '%s'", order.Field)
	}
	if order.Direction != OrderAsc {
		t.Errorf("Expected direction ASC, got %s", order.Direction)
	}
}

func TestLimit(t *testing.T) {
	option := Limit(10)
	query := &Query{}
	option.Apply(query)

	if query.Limit == nil {
		t.Fatal("Expected limit to be set")
	}
	if *query.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", *query.Limit)
	}
}

func TestOffset(t *testing.T) {
	option := Offset(20)
	query := &Query{}
	option.Apply(query)

	if query.Offset == nil {
		t.Fatal("Expected offset to be set")
	}
	if *query.Offset != 20 {
		t.Errorf("Expected offset 20, got %d", *query.Offset)
	}
}

func TestSelect(t *testing.T) {
	fields := []string{"id", "name", "email"}
	option := Select(fields...)
	query := &Query{}
	option.Apply(query)

	if len(query.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(query.Fields))
	}
	if !reflect.DeepEqual(query.Fields, fields) {
		t.Errorf("Expected fields %v, got %v", fields, query.Fields)
	}
}

func TestGroupBy(t *testing.T) {
	fields := []string{"department", "status"}
	option := GroupBy(fields...)
	query := &Query{}
	option.Apply(query)

	if len(query.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(query.Groups))
	}
	if !reflect.DeepEqual(query.Groups, fields) {
		t.Errorf("Expected groups %v, got %v", fields, query.Groups)
	}
}

func TestHaving(t *testing.T) {
	option := Having("COUNT(*)", OpGreaterThan, 5)
	query := &Query{}
	option.Apply(query)

	if len(query.Having) != 1 {
		t.Errorf("Expected 1 having condition, got %d", len(query.Having))
	}

	condition, ok := query.Having[0].(BasicCondition)
	if !ok {
		t.Fatal("Expected BasicCondition")
	}

	if condition.Field() != "COUNT(*)" {
		t.Errorf("Expected field 'COUNT(*)', got '%s'", condition.Field())
	}
	if condition.Operator() != OpGreaterThan {
		t.Errorf("Expected operator '>', got '%s'", condition.Operator())
	}
}

func TestJoin(t *testing.T) {
	option := Join(JoinLeft, "posts", "users.id = posts.user_id", "p")
	query := &Query{}
	option.Apply(query)

	if len(query.Joins) != 1 {
		t.Errorf("Expected 1 join, got %d", len(query.Joins))
	}

	join := query.Joins[0]
	if join.Type != JoinLeft {
		t.Errorf("Expected LEFT join, got %s", join.Type)
	}
	if join.Table != "posts" {
		t.Errorf("Expected table 'posts', got '%s'", join.Table)
	}
	if join.Condition != "users.id = posts.user_id" {
		t.Errorf("Expected condition to match, got '%s'", join.Condition)
	}
	if join.Alias != "p" {
		t.Errorf("Expected alias 'p', got '%s'", join.Alias)
	}
}

func TestInnerJoin(t *testing.T) {
	option := InnerJoin("profiles", "users.id = profiles.user_id")
	query := &Query{}
	option.Apply(query)

	join := query.Joins[0]
	if join.Type != JoinInner {
		t.Errorf("Expected INNER join, got %s", join.Type)
	}
}

func TestLeftJoin(t *testing.T) {
	option := LeftJoin("posts", "users.id = posts.user_id")
	query := &Query{}
	option.Apply(query)

	join := query.Joins[0]
	if join.Type != JoinLeft {
		t.Errorf("Expected LEFT join, got %s", join.Type)
	}
}

func TestPreload(t *testing.T) {
	relations := []string{"Posts", "Profile", "Comments"}
	option := Preload(relations...)
	query := &Query{}
	option.Apply(query)

	if len(query.Preloads) != 3 {
		t.Errorf("Expected 3 preloads, got %d", len(query.Preloads))
	}
	if !reflect.DeepEqual(query.Preloads, relations) {
		t.Errorf("Expected preloads %v, got %v", relations, query.Preloads)
	}
}

func TestDistinct(t *testing.T) {
	option := Distinct()
	query := &Query{}
	option.Apply(query)

	if !query.Distinct {
		t.Error("Expected distinct to be true")
	}
}

func TestWithSubQuery(t *testing.T) {
	innerQuery := &Query{
		Conditions: []Condition{
			BasicCondition{FieldName: "user_id", Op: OpEqual, Val: "users.id"},
		},
	}
	
	option := WhereSubQuery("id", OpExists, innerQuery)
	query := &Query{}
	option.Apply(query)

	if len(query.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(query.Conditions))
	}

	condition, ok := query.Conditions[0].(SubQueryCondition)
	if !ok {
		t.Fatal("Expected SubQueryCondition")
	}

	if condition.Field() != "id" {
		t.Errorf("Expected field 'id', got '%s'", condition.Field())
	}
}

func TestMultipleQueryOptions(t *testing.T) {
	query := &Query{}
	
	options := []QueryOption{
		Where("status", OpEqual, "active"),
		Where("age", OpGreaterThan, 18),
		OrderBy("name", OrderAsc),
		Limit(10),
		Offset(5),
		Select("id", "name", "email"),
	}
	
	for _, option := range options {
		option.Apply(query)
	}
	
	if len(query.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(query.Conditions))
	}
	if len(query.Orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(query.Orders))
	}
	if query.Limit == nil || *query.Limit != 10 {
		t.Error("Expected limit to be 10")
	}
	if query.Offset == nil || *query.Offset != 5 {
		t.Error("Expected offset to be 5")
	}
	if len(query.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(query.Fields))
	}
}