# GORM Provider Example

This example demonstrates the GORM provider for the Go Persistence API (GPA), showcasing SQL database operations with type safety and GORM-specific features.

## Features Demonstrated

### Core SQL Operations
- **Schema Migration**: Automatic table creation and updates
- **Index Management**: Creating and managing database indexes
- **CRUD Operations**: Type-safe create, read, update, delete operations
- **Raw SQL**: Direct SQL queries when needed
- **Relationships**: Foreign key relationships and preloading
- **Transactions**: ACID transactions with automatic rollback

### GORM-Specific Features
- **Auto-Migration**: Automatic schema synchronization
- **Associations**: Foreign key relationships and eager loading
- **Hooks**: Before/after operation hooks
- **Soft Deletes**: Logical deletion of records
- **Batch Operations**: Efficient bulk operations
- **Query Builder**: Fluent query interface

### Advanced Patterns
- **Complex Queries**: Multi-condition WHERE clauses, JOINs
- **Aggregation**: COUNT, SUM, AVG operations
- **Pagination**: LIMIT and OFFSET for large datasets
- **Performance**: Optimized queries and batch operations

## Prerequisites

- Go 1.18+
- SQLite3 (included with Go)

Optional for other databases:
- PostgreSQL
- MySQL
- SQL Server

## Quick Start

```bash
# Navigate to this directory
cd examples/gpagorm

# Install dependencies
go mod tidy

# Run the example
go run main.go
```

## Database Configuration

The example uses SQLite by default for simplicity:

```go
config := gpa.Config{
    Driver:   "sqlite",
    Database: "gorm_example.db",
    Options: map[string]interface{}{
        "gorm": map[string]interface{}{
            "log_level":      "info",
            "singular_table": false,
        },
    },
}
```

### Other Database Examples

#### PostgreSQL
```go
config := gpa.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Username: "user",
    Password: "password",
    Database: "gpa_example",
    SSL: gpa.SSLConfig{
        Enabled: true,
        Mode:    "require",
    },
}
```

#### MySQL
```go
config := gpa.Config{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     3306,
    Username: "user",
    Password: "password",
    Database: "gpa_example",
}
```

## Code Structure

### Entity Definitions
```go
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string    `gorm:"size:255;not null"`
    Email     string    `gorm:"uniqueIndex;size:255;not null"`
    Age       int       `gorm:"not null"`
    IsActive  bool      `gorm:"default:true"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
    Profile   *Profile  `gorm:"foreignKey:UserID"`
}
```

### Repository Operations
```go
// Type-safe provider creation
provider, err := gpagorm.NewTypeSafeProvider[User](config)
repo := provider.Repository()

// CRUD operations
err := repo.Create(ctx, &user)
user, err := repo.FindByID(ctx, 1)
err := repo.Update(ctx, &user)
err := repo.Delete(ctx, 1)
```

### SQL-Specific Operations
```go
if sqlRepo, ok := repo.(gpa.SQLRepository[User]); ok {
    // Raw SQL queries
    users, err := sqlRepo.FindBySQL(ctx, 
        "SELECT * FROM users WHERE age > ?", 
        []interface{}{25})
    
    // Table management
    err := sqlRepo.CreateTable(ctx)
    err := sqlRepo.CreateIndex(ctx, []string{"email"}, true)
}
```

### Migration Operations
```go
if migratableRepo, ok := repo.(gpa.MigratableRepository[User]); ok {
    // Auto-migrate schema
    err := migratableRepo.MigrateTable(ctx)
    
    // Check migration status
    status, err := migratableRepo.GetMigrationStatus(ctx)
    
    // Get table information
    info, err := migratableRepo.GetTableInfo(ctx)
}
```

## Key Concepts

### Type Safety
All operations are type-safe at compile time:
```go
// This won't compile if User doesn't match the repository type
var user User
err := repo.Create(ctx, &user)
```

### Error Handling
Structured errors with type information:
```go
if gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
    // Handle not found error
}
```

### Transactions
Automatic transaction management:
```go
err := repo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    if err := tx.Create(ctx, &user1); err != nil {
        return err // Automatic rollback
    }
    return tx.Update(ctx, &user2)
})
```

### Query Building
Fluent query interface:
```go
users, err := repo.Query(ctx,
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("active", gpa.OpEqual, true),
    gpa.OrderBy("name", gpa.OrderAsc),
    gpa.Limit(10),
    gpa.Offset(20),
)
```

## Performance Tips

1. **Use Batch Operations**: `CreateBatch()` for multiple inserts
2. **Index Wisely**: Create indexes on frequently queried columns
3. **Limit Results**: Use `Limit()` and `Offset()` for pagination
4. **Preload Relations**: Use `FindWithRelations()` to avoid N+1 queries
5. **Raw SQL**: Use raw SQL for complex queries when needed

## Testing

The example includes comprehensive error handling and edge cases:

- Connection failures
- Constraint violations
- Transaction rollbacks
- Migration errors
- Index conflicts

## Common Patterns

### Repository Pattern
```go
type UserService struct {
    repo gpa.Repository[User]
}

func (s *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
    user := &User{Name: name, Email: email}
    err := s.repo.Create(ctx, user)
    return user, err
}
```

### Service Layer
```go
func (s *UserService) GetActiveUsers(ctx context.Context, limit int) ([]*User, error) {
    return s.repo.Query(ctx,
        gpa.Where("is_active", gpa.OpEqual, true),
        gpa.OrderBy("created_at", gpa.OrderDesc),
        gpa.Limit(limit),
    )
}
```

## Troubleshooting

### Common Issues

1. **Migration Errors**: Check database permissions and existing schema
2. **Connection Issues**: Verify database server is running and accessible
3. **Type Errors**: Ensure Go 1.18+ for generics support
4. **Index Conflicts**: Handle duplicate index creation gracefully

### Debug Tips

1. Enable GORM logging: `"log_level": "info"`
2. Check migration status before operations
3. Use raw SQL for debugging complex queries
4. Monitor transaction rollbacks

## Next Steps

- Explore other providers: MongoDB, Redis, Bun
- Review the main documentation for architecture details
- Check the test files for additional usage patterns
- Consider implementing custom providers for specific needs