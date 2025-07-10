package gpa

import (
	"errors"
	"reflect"
	"testing"
)

// Mock provider for testing
type mockProvider struct {
	name         string
	closed       bool
	healthError  error
	configError  error
	features     []Feature
	databaseType DatabaseType
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:         name,
		features:     []Feature{FeatureTransactions, FeatureIndexes},
		databaseType: DatabaseTypeSQL,
	}
}

func (m *mockProvider) Configure(config Config) error {
	return m.configError
}

func (m *mockProvider) Health() error {
	return m.healthError
}

func (m *mockProvider) Close() error {
	if m.closed {
		return errors.New("already closed")
	}
	m.closed = true
	return nil
}

func (m *mockProvider) SupportedFeatures() []Feature {
	return m.features
}

func (m *mockProvider) ProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:         m.name,
		Version:      "1.0.0",
		DatabaseType: m.databaseType,
		Features:     m.features,
	}
}

func TestProviderRegistry_Register(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider1 := newMockProvider("gorm")
	provider2 := newMockProvider("gorm")

	registry.Register("primary", provider1)
	registry.Register("secondary", provider2)

	if len(registry.providers) != 1 {
		t.Errorf("Expected 1 provider type, got %d", len(registry.providers))
	}

	if len(registry.providers["gorm"]) != 2 {
		t.Errorf("Expected 2 gorm instances, got %d", len(registry.providers["gorm"]))
	}

	if registry.providers["gorm"]["primary"] != provider1 {
		t.Error("Primary provider not registered correctly")
	}

	if registry.providers["gorm"]["secondary"] != provider2 {
		t.Error("Secondary provider not registered correctly")
	}
}

func TestProviderRegistry_RegisterDefault(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider := newMockProvider("redis")
	registry.RegisterDefault(provider)

	if len(registry.providers) != 1 {
		t.Errorf("Expected 1 provider type, got %d", len(registry.providers))
	}

	if registry.providers["redis"]["default"] != provider {
		t.Error("Default provider not registered correctly")
	}
}

func TestProviderRegistry_Get(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider1 := newMockProvider("mongo")
	provider2 := newMockProvider("mongo")

	registry.Register("primary", provider1)
	registry.Register("secondary", provider2)

	// Test getting specific instance
	result, err := registry.Get("mongo", "primary")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != provider1 {
		t.Error("Got wrong provider instance")
	}

	// Test getting default instance (should fail as no default registered)
	_, err = registry.Get("mongo")
	if err == nil {
		t.Error("Expected error when getting non-existent default")
	}

	// Register default and test again
	defaultProvider := newMockProvider("mongo")
	registry.RegisterDefault(defaultProvider)

	result, err = registry.Get("mongo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != defaultProvider {
		t.Error("Got wrong default provider")
	}

	// Test non-existent provider type
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider type")
	}

	// Test non-existent instance
	_, err = registry.Get("mongo", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent instance")
	}
}

func TestProviderRegistry_MustGet(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider := newMockProvider("redis")
	registry.Register("cache", provider)

	// Test successful get
	result := registry.MustGet("redis", "cache")
	if result != provider {
		t.Error("Got wrong provider instance")
	}

	// Test panic on non-existent provider
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent provider")
		}
	}()
	registry.MustGet("redis", "nonexistent")
}

func TestProviderRegistry_GetByType(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider1 := newMockProvider("gorm")
	provider2 := newMockProvider("gorm")

	registry.Register("primary", provider1)
	registry.Register("secondary", provider2)

	results, err := registry.GetByType("gorm")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(results))
	}

	if results["primary"] != provider1 {
		t.Error("Primary provider not returned correctly")
	}

	if results["secondary"] != provider2 {
		t.Error("Secondary provider not returned correctly")
	}

	// Test modifying returned map doesn't affect registry
	results["new"] = newMockProvider("test")
	originalResults, _ := registry.GetByType("gorm")
	if len(originalResults) != 2 {
		t.Error("Registry was modified by external map change")
	}

	// Test non-existent type
	_, err = registry.GetByType("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider type")
	}
}

