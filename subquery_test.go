package gpa

import (
	"testing"
)

// TestSubQueryBuilder tests the subquery builder functions
func TestSubQueryBuilder(t *testing.T) {
	// Test basic subquery creation
	subQ := NewSubQuery("SELECT id FROM users WHERE status = ?", "active")
	if subQ.Query != "SELECT id FROM users WHERE status = ?" {
		t.Errorf("Expected query to be 'SELECT id FROM users WHERE status = ?', got %s", subQ.Query)
	}
	if len(subQ.Args) != 1 || subQ.Args[0] != "active" {
		t.Errorf("Expected args to be ['active'], got %v", subQ.Args)
	}
	if subQ.Type != SubQueryTypeScalar {
		t.Errorf("Expected type to be SubQueryTypeScalar, got %s", subQ.Type)
	}
}

// TestExistsSubQuery tests EXISTS subquery creation
func TestExistsSubQuery(t *testing.T) {
	option := ExistsSubQuery("SELECT 1 FROM orders WHERE user_id = users.id")
	
	// Apply to a query to test
	query := &Query{}
	option.Apply(query)
	
	if len(query.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(query.Conditions))
	}
	
	if len(query.SubQueries) != 1 {
		t.Errorf("Expected 1 subquery, got %d", len(query.SubQueries))
	}
	
	subQuery := query.SubQueries[0]
	if subQuery.Type != SubQueryTypeExists {
		t.Errorf("Expected SubQueryTypeExists, got %s", subQuery.Type)
	}
	if subQuery.Operator != OpExists {
		t.Errorf("Expected OpExists, got %s", subQuery.Operator)
	}
}

// TestNotExistsSubQuery tests NOT EXISTS subquery creation
func TestNotExistsSubQuery(t *testing.T) {
	option := NotExistsSubQuery("SELECT 1 FROM orders WHERE user_id = users.id")
	
	query := &Query{}
	option.Apply(query)
	
	if len(query.SubQueries) != 1 {
		t.Errorf("Expected 1 subquery, got %d", len(query.SubQueries))
	}
	
	subQuery := query.SubQueries[0]
	if subQuery.Type != SubQueryTypeExists {
		t.Errorf("Expected SubQueryTypeExists, got %s", subQuery.Type)
	}
	if subQuery.Operator != OpNotExists {
		t.Errorf("Expected OpNotExists, got %s", subQuery.Operator)
	}
}

// TestInSubQuery tests IN subquery creation
func TestInSubQuery(t *testing.T) {
	option := InSubQuery("user_id", "SELECT id FROM active_users", "premium")
	
	query := &Query{}
	option.Apply(query)
	
	if len(query.SubQueries) != 1 {
		t.Errorf("Expected 1 subquery, got %d", len(query.SubQueries))
	}
	
	subQuery := query.SubQueries[0]
	if subQuery.Type != SubQueryTypeIn {
		t.Errorf("Expected SubQueryTypeIn, got %s", subQuery.Type)
	}
	if subQuery.Operator != OpInSubQuery {
		t.Errorf("Expected OpInSubQuery, got %s", subQuery.Operator)
	}
	if subQuery.Field != "user_id" {
		t.Errorf("Expected field to be 'user_id', got %s", subQuery.Field)
	}
}

// TestNotInSubQuery tests NOT IN subquery creation
func TestNotInSubQuery(t *testing.T) {
	option := NotInSubQuery("user_id", "SELECT id FROM banned_users")
	
	query := &Query{}
	option.Apply(query)
	
	subQuery := query.SubQueries[0]
	if subQuery.Type != SubQueryTypeIn {
		t.Errorf("Expected SubQueryTypeIn, got %s", subQuery.Type)
	}
	if subQuery.Operator != OpNotInSubQuery {
		t.Errorf("Expected OpNotInSubQuery, got %s", subQuery.Operator)
	}
}

// TestWhereSubQuery tests scalar subquery creation
func TestWhereSubQuery(t *testing.T) {
	option := WhereSubQuery("price", OpGreaterThan, "SELECT AVG(price) FROM products WHERE category = ?", "electronics")
	
	query := &Query{}
	option.Apply(query)
	
	subQuery := query.SubQueries[0]
	if subQuery.Type != SubQueryTypeScalar {
		t.Errorf("Expected SubQueryTypeScalar, got %s", subQuery.Type)
	}
	if subQuery.Operator != OpGreaterThan {
		t.Errorf("Expected OpGreaterThan, got %s", subQuery.Operator)
	}
	if subQuery.Field != "price" {
		t.Errorf("Expected field to be 'price', got %s", subQuery.Field)
	}
}

// TestCorrelatedSubQuery tests correlated subquery creation
func TestCorrelatedSubQuery(t *testing.T) {
	option := CorrelatedSubQuery("user_id", OpExists, "SELECT 1 FROM orders o WHERE o.user_id = users.id AND o.status = ?", "completed")
	
	query := &Query{}
	option.Apply(query)
	
	subQuery := query.SubQueries[0]
	if subQuery.Type != SubQueryTypeCorrelated {
		t.Errorf("Expected SubQueryTypeCorrelated, got %s", subQuery.Type)
	}
	if !subQuery.IsCorrelated {
		t.Errorf("Expected IsCorrelated to be true")
	}
}

// TestSubQueryConditionString tests the String() method of SubQueryCondition
func TestSubQueryConditionString(t *testing.T) {
	tests := []struct {
		name     string
		subQuery SubQuery
		expected string
	}{
		{
			name: "EXISTS subquery",
			subQuery: SubQuery{
				Query:    "SELECT 1 FROM orders WHERE user_id = users.id",
				Type:     SubQueryTypeExists,
				Operator: OpExists,
			},
			expected: "EXISTS (SELECT 1 FROM orders WHERE user_id = users.id)",
		},
		{
			name: "IN subquery",
			subQuery: SubQuery{
				Query:    "SELECT id FROM active_users",
				Type:     SubQueryTypeIn,
				Field:    "user_id",
				Operator: OpInSubQuery,
			},
			expected: "user_id IN (SELECT id FROM active_users)",
		},
		{
			name: "Scalar subquery",
			subQuery: SubQuery{
				Query:    "SELECT AVG(price) FROM products",
				Type:     SubQueryTypeScalar,
				Field:    "price",
				Operator: OpGreaterThan,
			},
			expected: "price > (SELECT AVG(price) FROM products)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := SubQueryCondition{SubQuery: tt.subQuery}
			result := condition.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestSubQueryConditionMethods tests the Condition interface methods
func TestSubQueryConditionMethods(t *testing.T) {
	subQuery := SubQuery{
		Query:    "SELECT id FROM users WHERE status = ?",
		Args:     []interface{}{"active"},
		Type:     SubQueryTypeIn,
		Field:    "user_id",
		Operator: OpInSubQuery,
	}
	condition := SubQueryCondition{SubQuery: subQuery}
	
	if condition.Field() != "user_id" {
		t.Errorf("Expected field to be 'user_id', got %s", condition.Field())
	}
	
	if condition.Operator() != OpInSubQuery {
		t.Errorf("Expected operator to be OpInSubQuery, got %s", condition.Operator())
	}
	
	value := condition.Value()
	if subQueryValue, ok := value.(SubQuery); !ok {
		t.Errorf("Expected value to be SubQuery, got %T", value)
	} else if subQueryValue.Query != subQuery.Query {
		t.Errorf("Expected value query to be %s, got %s", subQuery.Query, subQueryValue.Query)
	}
}