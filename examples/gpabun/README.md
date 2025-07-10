# Bun Provider Example

This example demonstrates the Bun provider for the Go Persistence API (GPA), showcasing advanced SQL operations, query building, and Bun's powerful ORM capabilities.

## Features Demonstrated

### Core SQL Operations
- **Schema Migration**: Automatic table creation and schema management
- **Advanced Queries**: Complex SQL with joins, subqueries, and window functions
- **Relationship Mapping**: Foreign key relationships and eager loading
- **Index Management**: Creating and optimizing database indexes
- **Batch Operations**: Efficient bulk operations for better performance

### Bun-Specific Features
- **Query Builder**: Fluent, type-safe query construction
- **Struct Mapping**: Automatic mapping between Go structs and database tables
- **Relation Loading**: Efficient loading of related data
- **Raw SQL Support**: Direct SQL execution when needed
- **Hook System**: Before/after operation hooks
- **Debug Mode**: Query logging and performance monitoring

### Advanced Patterns
- **Window Functions**: Advanced analytics with SQL window functions
- **Aggregation**: Complex aggregation queries and analytics
- **Transaction Management**: ACID transactions with rollback support
- **Performance Optimization**: Efficient queries and batch operations
- **Schema Introspection**: Runtime schema information and metadata

## Prerequisites

- Go 1.18+
- SQLite3 (included with Go)

Optional for other databases:
- PostgreSQL 12+
- MySQL 8.0+
- SQL Server 2019+

## Quick Start

```bash
# Navigate to this directory
cd examples/gpabun

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
    Database: "bun_example.db",
    Options: map[string]interface{}{
        "bun": map[string]interface{}{
            "debug":           true,
            "log_slow_query":  true,
            "slow_query_time": "100ms",
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
    Database: "gpa_bun_example",
    SSL: gpa.SSLConfig{
        Enabled: true,
        Mode:    "require",
    },
    Options: map[string]interface{}{
        "bun": map[string]interface{}{
            "debug": true,
            "max_open_conns": 25,
            "max_idle_conns": 25,
        },
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
    Database: "gpa_bun_example",
    Options: map[string]interface{}{
        "bun": map[string]interface{}{
            "debug": true,
            "parse_time": true,
        },
    },
}
```

## Code Structure

### Entity Definitions
```go
type User struct {
    ID        int64     `bun:"id,pk,autoincrement"`
    Name      string    `bun:"name,notnull"`
    Email     string    `bun:"email,unique,notnull"`
    Age       int       `bun:"age,notnull"`
    IsActive  bool      `bun:"is_active,default:true"`
    Salary    *float64  `bun:"salary,nullzero"` // Nullable field
    CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
    UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`
    Profile   *Profile  `bun:"rel:has-one,join:id=user_id"`
    Posts     []*Post   `bun:"rel:has-many,join:id=user_id"`
}
```

### Repository Operations
```go
// Type-safe provider creation
provider, err := gpabun.NewTypeSafeProvider[User](config)
repo := provider.Repository()

// Basic CRUD operations
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
        `SELECT u.*, p.bio FROM users u 
         LEFT JOIN profiles p ON u.id = p.user_id 
         WHERE u.age > ? AND u.is_active = ?`, 
        []interface{}{25, true})
    
    // Execute raw SQL commands
    result, err := sqlRepo.ExecSQL(ctx, 
        "UPDATE users SET updated_at = ? WHERE id = ?", 
        time.Now(), userID)
    
    // Create indexes
    err := sqlRepo.CreateIndex(ctx, []string{"email"}, true)
    
    // Load relationships
    users, err := sqlRepo.FindWithRelations(ctx, []string{"Profile", "Posts"})
}
```

### Migration Operations
```go
if migratableRepo, ok := repo.(gpa.MigratableRepository[User]); ok {
    // Auto-migrate schema
    err := migratableRepo.MigrateTable(ctx)
    
    // Get migration status
    status, err := migratableRepo.GetMigrationStatus(ctx)
    
    // Get table information
    tableInfo, err := migratableRepo.GetTableInfo(ctx)
}
```

## Key Concepts

### Bun Tags
Bun uses struct tags to control database mapping:

```go
type User struct {
    ID        int64     `bun:"id,pk,autoincrement"`        // Primary key with auto-increment
    Name      string    `bun:"name,notnull"`               // Not null constraint
    Email     string    `bun:"email,unique,notnull"`       // Unique constraint
    Age       int       `bun:"age,notnull"`                // Not null
    IsActive  bool      `bun:"is_active,default:true"`     // Default value
    Salary    *float64  `bun:"salary,nullzero"`            // Nullable field
    CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
    Profile   *Profile  `bun:"rel:has-one,join:id=user_id"` // One-to-one relationship
    Posts     []*Post   `bun:"rel:has-many,join:id=user_id"` // One-to-many relationship
}
```

### Type Safety
All operations are type-safe at compile time:
```go
// This won't compile if User doesn't match the repository type
var user User
err := repo.Create(ctx, &user)
```

### Query Building
Fluent query interface with type safety:
```go
users, err := repo.Query(ctx,
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("is_active", gpa.OpEqual, true),
    gpa.WhereNotNull("salary"),
    gpa.OrderBy("name", gpa.OrderAsc),
    gpa.Limit(10),
    gpa.Offset(20),
)
```

### Error Handling
Structured errors with Bun-specific context:
```go
if gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
    // Handle record not found
}
if gpa.IsErrorType(err, gpa.ErrorTypeConstraintViolation) {
    // Handle unique constraint violation
}
```

### Transactions
Automatic transaction management:
```go
err := repo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    if err := tx.Create(ctx, &user1); err != nil {
        return err // Automatic rollback
    }
    if err := tx.Update(ctx, &user2); err != nil {
        return err // Automatic rollback
    }
    return nil // Commit
})
```

## Advanced Features

### Complex Queries
```go
// Subquery example
topPosters, err := sqlRepo.FindBySQL(ctx, `
    SELECT u.* FROM users u
    WHERE u.id IN (
        SELECT p.user_id 
        FROM posts p 
        WHERE p.published = ? 
        GROUP BY p.user_id 
        HAVING COUNT(*) >= ?
    )
    ORDER BY u.name
`, []interface{}{true, 2})
```

### Window Functions
```go
// Ranking with window functions
rankedUsers, err := sqlRepo.FindBySQL(ctx, `
    SELECT 
        name,
        age,
        salary,
        ROW_NUMBER() OVER (ORDER BY age DESC) as age_rank,
        RANK() OVER (ORDER BY COALESCE(salary, 0) DESC) as salary_rank
    FROM users 
    WHERE is_active = ?
    ORDER BY age_rank
`, []interface{}{true})
```

### Aggregation
```go
// Complex aggregation
stats, err := sqlRepo.FindBySQL(ctx, `
    SELECT 
        COUNT(*) as total_users,
        COUNT(CASE WHEN is_active THEN 1 END) as active_users,
        AVG(age) as avg_age,
        AVG(CASE WHEN salary IS NOT NULL THEN salary END) as avg_salary
    FROM users
`, []interface{}{})
```

### Relationship Loading
```go
// Load users with profiles and posts
users, err := sqlRepo.FindWithRelations(ctx, 
    []string{"Profile", "Posts"}, 
    gpa.Where("is_active", gpa.OpEqual, true))