func TestProviderRegistry_ListTypes(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	// Empty registry
	types := registry.ListTypes()
	if len(types) != 0 {
		t.Errorf("Expected 0 types, got %d", len(types))
	}

	// Add providers
	registry.Register("primary", newMockProvider("gorm"))
	registry.Register("cache", newMockProvider("redis"))
	registry.Register("docs", newMockProvider("mongo"))

	types = registry.ListTypes()
	if len(types) != 3 {
		t.Errorf("Expected 3 types, got %d", len(types))
	}

	expectedTypes := map[string]bool{"gorm": true, "redis": true, "mongo": true}
	for _, typ := range types {
		if !expectedTypes[typ] {
			t.Errorf("Unexpected type: %s", typ)
		}
	}
}

func TestProviderRegistry_ListInstances(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	registry.Register("primary", newMockProvider("gorm"))
	registry.Register("secondary", newMockProvider("gorm"))
	registry.Register("readonly", newMockProvider("gorm"))

	instances, err := registry.ListInstances("gorm")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(instances) != 3 {
		t.Errorf("Expected 3 instances, got %d", len(instances))
	}

	expectedInstances := map[string]bool{"primary": true, "secondary": true, "readonly": true}
	for _, instance := range instances {
		if !expectedInstances[instance] {
			t.Errorf("Unexpected instance: %s", instance)
		}
	}

	// Test non-existent type
	_, err = registry.ListInstances("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider type")
	}
}

func TestProviderRegistry_Remove(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider1 := newMockProvider("redis")
	provider2 := newMockProvider("redis")

	registry.Register("cache", provider1)
	registry.Register("sessions", provider2)

	// Remove one instance
	err := registry.Remove("redis", "cache")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !provider1.closed {
		t.Error("Provider was not closed")
	}

	// Check instance was removed
	_, err = registry.Get("redis", "cache")
	if err == nil {
		t.Error("Expected error, instance should be removed")
	}

	// Check other instance still exists
	result, err := registry.Get("redis", "sessions")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != provider2 {
		t.Error("Wrong provider returned")
	}

	// Remove last instance should clean up type map
	err = registry.Remove("redis", "sessions")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(registry.providers) != 0 {
		t.Error("Provider type map not cleaned up")
	}

	// Test removing non-existent type
	err = registry.Remove("nonexistent", "instance")
	if err == nil {
		t.Error("Expected error for non-existent provider type")
	}

	// Test removing non-existent instance
	registry.Register("test", newMockProvider("test"))
	err = registry.Remove("test", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent instance")
	}
}

func TestProviderRegistry_RemoveAll(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider1 := newMockProvider("gorm")
	provider2 := newMockProvider("redis")
	provider3 := newMockProvider("mongo")

	registry.Register("primary", provider1)
	registry.Register("cache", provider2)
	registry.Register("docs", provider3)

	err := registry.RemoveAll()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !provider1.closed || !provider2.closed || !provider3.closed {
		t.Error("Not all providers were closed")
	}

	if len(registry.providers) != 0 {
		t.Error("Registry not cleaned up")
	}

	// Test with provider that fails to close
	failingProvider := newMockProvider("failing")
	failingProvider.closed = true // This will cause Close() to return error
	registry.Register("failing", failingProvider)

	err = registry.RemoveAll()
	if err == nil {
		t.Error("Expected error when provider fails to close")
	}
}

func TestProviderRegistry_HealthCheck(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	healthyProvider := newMockProvider("gorm")
	unhealthyProvider := newMockProvider("redis")
	unhealthyProvider.healthError = errors.New("connection failed")

	registry.Register("primary", healthyProvider)
	registry.Register("cache", unhealthyProvider)

	results := registry.HealthCheck()

	if len(results) != 2 {
		t.Errorf("Expected 2 provider types, got %d", len(results))
	}

	if results["gorm"]["primary"] != nil {
		t.Error("Healthy provider should return nil error")
	}

	if results["redis"]["cache"] == nil {
		t.Error("Unhealthy provider should return error")
	}

	if results["redis"]["cache"].Error() != "connection failed" {
		t.Errorf("Expected 'connection failed', got %v", results["redis"]["cache"])
	}
}

