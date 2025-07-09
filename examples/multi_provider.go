package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpagorm"
	"github.com/lemmego/gpa/gpamongo"
	"github.com/lemmego/gpa/gparedis"
)

// Product represents a product that can be stored in different databases
type Product struct {
	ID          string  `gorm:"primaryKey" bson:"_id,omitempty" json:"id"`
	Name        string  `gorm:"size:255;not null" bson:"name" json:"name"`
	Description string  `gorm:"type:text" bson:"description" json:"description"`
	Price       float64 `gorm:"not null" bson:"price" json:"price"`
	Category    string  `gorm:"size:100" bson:"category" json:"category"`
	InStock     bool    `gorm:"default:true" bson:"in_stock" json:"in_stock"`
}

// CacheEntry represents a cached item for Redis
type CacheEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func RunMultiProvider() {
	fmt.Println("🚀 Multi-Provider Example")
	fmt.Println("Demonstrating the same operations across different database providers")

	ctx := context.Background()

	// ============================================
	// Setup SQL Database (GORM + SQLite)
	// ============================================
	fmt.Println("\n=== Setting up SQL Database (GORM) ===")
	sqlConfig := gpa.Config{
		Driver:   "sqlite",
		Database: "products.db",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "warn",
			},
		},
	}

	sqlProvider, err := gpagorm.NewTypeSafeProvider[Product](sqlConfig)
	if err != nil {
		log.Printf("Failed to create SQL provider: %v", err)
	} else {
		defer sqlProvider.Close()
		fmt.Println("✓ SQL provider created")

		// Auto-migrate the table
		sqlRepo := sqlProvider.Repository()
		if migratableRepo, ok := sqlRepo.(gpa.MigratableRepository[Product]); ok {
			err = migratableRepo.MigrateTable(ctx)
			if err != nil {
				log.Printf("Failed to migrate SQL table: %v", err)
			} else {
				fmt.Println("✓ SQL table migrated")
			}
		}
	}

	// ============================================
	// Setup Document Database (MongoDB)
	// ============================================
	fmt.Println("\n=== Setting up Document Database (MongoDB) ===")
	mongoConfig := gpa.Config{
		Driver:        "mongodb",
		ConnectionURL: "mongodb://localhost:27017",
		Database:      "products_db",
	}

	var mongoProvider gpa.Provider[Product]
	mongoProvider, err = gpamongo.NewTypeSafeProvider[Product](mongoConfig)
	if err != nil {
		log.Printf("MongoDB not available, skipping: %v", err)
		mongoProvider = nil
	} else {
		defer mongoProvider.Close()
		fmt.Println("✓ MongoDB provider created")
	}

	// ============================================
	// Setup Key-Value Store (Redis)
	// ============================================
	fmt.Println("\n=== Setting up Key-Value Store (Redis) ===")
	redisConfig := gpa.Config{
		Driver:        "redis",
		ConnectionURL: "redis://localhost:6379",
		Database:      "0",
	}

	var redisProvider gpa.Provider[CacheEntry]
	redisProvider, err = gparedis.NewTypeSafeProvider[CacheEntry](redisConfig)
	if err != nil {
		log.Printf("Redis not available, skipping: %v", err)
		redisProvider = nil
	} else {
		defer redisProvider.Close()
		fmt.Println("✓ Redis provider created")
	}

	// ============================================
	// Demonstrate Operations Across Providers
	// ============================================
	sampleProducts := []*Product{
		{ID: "1", Name: "Laptop", Description: "Gaming laptop", Price: 1299.99, Category: "Electronics", InStock: true},
		{ID: "2", Name: "Coffee Mug", Description: "Ceramic mug", Price: 19.99, Category: "Home", InStock: true},
		{ID: "3", Name: "Book", Description: "Programming guide", Price: 49.99, Category: "Books", InStock: false},
	}

	// SQL Operations
	if sqlProvider != nil {
		fmt.Println("\n=== SQL Database Operations ===")
		performSQLOperations(ctx, sqlProvider.Repository(), sampleProducts)
	}

	// MongoDB Operations
	if mongoProvider != nil {
		fmt.Println("\n=== MongoDB Operations ===")
		performMongoOperations(ctx, mongoProvider.Repository(), sampleProducts)
	}

	// Redis Operations
	if redisProvider != nil {
		fmt.Println("\n=== Redis Operations ===")
		performRedisOperations(ctx, redisProvider.Repository())
	}

	// ============================================
	// Compare Provider Features
	// ============================================
	fmt.Println("\n=== Provider Feature Comparison ===")
	providers := map[string]interface{}{
		"SQL (GORM)": sqlProvider,
		"MongoDB":    mongoProvider,
		"Redis":      redisProvider,
	}

	for name, provider := range providers {
		if provider == nil {
			fmt.Printf("%s: Not available\n", name)
			continue
		}

		var info gpa.ProviderInfo
		switch p := provider.(type) {
		case gpa.Provider[Product]:
			info = p.ProviderInfo()
		case gpa.Provider[CacheEntry]:
			info = p.ProviderInfo()
		}

		fmt.Printf("%s:\n", name)
		fmt.Printf("  Database Type: %s\n", info.DatabaseType)
		fmt.Printf("  Features: %v\n", info.Features)
		fmt.Printf("  Version: %s\n", info.Version)
	}

	fmt.Println("\n🎉 Multi-provider example completed!")
}