// Access related data
for _, user := range users {
    fmt.Printf("User: %s\n", user.Name)
    if user.Profile != nil {
        fmt.Printf("  Bio: %s\n", user.Profile.Bio)
    }
    fmt.Printf("  Posts: %d\n", len(user.Posts))
}
```

## Performance Tips

1. **Use Indexes**: Create indexes on frequently queried columns
2. **Batch Operations**: Use `CreateBatch()` for multiple inserts
3. **Limit Results**: Use `Limit()` and `Offset()` for pagination
4. **Select Specific Fields**: Use raw SQL to select only needed columns
5. **Optimize Relationships**: Use `FindWithRelations()` to avoid N+1 queries

## Testing

The example includes comprehensive testing scenarios:

- Schema migration and rollback
- Complex query patterns
- Transaction handling
- Error conditions
- Performance benchmarks

## Common Patterns

### Repository Pattern
```go
type UserService struct {
    repo gpa.Repository[User]
}

func (s *UserService) GetActiveUsers(ctx context.Context) ([]*User, error) {
    return s.repo.Query(ctx,
        gpa.Where("is_active", gpa.OpEqual, true),
        gpa.OrderBy("created_at", gpa.OrderDesc),
    )
}
```

### Service Layer
```go
func (s *UserService) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return s.repo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
        if err := tx.Create(ctx, user); err != nil {
            return err
        }
        
        profile.UserID = user.ID
        return s.profileRepo.Create(ctx, profile)
    })
}
```

### Query Builder Pattern
```go
func (s *UserService) SearchUsers(ctx context.Context, filters UserFilters) ([]*User, error) {
    conditions := []gpa.QueryOption{}
    
    if filters.MinAge > 0 {
        conditions = append(conditions, gpa.Where("age", gpa.OpGreaterThanOrEqual, filters.MinAge))
    }
    
    if filters.IsActive != nil {
        conditions = append(conditions, gpa.Where("is_active", gpa.OpEqual, *filters.IsActive))
    }
    
    if filters.Name != "" {
        conditions = append(conditions, gpa.WhereLike("name", filters.Name+"%"))
    }
    
    conditions = append(conditions, gpa.OrderBy("name", gpa.OrderAsc))
    
    return s.repo.Query(ctx, conditions...)
}
```

## Troubleshooting

### Common Issues

1. **Migration Errors**: Check database permissions and schema conflicts
2. **Constraint Violations**: Handle unique constraints and foreign key errors
3. **Performance Issues**: Add indexes and optimize queries
4. **Connection Issues**: Verify database connection settings

### Debug Tips

1. **Enable Debug Mode**: Set `debug: true` in Bun options
2. **Log Slow Queries**: Use `log_slow_query` and `slow_query_time` options
3. **Check Query Plans**: Use `EXPLAIN QUERY PLAN` for SQLite
4. **Monitor Connections**: Watch connection pool usage

### Performance Debugging

```go
// Enable query logging
config := gpa.Config{
    Driver:   "sqlite",
    Database: "app.db",
    Options: map[string]interface{}{
        "bun": map[string]interface{}{
            "debug":           true,
            "log_slow_query":  true,
            "slow_query_time": "100ms",
        },
    },
}
```

## Environment Variables

```bash
# Database configuration
export DB_DRIVER="postgres"
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="user"
export DB_PASSWORD="password"
export DB_NAME="gpa_bun_example"

# SSL configuration
export DB_SSL_MODE="require"

# Connection pooling
export DB_MAX_OPEN_CONNS="25"
export DB_MAX_IDLE_CONNS="25"
export DB_CONN_MAX_LIFETIME="5m"
```

## Production Considerations

1. **Connection Pooling**: Configure appropriate pool sizes
2. **Migration Strategy**: Use versioned migrations in production
3. **Error Handling**: Implement proper error logging and monitoring
4. **Performance Monitoring**: Track query performance and optimize
5. **Security**: Use parameterized queries and validate inputs

## Next Steps

- Explore other providers: MongoDB, Redis, GORM
- Review the main documentation for architecture details
- Check the test files for additional usage patterns
- Consider implementing custom query builders for complex use cases
- Explore Bun's advanced features like hooks and plugins