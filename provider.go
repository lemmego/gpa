package gpa

import (
	"fmt"
)

// =====================================
// Provider Interfaces
// =====================================

// Provider is the main interface for database provider implementations.
// Provides type-safe access to repository instances.
// This is the PRIMARY provider interface - use this for all new code.
type Provider[T any] interface {
	// ===============================
	// Repository Creation
	// ===============================
	
	// Repository creates a new type-safe repository for entity type T.
	// No reflection needed - type is known at compile time.
	// Example: repo := provider.Repository()
	Repository() Repository[T]

	// ===============================
	// Configuration and Lifecycle
	// ===============================
	
	// Configure applies new configuration to the provider.
	// Can be used to change connection settings, pool sizes, etc. at runtime.
	// May require reconnection depending on what settings changed.
	Configure(config Config) error
	
	// Health checks if the database connection is healthy and responsive.
	// Returns error if the database is unreachable or not functioning properly.
	// Useful for health check endpoints and monitoring.
	Health() error
	
	// Close shuts down the provider and releases all resources.
	// Closes database connections, stops background tasks, etc.
	// Should be called during application shutdown.
	Close() error

	// ===============================
	// Metadata and Capabilities
	// ===============================
	
	// SupportedFeatures returns a list of features this provider supports.
	// Features include things like transactions, full-text search, pub/sub, etc.
	// Use this to check capabilities before using advanced features.
	SupportedFeatures() []Feature
	
	// ProviderInfo returns metadata about this provider.
	// Includes provider name, version, database type, and supported features.
	// Useful for debugging, logging, and feature detection.
	ProviderInfo() ProviderInfo
}

// =====================================
// Provider Factory and Registry
// =====================================

// =====================================
// Type-Safe Provider Creation
// =====================================

// NewProvider creates a new type-safe provider instance.
// This is the RECOMMENDED way to create providers for new applications.
//
// Example:
//	provider, err := gpa.NewProvider[User]("gorm", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	repo := provider.Repository()
func NewProvider[T any](driverName string, config Config) (Provider[T], error) {
	return nil, fmt.Errorf("NewProvider[T] must be implemented by provider packages")
}