func performSQLOperations(ctx context.Context, repo gpa.Repository[Product], products []*Product) {
	// Create products
	for _, product := range products {
		err := repo.Create(ctx, product)
		if err != nil {
			log.Printf("Failed to create product in SQL: %v", err)
		}
	}
	fmt.Println("✓ Created products in SQL database")

	// Query operations
	count, err := repo.Count(ctx)
	if err != nil {
		log.Printf("Failed to count SQL products: %v", err)
	} else {
		fmt.Printf("✓ SQL database has %d products\n", count)
	}

	// Complex query
	expensiveProducts, err := repo.Query(ctx,
		gpa.Where("price", gpa.OpGreaterThan, 50.0),
		gpa.Where("in_stock", gpa.OpEqual, true),
		gpa.OrderBy("price", gpa.OrderDesc),
	)
	if err != nil {
		log.Printf("Failed to query expensive products: %v", err)
	} else {
		fmt.Printf("✓ Found %d expensive in-stock products\n", len(expensiveProducts))
	}

	// Transaction example
	err = repo.Transaction(ctx, func(tx gpa.Transaction[Product]) error {
		// Update price for all electronics
		return tx.UpdatePartial(ctx, "1", map[string]interface{}{
			"price": 1199.99,
		})
	})
	if err != nil {
		log.Printf("SQL transaction failed: %v", err)
	} else {
		fmt.Println("✓ SQL transaction completed")
	}

	// Raw SQL if supported
	if sqlRepo, ok := repo.(gpa.SQLRepository[Product]); ok {
		rawProducts, err := sqlRepo.FindBySQL(ctx, "SELECT * FROM products WHERE category = ?", []interface{}{"Electronics"})
		if err != nil {
			log.Printf("Failed to execute raw SQL: %v", err)
		} else {
			fmt.Printf("✓ Raw SQL found %d electronics\n", len(rawProducts))
		}
	}
}

