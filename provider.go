package gpa

// =====================================
// Provider Interface
// =====================================

// Provider is the main interface for database provider implementations.
// Provides unified access to database operations with type safety through
// repository creation functions.
//
// This interface is implemented by all provider packages:
// • gpagorm.Provider (GORM SQL adapter)
// • gpabun.Provider (Bun SQL adapter)
// • gpamongo.Provider (MongoDB adapter)
// • gparedis.Provider (Redis adapter)
//
// Usage:
//
//	provider, err := gpagorm.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Create type-safe repositories
//	userRepo := gpagorm.GetRepository[User](provider)
//	postRepo := gpagorm.GetRepository[Post](provider)
type Provider interface {
	// Configure applies new configuration to the provider.
	// Can be used to change connection settings, pool sizes, etc. at runtime.
	// May require reconnection depending on what settings changed.
	Configure(config Config) error

	// Health checks if the database connection is healthy and responsive.
	// Returns error if the database is unreachable or not functioning properly.
	// Useful for health check endpoints and monitoring.
	Health() error

	// Close shuts down the provider and releases all resources.
	// Closes database providers, stops background tasks, etc.
	// Should be called during application shutdown.
	Close() error

	// SupportedFeatures returns a list of features this provider supports.
	// Features include things like transactions, full-text search, pub/sub, etc.
	// Use this to check capabilities before using advanced features.
	SupportedFeatures() []Feature

	// ProviderInfo returns metadata about this provider.
	// Includes provider name, version, database type, and supported features.
	// Useful for debugging, logging, and feature detection.
	ProviderInfo() ProviderInfo
}
