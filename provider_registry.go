package gpa

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	once                  sync.Once
	instance              *ProviderRegistry
	ErrProviderNotFound   = errors.New("provider not found")
	// typeToProviderName maps Go types to provider names for type inference
	typeToProviderName    = make(map[reflect.Type]string)
	typeMapMutex          sync.RWMutex
)

// ProviderRegistry holds all registered providers organized by type and name
type ProviderRegistry struct {
	mutex     sync.RWMutex
	providers map[string]map[string]Provider // [providerType][instanceName]Provider
}

// Registry returns the singleton instance of ProviderRegistry
func Registry() *ProviderRegistry {
	once.Do(func() {
		instance = &ProviderRegistry{
			providers: make(map[string]map[string]Provider),
		}
	})
	return instance
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(instanceName string, provider Provider) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	providerType := provider.ProviderInfo().Name
	if r.providers[providerType] == nil {
		r.providers[providerType] = make(map[string]Provider)
	}
	r.providers[providerType][instanceName] = provider
	
	// Also register the type mapping for type inference
	goType := reflect.TypeOf(provider)
	typeMapMutex.Lock()
	typeToProviderName[goType] = providerType
	typeMapMutex.Unlock()
}

// RegisterDefault registers a provider as the default instance for its type
func (r *ProviderRegistry) RegisterDefault(provider Provider) {
	r.Register("default", provider)
}

// Get retrieves a provider by type and instance name
func (r *ProviderRegistry) Get(providerType string, instanceName ...string) (Provider, error) {
	name := "default"
	if len(instanceName) > 0 {
		name = instanceName[0]
	}
	
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	typeProviders, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("%w: provider type '%s' not found", ErrProviderNotFound, providerType)
	}
	
	provider, exists := typeProviders[name]
	if !exists {
		return nil, fmt.Errorf("%w: instance '%s' of type '%s' not found", ErrProviderNotFound, name, providerType)
	}
	
	return provider, nil
}

// MustGet retrieves a provider by type and instance name, panics if not found
func (r *ProviderRegistry) MustGet(providerType string, instanceName ...string) Provider {
	provider, err := r.Get(providerType, instanceName...)
	if err != nil {
		panic(err)
	}
	return provider
}

// GetByType returns all providers of a specific type
func (r *ProviderRegistry) GetByType(providerType string) (map[string]Provider, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	typeProviders, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("%w: provider type '%s' not found", ErrProviderNotFound, providerType)
	}
	
	// Return a copy to prevent external modification
	result := make(map[string]Provider)
	for name, provider := range typeProviders {
		result[name] = provider
	}
	return result, nil
}

// ListTypes returns all registered provider types
func (r *ProviderRegistry) ListTypes() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	types := make([]string, 0, len(r.providers))
	for providerType := range r.providers {
		types = append(types, providerType)
	}
	return types
}

// ListInstances returns all instance names for a given provider type
func (r *ProviderRegistry) ListInstances(providerType string) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	typeProviders, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("%w: provider type '%s' not found", ErrProviderNotFound, providerType)
	}
	
	instances := make([]string, 0, len(typeProviders))
	for instanceName := range typeProviders {
		instances = append(instances, instanceName)
	}
	return instances, nil
}

// Remove closes and removes a provider from the registry
func (r *ProviderRegistry) Remove(providerType, instanceName string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	typeProviders, exists := r.providers[providerType]
	if !exists {
		return fmt.Errorf("%w: provider type '%s' not found", ErrProviderNotFound, providerType)
	}
	
	provider, exists := typeProviders[instanceName]
	if !exists {
		return fmt.Errorf("%w: instance '%s' of type '%s' not found", ErrProviderNotFound, instanceName, providerType)
	}
	
	if err := provider.Close(); err != nil {
		return fmt.Errorf("error closing provider: %w", err)
	}
	
	delete(typeProviders, instanceName)
	
	// Clean up empty provider type maps
	if len(typeProviders) == 0 {
		delete(r.providers, providerType)
	}
	
	return nil
}