func performMongoOperations(ctx context.Context, repo gpa.Repository[Product], products []*Product) {
	// Create products
	err := repo.CreateBatch(ctx, products)
	if err != nil {
		log.Printf("Failed to create products in MongoDB: %v", err)
	} else {
		fmt.Println("✓ Created products in MongoDB")
	}

	// Document-specific operations
	if docRepo, ok := repo.(gpa.DocumentRepository[Product]); ok {
		// Find by document structure
		query := map[string]interface{}{
			"category": "Electronics",
			"price":    map[string]interface{}{"$gte": 1000},
		}
		electronics, err := docRepo.FindByDocument(ctx, query)
		if err != nil {
			log.Printf("Failed to find electronics: %v", err)
		} else {
			fmt.Printf("✓ Found %d electronics with MongoDB query\n", len(electronics))
		}

		// Aggregation example
		pipeline := []map[string]interface{}{
			{
				"$group": map[string]interface{}{
					"_id":        "$category",
					"avgPrice":   map[string]interface{}{"$avg": "$price"},
					"count":      map[string]interface{}{"$sum": 1},
					"totalValue": map[string]interface{}{"$sum": "$price"},
				},
			},
			{
				"$sort": map[string]interface{}{
					"avgPrice": -1,
				},
			},
		}
		results, err := docRepo.Aggregate(ctx, pipeline)
		if err != nil {
			log.Printf("Failed to aggregate: %v", err)
		} else {
			fmt.Printf("✓ Aggregation returned %d category summaries\n", len(results))
			for _, result := range results {
				if category, ok := result["_id"].(string); ok {
					if avgPrice, ok := result["avgPrice"].(float64); ok {
						fmt.Printf("  %s: avg price $%.2f\n", category, avgPrice)
					}
				}
			}
		}

		// Create index
		keys := map[string]interface{}{
			"category": 1,
			"price":    -1,
		}
		err = docRepo.CreateIndex(ctx, keys, false)
		if err != nil {
			log.Printf("Failed to create index: %v", err)
		} else {
			fmt.Println("✓ Created compound index on category and price")
		}
	}

	// Count and existence checks
	count, err := repo.Count(ctx)
	if err != nil {
		log.Printf("Failed to count MongoDB products: %v", err)
	} else {
		fmt.Printf("✓ MongoDB has %d products\n", count)
	}
}

func performRedisOperations(ctx context.Context, repo gpa.Repository[CacheEntry]) {
	// Key-Value operations
	if kvRepo, ok := repo.(gpa.BasicKeyValueRepository[CacheEntry]); ok {
		// Set individual values
		entries := map[string]*CacheEntry{
			"user:1:profile": {Key: "user:1:profile", Value: `{"name":"John","email":"john@example.com"}`},
			"product:1:info": {Key: "product:1:info", Value: `{"name":"Laptop","price":1299.99}`},
			"session:abc123": {Key: "session:abc123", Value: `{"userId":"1","expires":"2024-12-31"}`},
		}

		for key, entry := range entries {
			err := kvRepo.Set(ctx, key, entry)
			if err != nil {
				log.Printf("Failed to set Redis key %s: %v", key, err)
			}
		}
		fmt.Println("✓ Set individual cache entries in Redis")

		// Batch operations if supported
		if batchKV, ok := kvRepo.(gpa.BatchKeyValueRepository[CacheEntry]); ok {
			err := batchKV.MSet(ctx, entries)
			if err != nil {
				log.Printf("Failed to set multiple Redis keys: %v", err)
			} else {
				fmt.Println("✓ Batch set cache entries in Redis")
			}

			// Get multiple values
			keys := []string{"user:1:profile", "product:1:info", "nonexistent:key"}
			results, err := batchKV.MGet(ctx, keys)
			if err != nil {
				log.Printf("Failed to get multiple Redis keys: %v", err)
			} else {
				fmt.Printf("✓ Retrieved %d cache entries from Redis\n", len(results))
			}
		} else {
			fmt.Println("✓ Batch operations not supported by this repository")
		}

		// Key operations
		exists, err := kvRepo.KeyExists(ctx, "user:1:profile")
		if err != nil {
			log.Printf("Failed to check key existence: %v", err)
		} else {
			fmt.Printf("✓ Key 'user:1:profile' exists: %t\n", exists)
		}

		// Pattern-based key retrieval not available in BasicKeyValueRepository
		fmt.Println("✓ Pattern-based key retrieval would need PatternKeyValueRepository interface")

		// TTL operations if supported
		if advancedKV, ok := kvRepo.(gpa.TTLKeyValueRepository[CacheEntry]); ok {
			err := advancedKV.SetWithTTL(ctx, "temp:key", &CacheEntry{Key: "temp:key", Value: "temporary"}, time.Minute*5)
			if err != nil {
				log.Printf("Failed to set key with TTL: %v", err)
			} else {
				fmt.Println("✓ Set temporary key with 5-minute TTL")
			}

			ttl, err := advancedKV.GetTTL(ctx, "temp:key")
			if err != nil {
				log.Printf("Failed to get TTL: %v", err)
			} else {
				fmt.Printf("✓ Temporary key TTL: %v\n", ttl)
			}
		}

		// Atomic operations would go here when supported
		fmt.Println("✓ Atomic operations not implemented in this example")
	}
}