package gpa

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ExampleProviderUsage demonstrates how to use the provider interface hierarchy
func ExampleProviderUsage() {
	// This is a conceptual example showing how to use the provider hierarchy

	// Example 1: Using GORM provider with SQL-specific features
	fmt.Println("=== SQL Provider Example ===")

	// Assume we have a GORM provider stored as a generic Provider interface
	var provider Provider // This would be your actual GORM provider instance

	// Check if it's a SQL provider and access SQL-specific features
	if sqlProvider, ok := AsSQLProvider(provider); ok {
		// Now we can access the underlying database/sql.DB instance
		if db, ok := sqlProvider.DB().(*sql.DB); ok {
			fmt.Printf("Connected to SQL database with %d max open connections\n", db.Stats().MaxOpenConnections)
		}

		// Execute raw SQL queries
		ctx := context.Background()
		results, err := sqlProvider.RawQuery(ctx, "SELECT COUNT(*) FROM users")
		if err == nil {
			fmt.Printf("Raw query results: %v\n", results)
		}

		// Start a transaction with specific isolation level
		tx, err := sqlProvider.BeginTx(ctx, &TxOptions{
			IsolationLevel: IsolationRepeatableRead,
			ReadOnly:       false,
		})
		if err == nil {
			fmt.Printf("Transaction started: %v\n", tx)
		}
	}

	// Example 2: Using Redis provider with Key-Value features
	fmt.Println("\n=== Key-Value Provider Example ===")

	// Assume we have a Redis provider
	var redisProvider Provider // This would be your actual Redis provider instance

	if kvProvider, ok := AsKeyValueProvider(redisProvider); ok {
		ctx := context.Background()

		// Use Redis-specific operations
		err := kvProvider.Set(ctx, "user:1", "John Doe", 5*time.Minute)
		if err == nil {
			fmt.Println("Set user data with TTL")
		}

		// Get value
		value, err := kvProvider.Get(ctx, "user:1")
		if err == nil {
			fmt.Printf("Retrieved user: %v\n", value)
		}

		// Check TTL
		ttl, err := kvProvider.TTL(ctx, "user:1")
		if err == nil {
			fmt.Printf("TTL remaining: %v\n", ttl)
		}

		// Access underlying Redis client for advanced operations
		// client := kvProvider.Client().(*redis.Client)
		// client.HSet(ctx, "user:1:profile", "name", "John", "age", 30)
	}

	// Example 3: Using MongoDB provider with Document features
	fmt.Println("\n=== Document Provider Example ===")

	var mongoProvider Provider // This would be your actual MongoDB provider instance

	if docProvider, ok := AsDocumentProvider(mongoProvider); ok {
		ctx := context.Background()

		// Create an index on a collection
		err := docProvider.CreateIndex(ctx, "users", map[string]interface{}{
			"email": 1,
		}, &IndexOptions{
			Unique: true,
			Name:   "unique_email",
		})
		if err == nil {
			fmt.Println("Created unique index on email field")
		}

		// Run aggregation pipeline
		pipeline := []interface{}{
			map[string]interface{}{
				"$match": map[string]interface{}{
					"age": map[string]interface{}{"$gte": 18},
				},
			},
			map[string]interface{}{
				"$group": map[string]interface{}{
					"_id":   "$department",
					"count": map[string]interface{}{"$sum": 1},
				},
			},
		}

		results, err := docProvider.Aggregate(ctx, "users", pipeline)
		if err == nil {
			fmt.Printf("Aggregation results: %v\n", results)
		}

		// Access underlying MongoDB database
		// db := docProvider.Database().(*mongo.Database)
		// collection := db.Collection("users")
	}

	// Example 4: Provider type checking
	fmt.Println("\n=== Provider Type Checking ===")

	providers := []Provider{
		// These would be your actual provider instances
		// gormProvider,
		// redisProvider,
		// mongoProvider,
	}

	for i, p := range providers {
		fmt.Printf("Provider %d: ", i+1)

		if IsSQLProvider(p) {
			fmt.Print("SQL ")
		}
		if IsKeyValueProvider(p) {
			fmt.Print("KeyValue ")
		}
		if IsDocumentProvider(p) {
			fmt.Print("Document ")
		}
		if IsWideColumnProvider(p) {
			fmt.Print("WideColumn ")
		}
		if IsGraphProvider(p) {
			fmt.Print("Graph ")
		}

		fmt.Printf("- %s\n", p.ProviderInfo().Name)
	}
}

// RealWorldExample shows how to use this in a real application
func RealWorldExample() {
	fmt.Println("\n=== Real World Usage Pattern ===")

	// This is how you would use it in a real application
	providerRegistry := map[string]Provider{
		"primary_db": nil, // Would be your GORM provider
		"cache":      nil, // Would be your Redis provider
		"documents":  nil, // Would be your MongoDB provider
	}

	// Function to get raw database connection for migrations
	getRawConnection := func(providerName string) (*sql.DB, error) {
		provider := providerRegistry[providerName]
		if provider == nil {
			return nil, fmt.Errorf("provider %s not found", providerName)
		}

		if sqlProvider, ok := AsSQLProvider(provider); ok {
			if db, ok := sqlProvider.DB().(*sql.DB); ok {
				return db, nil
			}
		}

		return nil, fmt.Errorf("provider %s is not a SQL provider", providerName)
	}

	// Function to access Redis client for pub/sub
	getRedisClient := func(providerName string) (interface{}, error) {
		provider := providerRegistry[providerName]
		if provider == nil {
			return nil, fmt.Errorf("provider %s not found", providerName)
		}

		if kvProvider, ok := AsKeyValueProvider(provider); ok {
			return kvProvider.Client(), nil
		}

		return nil, fmt.Errorf("provider %s is not a KeyValue provider", providerName)
	}

	// Function to access MongoDB database for advanced operations
	getMongoDatabase := func(providerName string) (interface{}, error) {
		provider := providerRegistry[providerName]
		if provider == nil {
			return nil, fmt.Errorf("provider %s not found", providerName)
		}

		if docProvider, ok := AsDocumentProvider(provider); ok {
			return docProvider.Database(), nil
		}

		return nil, fmt.Errorf("provider %s is not a Document provider", providerName)
	}

	// Usage examples
	fmt.Println("Functions for accessing provider-specific features:")
	fmt.Printf("- getRawConnection: %v\n", getRawConnection != nil)
	fmt.Printf("- getRedisClient: %v\n", getRedisClient != nil)
	fmt.Printf("- getMongoDatabase: %v\n", getMongoDatabase != nil)
}
