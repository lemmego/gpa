# GPA Examples

This directory contains comprehensive examples demonstrating the capabilities of the Go Persistence API (GPA) framework.

## Examples Overview

### 1. Basic Usage (`basic_usage.go`)
Demonstrates fundamental CRUD operations and basic query patterns:
- Creating single and batch entities
- Finding entities by ID and with conditions
- Updating entities (full and partial updates)
- Deleting entities
- Counting and existence checks
- Basic transactions
- Raw SQL queries

**Key Concepts:**
- Type-safe repositories
- Query options and conditions
- Error handling
- Transaction management

### 2. Multi-Provider (`multi_provider.go`)
Shows how to use different database providers for different use cases:
- SQL operations with GORM (SQLite)
- Document operations with MongoDB
- Key-value operations with Redis
- Provider feature comparison
- Database-specific operations

**Key Concepts:**
- Provider abstraction
- Database-specific interfaces
- Feature detection
- Multi-database architectures

### 3. Advanced Queries (`advanced_queries.go`)
Explores complex query patterns and relationships:
- Complex WHERE conditions
- Comparison and logical operators
- Sorting, limiting, and pagination
- Field selection and aggregation
- Raw SQL for complex queries
- Performance optimizations

**Key Concepts:**
- Query builders
- Complex conditions
- Aggregation patterns
- Performance considerations

## Running the Examples

### Prerequisites

1. **Go 1.18+**: The framework requires Go generics support
2. **Database Dependencies** (optional, examples will skip unavailable databases):
   - SQLite: Built-in support
   - MongoDB: `brew install mongodb-community` or Docker
   - Redis: `brew install redis` or Docker

### Basic Setup

```bash
# Clone the repository
git clone <repository-url>
cd gpa

# Install dependencies
go mod tidy

# Run basic usage example
go run examples/basic_usage.go

# Run multi-provider example (with optional databases)
go run examples/multi_provider.go

# Run advanced queries example
go run examples/advanced_queries.go
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
export MONGODB_TEST_URL="mongodb://localhost:27017"

# Redis
export REDIS_TEST_URL="redis://localhost:6379"

# Run examples with custom connections
go run examples/multi_provider.go
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
// SQL-specific operations
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