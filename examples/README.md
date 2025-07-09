# GPA Examples

This directory contains comprehensive examples demonstrating the capabilities of the Go Persistence API (GPA) framework.

## Examples Overview

### 1. GORM Provider (`gpagorm/`)
Demonstrates SQL operations using GORM with SQLite:
- Schema migration and table management
- Basic CRUD operations with type safety
- Complex SQL queries and joins
- Relationship management and preloading
- Raw SQL queries for advanced use cases
- Transaction support with rollbacks
- Index creation and performance optimization

**Key Features:**
- Type-safe SQL operations
- Automatic schema migration
- Relationship mapping
- Query optimization

### 2. MongoDB Provider (`gpamongo/`)
Shows document database operations with MongoDB:
- Document creation with nested structures
- MongoDB-specific queries and aggregations
- Text search and geospatial queries
- Index management for performance
- Aggregation pipelines for analytics
- Document updates with MongoDB operators
- Transaction support in MongoDB

**Key Features:**
- Document-oriented operations
- Aggregation framework
- Geospatial queries
- Full-text search

### 3. Redis Provider (`gparedis/`)
Demonstrates key-value operations with Redis:
- Key-value storage with TTL support
- Caching patterns and session management
- Redis-specific data structures
- Performance optimization techniques
- Atomic operations and counters
- Pattern matching and key operations

**Key Features:**
- High-performance key-value storage
- TTL and expiration management
- Atomic operations
- Caching strategies

### 4. Bun Provider (`gpabun/`)
Explores advanced SQL operations with Bun:
- Advanced query patterns and CTEs
- Complex aggregations and analytics
- Relationship queries and joins
- Performance optimization
- Raw SQL with type safety
- Transaction management
- Window functions and analytics

**Key Features:**
- Advanced SQL query building
- High-performance queries
- Complex aggregations
- Analytics capabilities

## Running the Examples

### Prerequisites

1. **Go 1.18+**: The framework requires Go generics support
2. **Database Dependencies** (optional, examples will skip unavailable databases):
   - SQLite: Built-in support (used by GORM and Bun examples)
   - MongoDB: `brew install mongodb-community` or Docker
   - Redis: `brew install redis` or Docker

### Basic Setup

```bash
# Clone the repository
git clone <repository-url>
cd gpa

# Install dependencies
go mod tidy

# Run SQL examples (work without external databases)
go run ./examples/gpagorm    # GORM with SQLite (in-memory)
go run ./examples/gpabun     # Bun with SQLite (temporary file)

# Run database examples (require running databases)
go run ./examples/gpamongo   # MongoDB example
go run ./examples/gparedis   # Redis example
```

### Database Setup (Optional)

#### MongoDB
```bash
# Using Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Using Homebrew (macOS)
brew install mongodb-community
brew services start mongodb-community
```

#### Redis
```bash
# Using Docker
docker run -d -p 6379:6379 --name redis redis:latest

# Using Homebrew (macOS)
brew install redis
brew services start redis
```

### Environment Variables

You can customize database connections using environment variables:

```bash
# MongoDB
export MONGODB_URL="mongodb://localhost:27017"

# Redis  
export REDIS_URL="redis://localhost:6379"

# Run examples with custom connections
go run ./examples/gpamongo   # Uses MONGODB_URL if set
go run ./examples/gparedis   # Uses REDIS_URL if set
```

## Example Code Patterns

### 1. Provider Setup

```go
// Type-safe provider creation
provider, err := gpagorm.NewTypeSafeProvider[User](config)
if err != nil {
    log.Fatalf("Failed to create provider: %v", err)
}
defer provider.Close()

// Get repository
repo := provider.Repository()
```

### 2. CRUD Operations

```go
// Create
user := &User{Name: "John", Email: "john@example.com"}
err := repo.Create(ctx, user)

// Read
found, err := repo.FindByID(ctx, user.ID)

// Update
user.Name = "John Smith"
err = repo.Update(ctx, user)

// Delete
err = repo.Delete(ctx, user.ID)
```

### 3. Query Building

```go
// Complex queries with type safety
users, err := repo.Query(ctx,
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("active", gpa.OpEqual, true),
    gpa.OrderBy("name", gpa.OrderAsc),
    gpa.Limit(10),
    gpa.Offset(20),
)
```

### 4. Transactions

```go
err := repo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    if err := tx.Create(ctx, &user1); err != nil {
        return err
    }
    if err := tx.Update(ctx, &user2); err != nil {
        return err
    }
    return nil
})
```

### 5. Database-Specific Operations

```go
// SQL-specific operations (GORM/Bun)
if sqlRepo, ok := repo.(gpa.SQLRepository[User]); ok {
    users, err := sqlRepo.FindBySQL(ctx, "SELECT * FROM users WHERE age > ?", []interface{}{18})
}

// Document-specific operations (MongoDB)
if docRepo, ok := repo.(gpa.DocumentRepository[User]); ok {
    results, err := docRepo.Aggregate(ctx, pipeline)
}

// Key-value operations (Redis)
if kvRepo, ok := repo.(gpa.KeyValueRepository[CacheEntry]); ok {
    err := kvRepo.Set(ctx, "key", &entry)
}
```

## Example Features

### SQL Examples (GORM & Bun)
- **In-memory databases**: Fast execution without external dependencies
- **Schema migration**: Automatic table creation and management
- **Relationships**: Foreign keys and data preloading
- **Complex queries**: JOINs, subqueries, and aggregations
- **Raw SQL**: Direct SQL execution when needed
- **Transactions**: ACID compliance with rollback support

### Document Example (MongoDB)
- **Nested documents**: Complex data structures
- **Aggregation pipelines**: Data processing and analytics
- **Geospatial queries**: Location-based searches
- **Text search**: Full-text search capabilities
- **Indexes**: Performance optimization

### Key-Value Example (Redis)
- **Caching patterns**: Session management and data caching
- **TTL support**: Automatic expiration handling
- **Atomic operations**: Counters and atomic updates
- **Pattern matching**: Key discovery and management

## Key Features Demonstrated

### Type Safety
- Compile-time type checking
- Generic repository interfaces
- Type-safe query building

### Database Abstraction
- Unified API across different databases
- Provider-specific optimizations
- Feature detection and graceful degradation

### Query Flexibility
- Fluent query builder interface
- Support for complex conditions
- Raw SQL escape hatch

### Transaction Support
- ACID transactions where supported
- Nested transaction handling
- Automatic rollback on errors

### Error Handling
- Structured error types
- Error type detection
- Provider-specific error mapping

## Best Practices Shown

1. **Resource Management**: Always close providers and handle errors
2. **Configuration**: Use structured configuration with sensible defaults
3. **Feature Detection**: Check for interface support before using advanced features
4. **Error Handling**: Use typed errors for better error handling
5. **Testing**: Examples include error scenarios and edge cases
6. **Performance**: Use appropriate query patterns and batch operations

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Ensure database services are running
   - Check connection URLs and credentials
   - Verify network connectivity

2. **Type Errors**
   - Ensure Go 1.18+ for generics support
   - Check struct tags match database schema
   - Verify interface implementations

3. **Migration Issues**
   - Ensure proper table permissions
   - Check for conflicting schema changes
   - Verify foreign key constraints

### Getting Help

- Check the main README for architecture details
- Review test files for additional examples
- See provider-specific documentation
- Open issues for bugs or feature requests

## Contributing

When adding new examples:

1. Follow the existing code structure
2. Include comprehensive error handling
3. Add comments explaining key concepts
4. Test with multiple providers when applicable
5. Update this README with new examples