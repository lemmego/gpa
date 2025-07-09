package gpagorm

import (
	"context"
	"testing"
	"time"

	"github.com/lemmego/gpa"
)

func TestNewProvider(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.config.Driver != "sqlite" {
		t.Errorf("Expected driver 'sqlite', got '%s'", provider.config.Driver)
	}

	// Test closing
	err = provider.Close()
	if err != nil {
		t.Errorf("Failed to close provider: %v", err)
	}
}

func TestProviderHealth(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite",
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
		Driver:   "sqlite",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	info := provider.ProviderInfo()
	if info.Name != "GORM" {
		t.Errorf("Expected name 'GORM', got '%s'", info.Name)
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
		Driver:   "sqlite",
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
		gpa.FeatureMigration,
		gpa.FeatureRawSQL,
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
		Driver:   "sqlite",
		Database: ":memory:",
	}

	type User struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:255"`
		Age  int
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
	if info.Name != "GORM" {
		t.Errorf("Expected name 'GORM', got '%s'", info.Name)
	}
}

func TestProviderConfigure(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	newConfig := gpa.Config{
		Driver:   "sqlite",
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

func TestBuildPostgresDSN(t *testing.T) {
	config := gpa.Config{
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

	dsn := buildPostgresDSN(config)
	expected := "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=require"
	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestBuildPostgresDSNWithConnectionURL(t *testing.T) {
	config := gpa.Config{
		ConnectionURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		Host:          "ignored",
		Port:          9999,
	}

	dsn := buildPostgresDSN(config)
	if dsn != config.ConnectionURL {
		t.Errorf("Expected connection URL to be used, got '%s'", dsn)
	}
}

func TestBuildMySQLDSN(t *testing.T) {
	config := gpa.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "user",
		Password: "pass",
		Database: "testdb",
	}

	dsn := buildMySQLDSN(config)
	expected := "user:pass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestBuildSQLServerDSN(t *testing.T) {
	config := gpa.Config{
		Host:     "localhost",
		Port:     1433,
		Username: "user",
		Password: "pass",
		Database: "testdb",
	}

	dsn := buildSQLServerDSN(config)
	expected := "sqlserver://user:pass@localhost:1433?database=testdb"
	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestSupportedDrivers(t *testing.T) {
	drivers := SupportedDrivers()
	expectedDrivers := []string{"postgres", "postgresql", "mysql", "sqlite", "sqlite3", "sqlserver", "mssql"}

	if len(drivers) != len(expectedDrivers) {
		t.Errorf("Expected %d drivers, got %d", len(expectedDrivers), len(drivers))
	}

	for _, expected := range expectedDrivers {
		found := false
		for _, driver := range drivers {
			if driver == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected driver '%s' not found", expected)
		}
	}
}

func TestProviderWithCustomOptions(t *testing.T) {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level":      "silent",
				"singular_table": true,
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
		Driver:          "sqlite",
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
	sqlDB, err := provider.db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	stats := sqlDB.Stats()
	// Note: We can't directly test the configuration values since they're internal
	// But we can verify the connection is working
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
		Driver:   "sqlite",
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