func TestRegistry_Singleton(t *testing.T) {
	registry1 := Registry()
	registry2 := Registry()

	if registry1 != registry2 {
		t.Error("Registry should return same instance")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Clear global registry for test
	instance = &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	provider := newMockProvider("test")

	// Test Register
	Register[*mockProvider]("instance", provider)
	result, err := Get[*mockProvider]("instance")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != provider {
		t.Error("Provider not registered correctly")
	}

	// Test RegisterDefault
	defaultProvider := newMockProvider("test")
	RegisterDefault[*mockProvider](defaultProvider)
	result, err = Get[*mockProvider]()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != defaultProvider {
		t.Error("Default provider not registered correctly")
	}

	// Test MustGet
	result = MustGet[*mockProvider]("instance")
	if result != provider {
		t.Error("MustGet returned wrong provider")
	}

	// Test MustGet panic on non-existent instance
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent provider")
		}
	}()
	MustGet[*mockProvider]("nonexistent")
}

func TestGenericFunctions(t *testing.T) {
	// Clear global registry for test
	instance = &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}
	// Clear type mapping
	typeMapMutex.Lock()
	typeToProviderName = make(map[reflect.Type]string)
	typeMapMutex.Unlock()

	// Create providers that all have the same provider name for testing
	// Since we're using the same Go type (*mockProvider), they will map to the same type
	gormProvider := newMockProvider("test")  // Same provider name
	redisProvider := newMockProvider("test") // Same provider name for testing

	// Test Register
	Register[*mockProvider]("primary", gormProvider)
	Register[*mockProvider]("cache", redisProvider)

	// Test Get
	result, err := Get[*mockProvider]("primary")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != gormProvider {
		t.Error("Get returned wrong provider")
	}

	// Test Get with different type (should fail because type not registered)
	type differentMockProvider struct {
		mockProvider
	}
	_, err = Get[*differentMockProvider]("primary")
	if err == nil {
		t.Error("Expected error for unregistered type")
	}

	// Test MustGet for cache provider
	result = MustGet[*mockProvider]("cache")
	if result != redisProvider {
		t.Error("MustGet returned wrong provider")
	}

	// Test RegisterDefault
	defaultProvider := newMockProvider("test")
	RegisterDefault[*mockProvider](defaultProvider)

	result, err = Get[*mockProvider]()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != defaultProvider {
		t.Error("RegisterDefault not working correctly")
	}

	// Test GetByType
	allProviders, err := GetByType[*mockProvider]()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(allProviders) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(allProviders))
	}

	// Test panic on non-existent provider instance
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent provider")
		}
	}()
	MustGet[*mockProvider]("nonexistent")
}

func TestTypeInferenceAPI(t *testing.T) {
	// Clear global registry for test
	instance = &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}
	// Clear type mapping
	typeMapMutex.Lock()
	typeToProviderName = make(map[reflect.Type]string)
	typeMapMutex.Unlock()

	// Demonstrate the new cleaner API
	provider := newMockProvider("test")

	// Register provider - type is inferred from the generic parameter
	Register[*mockProvider]("primary", provider)

	// Get provider - no need to specify provider type string!
	result, err := Get[*mockProvider]("primary")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != provider {
		t.Error("Type inference failed")
	}

	// Get default provider
	defaultProvider := newMockProvider("test")
	RegisterDefault[*mockProvider](defaultProvider)

	defaultResult, err := Get[*mockProvider]() // No instance name = default
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if defaultResult != defaultProvider {
		t.Error("Default provider retrieval failed")
	}

	// Get all providers of this type
	allProviders, err := GetByType[*mockProvider]()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(allProviders) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(allProviders))
	}

	// Must get (panic version)
	mustResult := MustGet[*mockProvider]("primary")
	if mustResult != provider {
		t.Error("MustGet failed")
	}
}

func TestProviderRegistry_ConcurrentAccess(t *testing.T) {
	registry := &ProviderRegistry{
		providers: make(map[string]map[string]Provider),
	}

	// Test concurrent registration and access
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			provider := newMockProvider("test")
			registry.Register("instance", provider)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.Get("test", "instance")
			registry.ListTypes()
		}
		done <- true
	}()

	<-done
	<-done

	// Should not crash - test passes if no race condition detected
}
