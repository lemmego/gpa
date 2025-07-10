# GPA (Go Persistence API)

[![Go Reference](https://pkg.go.dev/badge/github.com/lemmego/gpa.svg)](https://pkg.go.dev/github.com/lemmego/gpa)
[![Go Report Card](https://goreportcard.com/badge/github.com/lemmego/gpa)](https://goreportcard.com/report/github.com/lemmego/gpa)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A unified, type-safe persistence API for Go applications supporting both SQL and NoSQL databases through provider implementations with an advanced multi-provider registry system.

## ✨ Key Features

- **🔒 Compile-time Type Safety** - No `interface{}` or type assertions needed
- **🚀 High Performance** - Zero reflection overhead in runtime operations
- **🔄 Unified API** - Same interface for SQL, NoSQL, and key-value databases
- **📦 Multiple Provider Support** - GORM, Bun, MongoDB, Redis out of the box
- **🎯 Advanced Provider Registry** - Manage multiple database connections with type inference
- **🧰 Rich Query Builder** - Fluent, type-safe query construction
- **⚡ Connection Pooling** - Built-in connection management and health checks
- **🔍 Generic Repositories** - Type-safe repository pattern with Go generics

## 🚀 Quick Start

### Installation

```bash
go get github.com/lemmego/gpa
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/lemmego/gpa"
    "github.com/lemmego/gpagorm"
)

type User struct {
    ID   int    `gorm:"primaryKey" json:"id"`
    Name string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Configure your database
    config := gpa.Config{
        Driver:   "postgres",
        Host:     "localhost",
        Port:     5432,
        Database: "myapp",
        Username: "user",
        Password: "password",
    }
    
    // Create provider
    provider, err := gpagorm.NewProvider(config)
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()
    
    // Register provider in global registry with type inference
    gpa.RegisterDefault[*gpagorm.Provider](provider)
    
    // Get type-safe repository
    userRepo := gpagorm.GetRepository[User](provider)
    
    ctx := context.Background()
    
    // Create user
    user := &User{Name: "John Doe", Email: "john@example.com"}
    err = userRepo.Create(ctx, user)
    if err != nil {
        log.Fatal(err)
    }
    
    // Find user - returns *User directly, no type assertions!
    foundUser, err := userRepo.FindByID(ctx, user.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    // Query users with type-safe conditions
    users, err := userRepo.Query(ctx,
        gpa.Where("name", gpa.OpLike, "John%"),
        gpa.OrderBy("created_at", gpa.OrderDesc),
        gpa.Limit(10),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Found %d users", len(users))
}
```

## 🎯 Advanced Provider Registry

GPA features an advanced provider registry that allows you to manage multiple database connections with automatic type inference:

### Register Providers

```go
// Register multiple providers with full type safety
gpa.Register[*gpagorm.Provider]("primary", primaryDB)
gpa.Register[*gpagorm.Provider]("readonly", readonlyDB)
gpa.Register[*gparedis.Provider]("cache", redisCache)
gpa.Register[*gparedis.Provider]("sessions", sessionRedis)
gpa.Register[*gpamongo.Provider]("documents", mongoProvider)

// Register default providers
gpa.RegisterDefault[*gpagorm.Provider](primaryDB)
```

### Retrieve Providers with Type Inference

```go
// Get providers - type is automatically inferred, no strings needed!
primary, err := gpa.Get[*gpagorm.Provider]("primary")
cache := gpa.MustGet[*gparedis.Provider]("cache")
defaultDB, err := gpa.Get[*gpagorm.Provider]() // Gets default

// Get all providers of a specific type
allGormProviders, err := gpa.GetByType[*gpagorm.Provider]()
allRedisProviders, err := gpa.GetByType[*gparedis.Provider]()
```

### Multi-Database Patterns

#### Primary/Replica Pattern
```go
func setupPrimaryReplica() {
    primary, _ := gpagorm.NewProvider(primaryConfig)
    replica, _ := gpagorm.NewProvider(replicaConfig)
    
    gpa.Register[*gpagorm.Provider]("primary", primary)
    gpa.Register[*gpagorm.Provider]("replica", replica)
    gpa.RegisterDefault[*gpagorm.Provider](primary)
}

func useDatabase() {
    // Write operations use primary
    primary := gpa.MustGet[*gpagorm.Provider]("primary")
    userRepo := gpagorm.GetRepository[User](primary)
    userRepo.Create(ctx, &user)
    
    // Read operations can use replica
    replica := gpa.MustGet[*gpagorm.Provider]("replica")
    readRepo := gpagorm.GetRepository[User](replica)
    users := readRepo.FindAll(ctx)
}
```

#### Multi-Tenant Pattern
```go
func setupMultiTenant() {
    tenants := []string{"tenant1", "tenant2", "tenant3"}
    
    for _, tenant := range tenants {
        config := gpa.Config{
            Database: fmt.Sprintf("app_%s", tenant),
            // ... other config
        }
        provider, _ := gpagorm.NewProvider(config)
        gpa.Register[*gpagorm.Provider](tenant, provider)
    }
}

func useTenant(tenantID string) {
    provider := gpa.MustGet[*gpagorm.Provider](tenantID)
    userRepo := gpagorm.GetRepository[User](provider)
    // Operations are scoped to this tenant's database
}
```

#### Hybrid Storage Pattern
```go
func setupHybridStorage() {
    // SQL for transactional data
    sqlProvider, _ := gpagorm.NewProvider(sqlConfig)
    gpa.Register[*gpagorm.Provider]("sql", sqlProvider)
    
    // Redis for caching
    redisProvider, _ := gparedis.NewProvider(redisConfig)
    gpa.Register[*gparedis.Provider]("cache", redisProvider)
    
    // MongoDB for document storage
    mongoProvider, _ := gpamongo.NewProvider(mongoConfig)
    gpa.Register[*gpamongo.Provider]("documents", mongoProvider)
}

func useHybridStorage() {
    // User data in SQL
    sqlProvider := gpa.MustGet[*gpagorm.Provider]("sql")
    userRepo := gpagorm.GetRepository[User](sqlProvider)
    
    // Session data in Redis
    redisProvider := gpa.MustGet[*gparedis.Provider]("cache")
    sessionRepo := gparedis.GetRepository[Session](redisProvider)
    
    // Content in MongoDB
    mongoProvider := gpa.MustGet[*gpamongo.Provider]("documents")
    contentRepo := gpamongo.GetRepository[Content](mongoProvider)
}
```

## 🔧 Supported Providers

### SQL Databases

#### GORM Provider (`gpagorm`)
```go
import "github.com/lemmego/gpagorm"

provider, err := gpagorm.NewProvider(gpa.Config{
    Driver:   "postgres", // postgres, mysql, sqlite, sqlserver
    Host:     "localhost",
    Port:     5432,
    Database: "myapp",
    Username: "user",
    Password: "password",
})

userRepo := gpagorm.GetRepository[User](provider)
```

#### Bun Provider (`gpabun`)
```go
import "github.com/lemmego/gpabun"

provider, err := gpabun.NewProvider(gpa.Config{
    Driver:   "postgres", // postgres, mysql, sqlite
    Host:     "localhost",
    Port:     5432,
    Database: "myapp",
    Username: "user",
    Password: "password",
})

userRepo := gpabun.GetRepository[User](provider)
```

### NoSQL Databases

#### MongoDB Provider (`gpamongo`)
```go
import "github.com/lemmego/gpamongo"

provider, err := gpamongo.NewProvider(gpa.Config{
    Host:     "localhost",
    Port:     27017,
    Database: "myapp",
    Username: "user",
    Password: "password",
})

userRepo := gpamongo.GetRepository[User](provider)
```

#### Redis Provider (`gparedis`)
```go
import "github.com/lemmego/gparedis"

provider, err := gparedis.NewProvider(gpa.Config{
    Host:     "localhost",
    Port:     6379,
    Password: "password",
    Database: 0,
})

sessionRepo := gparedis.GetRepository[Session](provider)
```

## 📚 Repository Operations

### Basic CRUD Operations

```go
ctx := context.Background()

// Create
user := &User{Name: "John", Email: "john@example.com"}
err := userRepo.Create(ctx, user)

// Create multiple
users := []*User{{Name: "Alice"}, {Name: "Bob"}}
err := userRepo.CreateBatch(ctx, users)

// Find by ID
user, err := userRepo.FindByID(ctx, 123)

// Find all
users, err := userRepo.FindAll(ctx)

// Update
user.Name = "John Updated"
err := userRepo.Update(ctx, user)

// Partial update
err := userRepo.UpdatePartial(ctx, 123, map[string]interface{}{
    "name": "John Partial",
})

// Delete
err := userRepo.Delete(ctx, user)

// Delete by ID
err := userRepo.DeleteByID(ctx, 123)
```

### Advanced Querying

```go
// Complex queries with type-safe conditions
users, err := userRepo.Query(ctx,
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("status", gpa.OpEqual, "active"),
    gpa.WhereLike("name", "John%"),
    gpa.WhereIn("role", []interface{}{"admin", "user"}),
    gpa.OrderBy("created_at", gpa.OrderDesc),
    gpa.Limit(50),
    gpa.Offset(0),
)

// Counting records
count, err := userRepo.Count(ctx, 
    gpa.Where("status", gpa.OpEqual, "active"),
)

// Check existence
exists, err := userRepo.Exists(ctx,
    gpa.Where("email", gpa.OpEqual, "john@example.com"),
)

// Aggregations (provider-specific)
if sqlRepo, ok := userRepo.(gpa.SQLRepository[User]); ok {
    // Raw SQL queries
    users, err := sqlRepo.FindBySQL(ctx,
        "SELECT * FROM users WHERE age > ? ORDER BY created_at DESC",
        []interface{}{18},
    )
    
    // Execute raw SQL
    result, err := sqlRepo.ExecSQL(ctx,
        "UPDATE users SET status = ? WHERE last_login < ?",
        "inactive", time.Now().AddDate(0, -6, 0),
    )
}
```

### Transactions

```go
err := userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    // Create user
    user := &User{Name: "John", Email: "john@example.com"}
    if err := tx.Create(ctx, user); err != nil {
        return err
    }
    
    // Update related data
    if err := tx.UpdatePartial(ctx, relatedID, map[string]interface{}{
        "user_id": user.ID,
    }); err != nil {
        return err
    }
    
    return nil // Commit
})
```

## 🔍 Registry Management

### Discovery and Health Checks

```go
// List all provider types
types := gpa.Registry().ListTypes()
fmt.Printf("Available types: %v\n", types) // [GORM Redis MongoDB]

// List instances of a specific type
instances, _ := gpa.Registry().ListInstances("GORM")
fmt.Printf("GORM instances: %v\n", instances) // [primary readonly default]

// Health check all providers
healthResults := gpa.Registry().HealthCheck()
for providerType, instances := range healthResults {
    for instanceName, err := range instances {
        if err != nil {
            fmt.Printf("❌ %s:%s is unhealthy: %v\n", providerType, instanceName, err)
        } else {
            fmt.Printf("✅ %s:%s is healthy\n", providerType, instanceName)
        }
    }
}

// Remove providers
err := gpa.Registry().Remove("GORM", "readonly")
err := gpa.Registry().RemoveAll() // Remove all providers
```

## ⚙️ Configuration

```go
config := gpa.Config{
    Driver:          "postgres",
    Host:            "localhost",
    Port:            5432,
    Database:        "myapp",
    Username:        "user",
    Password:        "password",
    ConnectionURL:   "", // Alternative to individual fields
    MaxOpenConns:    25,
    MaxIdleConns:    10,
    ConnMaxLifetime: time.Hour,
    ConnMaxIdleTime: time.Minute * 30,
    SSL: gpa.SSLConfig{
        Enabled:  true,
        Mode:     "require",
        CertFile: "/path/to/cert.pem",
        KeyFile:  "/path/to/key.pem",
        CAFile:   "/path/to/ca.pem",
    },
    Options: map[string]interface{}{
        "gorm": map[string]interface{}{
            "log_level":      "info",
            "singular_table": false,
        },
    },
}
```

## 🎯 Type Safety Benefits

Traditional approach with type assertions:
```go
// ❌ Runtime type assertions and potential panics
provider, err := registry.Get("GORM", "primary")
gormProvider := provider.(*gpagorm.Provider) // Can fail at runtime!

result, err := repo.FindByID(ctx, 123)
user := result.(*User) // Another type assertion!
```

GPA approach with compile-time safety:
```go
// ✅ Compile-time type safety, no type assertions
gormProvider, err := gpa.Get[*gpagorm.Provider]("primary")
user, err := userRepo.FindByID(ctx, 123) // Returns *User directly!
```

## 🚀 Performance

- **Zero reflection overhead** in repository operations
- **Compile-time type checking** eliminates runtime type assertion costs
- **Efficient connection pooling** with health checks
- **Lazy provider initialization** for better startup times
- **Concurrent-safe registry** with optimized read-write locks

## 📖 Documentation

- [API Reference](https://pkg.go.dev/github.com/lemmego/gpa)
- [Provider Documentation](./docs/providers.md)
- [Query Builder Guide](./docs/query-builder.md)
- [Migration Guide](./docs/migration.md)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

## 🙏 Acknowledgments

- Inspired by the repository pattern and clean architecture principles
- Built with Go generics for maximum type safety
- Supports multiple excellent Go database libraries (GORM, Bun, MongoDB driver, Redis)

## 📊 Project Status

- ✅ Core API stable
- ✅ Provider registry with type inference
- ✅ GORM provider complete
- ✅ Redis provider complete
- 🚧 Bun provider (in development)
- 🚧 MongoDB provider (in development)
- 📋 Additional providers planned

---

**Made with ❤️ by [Tanmay Das](https://github.com/tanmaymishu)**