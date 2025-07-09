package gpa

import (
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	config := Config{
		Driver:          "postgres",
		Host:            "localhost",
		Port:            5432,
		Database:        "testdb",
		Username:        "user",
		Password:        "pass",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		SSL: SSLConfig{
			Enabled:  true,
			Mode:     "require",
			CertFile: "/path/to/cert",
			KeyFile:  "/path/to/key",
			CAFile:   "/path/to/ca",
		},
		Options: map[string]interface{}{
			"timeout": "30s",
		},
	}

	if config.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got '%s'", config.Driver)
	}
	if config.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", config.Port)
	}
	if !config.SSL.Enabled {
		t.Error("Expected SSL to be enabled")
	}
	if config.Options["timeout"] != "30s" {
		t.Errorf("Expected timeout '30s', got '%v'", config.Options["timeout"])
	}
}

func TestSSLConfig(t *testing.T) {
	ssl := SSLConfig{
		Enabled:  true,
		Mode:     "require",
		CertFile: "/cert.pem",
		KeyFile:  "/key.pem",
		CAFile:   "/ca.pem",
	}

	if !ssl.Enabled {
		t.Error("Expected SSL to be enabled")
	}
	if ssl.Mode != "require" {
		t.Errorf("Expected mode 'require', got '%s'", ssl.Mode)
	}
}

func TestProviderInfo(t *testing.T) {
	info := ProviderInfo{
		Name:         "TestProvider",
		Version:      "1.0.0",
		DatabaseType: DatabaseTypeSQL,
		Features: []Feature{
			FeatureTransactions,
			FeatureIndexes,
			FeatureJSONQueries,
		},
	}

	if info.Name != "TestProvider" {
		t.Errorf("Expected name 'TestProvider', got '%s'", info.Name)
	}
	if info.DatabaseType != DatabaseTypeSQL {
		t.Errorf("Expected database type SQL, got %s", info.DatabaseType)
	}
	if len(info.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(info.Features))
	}
}

func TestDatabaseTypes(t *testing.T) {
	tests := []struct {
		dbType   DatabaseType
		expected string
	}{
		{DatabaseTypeSQL, "sql"},
		{DatabaseTypeDocument, "document"},
		{DatabaseTypeKV, "key-value"},
		{DatabaseTypeGraph, "graph"},
		{DatabaseTypeMemory, "memory"},
	}

	for _, test := range tests {
		if string(test.dbType) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.dbType))
		}
	}
}

func TestFeatures(t *testing.T) {
	features := []Feature{
		FeatureTransactions,
		FeatureIndexes,
		FeatureTTL,
		FeatureAtomicOps,
		FeatureFullText,
		FeatureGeospatial,
		FeatureSubQueries,
		FeatureJoins,
		FeatureJSONQueries,
		FeatureIndexing,
		FeatureAggregation,
		FeatureMigration,
		FeatureRawSQL,
	}

	expectedCount := 13
	if len(features) != expectedCount {
		t.Errorf("Expected %d features, got %d", expectedCount, len(features))
	}

	// Test specific features
	if FeatureTransactions != "transactions" {
		t.Errorf("Expected 'transactions', got '%s'", FeatureTransactions)
	}
	if FeatureJSONQueries != "json_queries" {
		t.Errorf("Expected 'json_queries', got '%s'", FeatureJSONQueries)
	}
}

func TestOperators(t *testing.T) {
	tests := []struct {
		op       Operator
		expected string
	}{
		{OpEqual, "="},
		{OpNotEqual, "!="},
		{OpGreaterThan, ">"},
		{OpLessThan, "<"},
		{OpLike, "LIKE"},
		{OpIn, "IN"},
		{OpIsNull, "IS NULL"},
		{OpBetween, "BETWEEN"},
		{OpContains, "CONTAINS"},
		{OpRegex, "REGEX"},
	}

	for _, test := range tests {
		if string(test.op) != test.expected {
			t.Errorf("Expected operator %s, got %s", test.expected, string(test.op))
		}
	}
}

func TestLogicOperators(t *testing.T) {
	tests := []struct {
		op       LogicOperator
		expected string
	}{
		{LogicAnd, "AND"},
		{LogicOr, "OR"},
		{LogicNot, "NOT"},
	}

	for _, test := range tests {
		if string(test.op) != test.expected {
			t.Errorf("Expected logic operator %s, got %s", test.expected, string(test.op))
		}
	}
}

func TestOrder(t *testing.T) {
	order := Order{
		Field:     "name",
		Direction: OrderAsc,
	}

	if order.Field != "name" {
		t.Errorf("Expected field 'name', got '%s'", order.Field)
	}
	if order.Direction != OrderAsc {
		t.Errorf("Expected direction ASC, got %s", order.Direction)
	}
}

func TestJoinClause(t *testing.T) {
	join := JoinClause{
		Type:      JoinLeft,
		Table:     "posts",
		Condition: "users.id = posts.user_id",
		Alias:     "p",
	}

	if join.Type != JoinLeft {
		t.Errorf("Expected join type LEFT, got %s", join.Type)
	}
	if join.Table != "posts" {
		t.Errorf("Expected table 'posts', got '%s'", join.Table)
	}
	if join.Alias != "p" {
		t.Errorf("Expected alias 'p', got '%s'", join.Alias)
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		errType  ErrorType
		expected string
	}{
		{ErrorTypeValidation, "validation"},
		{ErrorTypeNotFound, "not_found"},
		{ErrorTypeDuplicate, "duplicate"},
		{ErrorTypeConnection, "connection"},
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypePermission, "permission"},
		{ErrorTypeConstraint, "constraint"},
		{ErrorTypeTransaction, "transaction"},
		{ErrorTypeUnsupported, "unsupported"},
		{ErrorTypeInternal, "internal"},
		{ErrorTypeSerialization, "serialization"},
		{ErrorTypeInvalidArgument, "invalid_argument"},
		{ErrorTypeDatabase, "database"},
	}

	for _, test := range tests {
		if string(test.errType) != test.expected {
			t.Errorf("Expected error type %s, got %s", test.expected, string(test.errType))
		}
	}
}

func TestSubQueryTypes(t *testing.T) {
	tests := []struct {
		sqType   SubQueryType
		expected string
	}{
		{SubQueryExists, "EXISTS"},
		{SubQueryNotExists, "NOT EXISTS"},
		{SubQueryIn, "IN"},
		{SubQueryNotIn, "NOT IN"},
		{SubQueryScalar, "SCALAR"},
		{SubQueryAny, "ANY"},
		{SubQueryAll, "ALL"},
	}

	for _, test := range tests {
		if string(test.sqType) != test.expected {
			t.Errorf("Expected subquery type %s, got %s", test.expected, string(test.sqType))
		}
	}
}

func TestIndexTypes(t *testing.T) {
	tests := []struct {
		indexType IndexType
		expected  string
	}{
		{IndexTypePrimary, "primary"},
		{IndexTypeUnique, "unique"},
		{IndexTypeStandard, "standard"},
		{IndexTypeFullText, "fulltext"},
		{IndexTypeGeoSpatial, "geospatial"},
		{IndexTypeComposite, "composite"},
	}

	for _, test := range tests {
		if string(test.indexType) != test.expected {
			t.Errorf("Expected index type %s, got %s", test.expected, string(test.indexType))
		}
	}
}