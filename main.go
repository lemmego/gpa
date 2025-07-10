// Package gpa provides a unified persistence API for Go applications
// supporting both SQL and NoSQL databases through adapter implementations.
//
// # Unified Provider API (Recommended)
//
// GPA provides compile-time type safety for all database operations using
// a unified provider pattern. This is the RECOMMENDED approach for all applications:
//
//	type User struct {
//	    ID   int    `json:"id"`
//	    Name string `json:"name"`
//	}
//
//	type Post struct {
//	    ID     int    `json:"id"`
//	    UserID int    `json:"user_id"`
//	    Title  string `json:"title"`
//	}
//
//	// Create a single provider for your database
//	provider, err := gpagorm.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Create multiple type-safe repositories from the same provider
//	userRepo := gpagorm.GetRepository[User](provider)
//	postRepo := gpagorm.GetRepository[Post](provider)
//
//	// All operations return strongly-typed results
//	user, err := userRepo.FindByID(ctx, 123)    // Returns *User directly
//	users, err := userRepo.FindAll(ctx)         // Returns []*User directly
//	posts, err := postRepo.FindAll(ctx)         // Returns []*Post directly
//
// # Benefits of Unified Provider API
//
// • Single provider per database connection - efficient resource usage
// • Multiple type-safe repositories from one provider
// • Compile-time type safety - catch errors at build time
// • No interface{} conversions or type assertions
// • Better IDE support with autocompletion and refactoring
// • Improved performance - no runtime reflection for type checking
// • Cleaner, more readable code
//
// # Provider Support
//
// All providers support the unified API:
// • GORM: gpagorm.NewProvider(config) + gpagorm.GetRepository[T](provider)
// • Bun: gpabun.NewProvider(config) + gpabun.GetRepository[T](provider)
// • MongoDB: gpamongo.NewProvider(config) + gpamongo.GetRepository[T](provider)
// • Redis: gparedis.NewProvider(config) + gparedis.GetRepository[T](provider)
//
// Each provider supports their respective database drivers:
// • GORM: PostgreSQL, MySQL, SQLite, SQL Server
// • Bun: PostgreSQL, MySQL, SQLite
// • MongoDB: MongoDB
// • Redis: Redis
//
// Use the unified provider API for all new development for the best
// developer experience and resource efficiency.
package gpa