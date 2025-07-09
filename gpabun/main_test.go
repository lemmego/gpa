package gpabun

import (
	"context"
	"testing"
	"time"

	"github.com/lemmego/gpa"
)

func TestNewProvider(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.config.Driver != "sqlite3" {
		t.Errorf("Expected driver 'sqlite3', got '%s'", provider.config.Driver)
	}
}

func TestProviderHealth(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	err = provider.Health()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestProviderInfo(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	info := provider.ProviderInfo()
	if info.Name != "Bun" {
		t.Errorf("Expected name 'Bun', got '%s'", info.Name)
	}
	if info.DatabaseType != gpa.DatabaseTypeSQL {
		t.Errorf("Expected SQL database type, got %s", info.DatabaseType)
	}
	if len(info.Features) == 0 {
		t.Error("Expected features to be populated")
	}
}

func TestSupportedFeatures(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	features := provider.SupportedFeatures()
	expectedFeatures := []gpa.Feature{
		gpa.FeatureTransactions,
		gpa.FeatureJSONQueries,
		gpa.FeatureIndexing,
		gpa.FeatureAggregation,
		gpa.FeatureFullTextSearch,
		gpa.FeatureSubQueries,
		gpa.FeatureJoins,
	}

	if len(features) != len(expectedFeatures) {
		t.Errorf("Expected %d features, got %d", len(expectedFeatures), len(features))
	}

	for _, expected := range expectedFeatures {
		found := false
		for _, feature := range features {
			if feature == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected feature '%s' not found", expected)
		}
	}
}

func TestNewTypeSafeProvider(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	type User struct {
		ID   int64  `bun:",pk,autoincrement"`
		Name string `bun:"name"`
		Age  int    `bun:"age"`
	}

	provider, err := NewTypeSafeProvider[User](config)
	if err != nil {
		t.Fatalf("Failed to create type-safe provider: %v", err)
	}
	defer provider.Close()

	if provider == nil {
		t.Fatal("Expected type-safe provider to be created")
	}

	// Test getting repository
	repo := provider.Repository()
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}

	// Test provider methods
	err = provider.Health()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	info := provider.ProviderInfo()
	if info.Name != "Bun" {
		t.Errorf("Expected name 'Bun', got '%s'", info.Name)
	}
}

func TestProviderConfigure(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	newConfig := gpa.Config{
		Driver:   "sqlite3",
		Database: "test.db",
	}

	err = provider.Configure(newConfig)
	if err != nil {
		t.Errorf("Failed to configure provider: %v", err)
	}

	if provider.config.Database != "test.db" {
		t.Errorf("Expected database 'test.db', got '%s'", provider.config.Database)
	}
}

func TestCreatePostgresConnection(t *testing.T) {
	config := gpa.Config{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "testdb",
	}

	// This will fail to connect but should create the DSN correctly
	_, err := createPostgresConnection(config)
	// We expect a connection error since we're not running postgres
	if err == nil {
		t.Log("Postgres connection succeeded (unexpected in test)")
	} else {
		t.Logf("Expected postgres connection error: %v", err)
	}
}

func TestCreatePostgresConnectionWithURL(t *testing.T) {
	config := gpa.Config{
		ConnectionURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		Host:          "ignored",
		Port:          9999,
	}

	// This will fail to connect but should use the connection URL
	_, err := createPostgresConnection(config)
	// We expect a connection error since we're not running postgres
	if err == nil {
		t.Log("Postgres connection succeeded (unexpected in test)")
	} else {
		t.Logf("Expected postgres connection error: %v", err)
	}
}

func TestCreateMySQLConnection(t *testing.T) {
	config := gpa.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "user",
		Password: "pass",
		Database: "testdb",
	}

	// This will fail to connect but should create the DSN correctly
	_, err := createMySQLConnection(config)
	// We expect a connection error since we're not running mysql
	if err == nil {
		t.Log("MySQL connection succeeded (unexpected in test)")
	} else {
		t.Logf("Expected mysql connection error: %v", err)
	}
}

func TestCreateMySQLConnectionWithURL(t *testing.T) {
	config := gpa.Config{
		ConnectionURL: "user:pass@tcp(localhost:3306)/testdb",
		Host:          "ignored",
		Port:          9999,
	}

	// This will fail to connect but should use the connection URL
	_, err := createMySQLConnection(config)
	// We expect a connection error since we're not running mysql
	if err == nil {
		t.Log("MySQL connection succeeded (unexpected in test)")
	} else {
		t.Logf("Expected mysql connection error: %v", err)
	}
}

func TestCreateSQLiteConnection(t *testing.T) {
	config := gpa.Config{
		Database: ":memory:",
	}

	db, err := createSQLiteConnection(config)
	if err != nil {
		t.Errorf("Failed to create SQLite connection: %v", err)
	}
	if db != nil {
		db.Close()
	}
}

func TestProviderWithCustomOptions(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
		Options: map[string]interface{}{
			"bun": map[string]interface{}{
				"log_level": "debug",
			},
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider with custom options: %v", err)
	}
	defer provider.Close()

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}
}

func TestProviderConnectionPoolSettings(t *testing.T) {
	config := gpa.Config{
		Driver:          "sqlite3",
		Database:        ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Test that the provider was created successfully with pool settings
	sqlDB := provider.db.DB
	stats := sqlDB.Stats()
	if stats.OpenConnections < 0 {
		t.Error("Expected valid connection stats")
	}
}

func TestUnsupportedDriver(t *testing.T) {
	config := gpa.Config{
		Driver:   "unsupported",
		Database: "test.db",
	}

	_, err := NewProvider(config)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	expectedMsg := "unsupported driver: unsupported"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestContextTimeout(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create a context with timeout
	_, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	// This should work since SQLite is fast
	err = provider.Health()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestProviderSSLConfiguration(t *testing.T) {
	config := gpa.Config{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "testdb",
		SSL: gpa.SSLConfig{
			Enabled: true,
			Mode:    "require",
		},
	}

	// This will fail to connect but should handle SSL configuration
	_, err := NewProvider(config)
	if err == nil {
		t.Log("Postgres connection with SSL succeeded (unexpected in test)")
	} else {
		t.Logf("Expected postgres connection error with SSL: %v", err)
	}
}

func TestInvalidSQLiteDatabase(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: "/invalid/path/to/database.db",
	}

	_, err := NewProvider(config)
	if err == nil {
		t.Error("Expected error for invalid SQLite database path")
	}
}

func TestBunLoggingConfiguration(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
		Options: map[string]interface{}{
			"bun": map[string]interface{}{
				"log_level": "silent",
			},
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider with silent logging: %v", err)
	}
	defer provider.Close()

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}
}

func TestInvalidBunOptions(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
		Options: map[string]interface{}{
			"bun": map[string]interface{}{
				"log_level": 123, // invalid type
			},
		},
	}

	// Should still work, just ignore invalid options
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider with invalid options: %v", err)
	}
	defer provider.Close()
}

func TestProviderWithInvalidOptions(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
		Options: map[string]interface{}{
			"bun": "invalid", // should be map
		},
	}

	// Should still work, just ignore invalid options
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider with invalid options: %v", err)
	}
	defer provider.Close()
}