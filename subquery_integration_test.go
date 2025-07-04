package gpa

import (
	"testing"
)

// TestSubQueryIntegration tests subquery functionality with different query scenarios
func TestSubQueryIntegration(t *testing.T) {
	// Test complex query building with multiple subqueries
	testCases := []struct {
		name        string
		buildQuery  func() []QueryOption
		expectsSQL  bool // For SQL adapters
		expectsNoSQL bool // For NoSQL adapters
	}{
		{
			name: "EXISTS subquery",
			buildQuery: func() []QueryOption {
				return []QueryOption{
					Where("status", OpEqual, "active"),
					ExistsSubQuery("SELECT 1 FROM orders WHERE orders.user_id = users.id AND orders.status = ?", "completed"),
					OrderBy("name", OrderAsc),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
		{
			name: "NOT EXISTS subquery", 
			buildQuery: func() []QueryOption {
				return []QueryOption{
					NotExistsSubQuery("SELECT 1 FROM banned_users WHERE banned_users.user_id = users.id"),
					Where("created_at", OpGreaterThan, "2023-01-01"),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
		{
			name: "IN subquery",
			buildQuery: func() []QueryOption {
				return []QueryOption{
					InSubQuery("user_id", "SELECT id FROM premium_users WHERE subscription_active = ?", true),
					OrderBy("last_login", OrderDesc),
					Limit(100),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
		{
			name: "NOT IN subquery",
			buildQuery: func() []QueryOption {
				return []QueryOption{
					NotInSubQuery("user_id", "SELECT user_id FROM suspended_accounts"),
					Where("age", OpGreaterThanOrEqual, 18),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
		{
			name: "Scalar subquery comparison",
			buildQuery: func() []QueryOption {
				return []QueryOption{
					WhereSubQuery("order_total", OpGreaterThan, "SELECT AVG(order_total) FROM orders WHERE created_at > ?", "2023-01-01"),
					Where("status", OpEqual, "completed"),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
		{
			name: "Correlated subquery",
			buildQuery: func() []QueryOption {
				return []QueryOption{
					CorrelatedSubQuery("user_id", OpExists, "SELECT 1 FROM user_preferences up WHERE up.user_id = users.id AND up.newsletter = ?", true),
					Where("status", OpEqual, "active"),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
		{
			name: "Complex mixed conditions with subqueries",
			buildQuery: func() []QueryOption {
				return []QueryOption{
					Where("status", OpEqual, "premium"),
					ExistsSubQuery("SELECT 1 FROM subscriptions WHERE subscriptions.user_id = users.id AND subscriptions.active = ?", true),
					InSubQuery("user_id", "SELECT user_id FROM special_promotions WHERE valid_until > NOW()"),
					Where("created_at", OpGreaterThan, "2022-01-01"),
					OrderBy("last_activity", OrderDesc),
					Limit(50),
				}
			},
			expectsSQL: true,
			expectsNoSQL: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build the query
			opts := tc.buildQuery()
			
			// Test that query builds without errors
			query := &Query{}
			for _, opt := range opts {
				opt.Apply(query)
			}
			
			// Verify subqueries were added
			if len(query.SubQueries) == 0 && (tc.expectsSQL || tc.expectsNoSQL) {
				t.Errorf("Expected subqueries to be present, but none found")
			}
			
			// Verify conditions were added
			if len(query.Conditions) == 0 {
				t.Errorf("Expected conditions to be present, but none found")
			}
			
			// Check that subquery conditions have proper types
			for _, condition := range query.Conditions {
				if subQueryCond, ok := condition.(SubQueryCondition); ok {
					if subQueryCond.SubQuery.Query == "" {
						t.Errorf("SubQuery condition has empty query")
					}
					if subQueryCond.SubQuery.Type == "" {
						t.Errorf("SubQuery condition has empty type")
					}
				}
			}
			
			// Verify that we can serialize conditions to strings (useful for debugging)
			for _, condition := range query.Conditions {
				condStr := condition.String()
				if condStr == "" {
					t.Errorf("Condition String() method returned empty string")
				}
			}
		})
	}
}

// TestSubQueryTypes tests all subquery types
func TestSubQueryTypes(t *testing.T) {
	tests := []struct {
		name           string
		createOption   func() QueryOption
		expectedType   SubQueryType
		expectedOp     Operator
		expectedField  string
	}{
		{
			name:          "EXISTS",
			createOption:  func() QueryOption { return ExistsSubQuery("SELECT 1 FROM table") },
			expectedType:  SubQueryTypeExists,
			expectedOp:    OpExists,
			expectedField: "",
		},
		{
			name:          "NOT EXISTS", 
			createOption:  func() QueryOption { return NotExistsSubQuery("SELECT 1 FROM table") },
			expectedType:  SubQueryTypeExists,
			expectedOp:    OpNotExists,
			expectedField: "",
		},
		{
			name:          "IN",
			createOption:  func() QueryOption { return InSubQuery("field", "SELECT id FROM table") },
			expectedType:  SubQueryTypeIn,
			expectedOp:    OpInSubQuery,
			expectedField: "field",
		},
		{
			name:          "NOT IN",
			createOption:  func() QueryOption { return NotInSubQuery("field", "SELECT id FROM table") },
			expectedType:  SubQueryTypeIn,
			expectedOp:    OpNotInSubQuery,
			expectedField: "field",
		},
		{
			name:          "Scalar",
			createOption:  func() QueryOption { return WhereSubQuery("field", OpGreaterThan, "SELECT AVG(col) FROM table") },
			expectedType:  SubQueryTypeScalar,
			expectedOp:    OpGreaterThan,
			expectedField: "field",
		},
		{
			name:          "Correlated",
			createOption:  func() QueryOption { return CorrelatedSubQuery("field", OpExists, "SELECT 1 FROM table t WHERE t.id = main.id") },
			expectedType:  SubQueryTypeCorrelated,
			expectedOp:    OpExists,
			expectedField: "field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := tt.createOption()
			
			query := &Query{}
			option.Apply(query)
			
			if len(query.SubQueries) != 1 {
				t.Fatalf("Expected 1 subquery, got %d", len(query.SubQueries))
			}
			
			subQuery := query.SubQueries[0]
			
			if subQuery.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, subQuery.Type)
			}
			
			if subQuery.Operator != tt.expectedOp {
				t.Errorf("Expected operator %s, got %s", tt.expectedOp, subQuery.Operator)
			}
			
			if subQuery.Field != tt.expectedField {
				t.Errorf("Expected field %s, got %s", tt.expectedField, subQuery.Field)
			}
		})
	}
}

// TestSubQueryWithArgs tests subqueries with arguments
func TestSubQueryWithArgs(t *testing.T) {
	option := InSubQuery("user_id", "SELECT id FROM users WHERE status = ? AND created_at > ?", "active", "2023-01-01")
	
	query := &Query{}
	option.Apply(query)
	
	if len(query.SubQueries) != 1 {
		t.Fatalf("Expected 1 subquery, got %d", len(query.SubQueries))
	}
	
	subQuery := query.SubQueries[0]
	
	if len(subQuery.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(subQuery.Args))
	}
	
	if subQuery.Args[0] != "active" {
		t.Errorf("Expected first arg to be 'active', got %v", subQuery.Args[0])
	}
	
	if subQuery.Args[1] != "2023-01-01" {
		t.Errorf("Expected second arg to be '2023-01-01', got %v", subQuery.Args[1])
	}
}

// TestComplexQueryWithSubQueries tests building complex queries with multiple subqueries
func TestComplexQueryWithSubQueries(t *testing.T) {
	opts := []QueryOption{
		// Main conditions
		Where("status", OpEqual, "active"),
		Where("age", OpGreaterThanOrEqual, 18),
		
		// Subquery conditions
		ExistsSubQuery("SELECT 1 FROM orders WHERE orders.user_id = users.id AND orders.total > ?", 100),
		NotInSubQuery("user_id", "SELECT user_id FROM blacklisted_users"),
		WhereSubQuery("credit_score", OpGreaterThan, "SELECT AVG(credit_score) FROM users WHERE verified = ?", true),
		
		// Additional options
		OrderBy("created_at", OrderDesc),
		Limit(25),
		Offset(50),
	}
	
	query := &Query{}
	for _, opt := range opts {
		opt.Apply(query)
	}
	
	// Verify we have the expected number of conditions (5 total: 2 basic + 3 subqueries)
	if len(query.Conditions) != 5 {
		t.Errorf("Expected 5 conditions, got %d", len(query.Conditions))
	}
	
	// Verify we have 3 subqueries
	if len(query.SubQueries) != 3 {
		t.Errorf("Expected 3 subqueries, got %d", len(query.SubQueries))
	}
	
	// Verify ordering
	if len(query.Orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(query.Orders))
	}
	
	// Verify pagination
	if query.Limit == nil || *query.Limit != 25 {
		t.Errorf("Expected limit to be 25")
	}
	
	if query.Offset == nil || *query.Offset != 50 {
		t.Errorf("Expected offset to be 50")
	}
	
	// Test that all conditions can be converted to strings
	for i, condition := range query.Conditions {
		str := condition.String()
		if str == "" {
			t.Errorf("Condition %d has empty string representation", i)
		}
	}
}