// RemoveAll closes and removes all providers from the registry
func (r *ProviderRegistry) RemoveAll() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	for providerType, typeProviders := range r.providers {
		for instanceName, provider := range typeProviders {
			if err := provider.Close(); err != nil {
				return fmt.Errorf("error closing provider %s:%s: %w", providerType, instanceName, err)
			}
		}
	}
	
	r.providers = make(map[string]map[string]Provider)
	return nil
}

// HealthCheck checks the health of all registered providers
func (r *ProviderRegistry) HealthCheck() map[string]map[string]error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	results := make(map[string]map[string]error)
	for providerType, typeProviders := range r.providers {
		results[providerType] = make(map[string]error)
		for instanceName, provider := range typeProviders {
			results[providerType][instanceName] = provider.Health()
		}
	}
	return results
}

// Package-level functions

// Register registers a provider using Go generics for compile-time type safety
// T must be a concrete provider type that implements Provider interface
// Usage: gpa.Register[*gpagorm.Provider]("instance", provider)
func Register[T Provider](instanceName string, provider T) {
	Registry().Register(instanceName, provider)
}

// RegisterDefault registers a provider as default using Go generics for compile-time type safety
// T must be a concrete provider type that implements Provider interface
// Usage: gpa.RegisterDefault[*gpagorm.Provider](provider)
func RegisterDefault[T Provider](provider T) {
	Registry().RegisterDefault(provider)
}

// getProviderTypeFromGeneric extracts the provider type name from the generic type T
func getProviderTypeFromGeneric[T Provider]() (string, error) {
	var zero T
	goType := reflect.TypeOf(zero)
	
	typeMapMutex.RLock()
	providerType, exists := typeToProviderName[goType]
	typeMapMutex.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("provider type %T is not registered. You must register a provider of this type first", zero)
	}
	
	return providerType, nil
}

// Get retrieves a provider by type using Go generics for compile-time type safety
// T must be a concrete provider type that implements Provider interface
// The provider type is automatically inferred from T
// Usage: provider, err := gpa.Get[*gpagorm.Provider]("instance")
func Get[T Provider](instanceName ...string) (T, error) {
	var zero T
	
	// Get provider type from the registered type mapping
	providerType, err := getProviderTypeFromGeneric[T]()
	if err != nil {
		return zero, err
	}
	
	// Get the provider from registry
	provider, err := Registry().Get(providerType, instanceName...)
	if err != nil {
		return zero, err
	}
	
	// Type assertion to ensure we get the correct type
	typedProvider, ok := provider.(T)
	if !ok {
		return zero, fmt.Errorf("provider is not of expected type %T, got %T", zero, provider)
	}
	
	return typedProvider, nil
}

// MustGet retrieves a provider by type using Go generics, panics if not found
// T must be a concrete provider type that implements Provider interface
// The provider type is automatically inferred from T
// Usage: provider := gpa.MustGet[*gpagorm.Provider]("instance")
func MustGet[T Provider](instanceName ...string) T {
	provider, err := Get[T](instanceName...)
	if err != nil {
		panic(err)
	}
	return provider
}

// GetByType returns all providers of a specific type using Go generics
// T must be a concrete provider type that implements Provider interface
// The provider type is automatically inferred from T
// Usage: providers, err := gpa.GetByType[*gpagorm.Provider]()
func GetByType[T Provider]() (map[string]T, error) {
	// Get provider type from the registered type mapping
	providerType, err := getProviderTypeFromGeneric[T]()
	if err != nil {
		return nil, err
	}
	
	providers, err := Registry().GetByType(providerType)
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]T)
	for name, provider := range providers {
		typedProvider, ok := provider.(T)
		if !ok {
			return nil, fmt.Errorf("provider %s is not of expected type %T, got %T", name, (*T)(nil), provider)
		}
		result[name] = typedProvider
	}
	
	return result, nil
}

