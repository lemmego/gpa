// Package gpa provides a unified persistence API for Go applications
// supporting both SQL and NoSQL databases through adapter implementations.
//
// # Type-Safe Approach (Recommended)
//
// GPA provides compile-time type safety for all database operations.
// This is the RECOMMENDED approach for new applications:
//
//	type User struct {
//	    ID   int    `json:"id"`
//	    Name string `json:"name"`
//	}
//
//	// Create a type-safe provider
//	provider, err := gpa.NewProvider[User]("postgres", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get a type-safe repository - no reflection needed!
//	repo := provider.Repository()
//
//	// All operations return strongly-typed results
//	user, err := repo.FindByID(ctx, 123)    // Returns *User directly
//	users, err := repo.FindAll(ctx)         // Returns []*User directly
//
// # Benefits of Type-Safe Approach
//
// • Compile-time type safety - catch errors at build time
// • No interface{} conversions or type assertions
// • Better IDE support with autocompletion and refactoring
// • Improved performance - no runtime reflection for type checking
// • Cleaner, more readable code
//
// # Provider Support
//
// All providers support the type-safe approach:
// • GORM (PostgreSQL, MySQL, SQLite, SQL Server)
// • Bun (PostgreSQL, MySQL, SQLite)  
// • MongoDB
// • Redis
//
// Choose the type-safe approach for new development and gradually migrate
// existing code for the best developer experience.
package gpa