# MongoDB Provider Example

This example demonstrates the MongoDB provider for the Go Persistence API (GPA), showcasing document database operations with type safety and MongoDB-specific features.

## Features Demonstrated

### Core Document Operations
- **Document CRUD**: Type-safe create, read, update, delete operations
- **Nested Documents**: Complex document structures with embedded objects
- **Array Operations**: Working with arrays and embedded arrays
- **ObjectID Handling**: MongoDB ObjectID generation and manipulation
- **Aggregation Pipelines**: Complex data processing and analytics
- **Text Search**: Full-text search with text indexes

### MongoDB-Specific Features
- **Geospatial Queries**: Location-based searches with 2dsphere indexes
- **Index Management**: Creating compound, text, and geospatial indexes
- **Document Updates**: Using MongoDB update operators ($set, $inc, $addToSet)
- **Aggregation Framework**: Complex pipelines for data analysis
- **Query Operators**: MongoDB-native query syntax and operators
- **Distinct Operations**: Finding unique values across collections

### Advanced Patterns
- **Complex Aggregations**: Multi-stage pipelines with grouping and sorting
- **Relationship Modeling**: Document references and embedded documents
- **Performance Optimization**: Efficient queries and indexing strategies
- **Schema Flexibility**: Dynamic document structures

## Prerequisites

- Go 1.18+
- MongoDB 4.4+ (running locally or remote)

### MongoDB Setup

#### Local Installation
```bash
# macOS with Homebrew
brew install mongodb-community

# Start MongoDB service
brew services start mongodb-community

# Ubuntu/Debian
sudo apt-get install mongodb

# Start MongoDB service
sudo systemctl start mongod
```

#### Docker
```bash
# Run MongoDB in Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Or with persistent storage
docker run -d -p 27017:27017 -v mongodb_data:/data/db --name mongodb mongo:latest
```

## Quick Start

```bash
# Navigate to this directory
cd examples/gpamongo

# Install dependencies
go mod tidy

# Run the example (MongoDB must be running)
go run main.go

# Or with custom MongoDB URL
MONGODB_URL="mongodb://username:password@localhost:27017" go run main.go
```

## Database Configuration

The example uses local MongoDB by default:

```go
config := gpa.Config{
    Driver:        "mongodb",
    ConnectionURL: "mongodb://localhost:27017",
    Database:      "gpa_mongo_example",
    Options: map[string]interface{}{
        "mongo": map[string]interface{}{
            "max_pool_size": uint64(50),
            "min_pool_size": uint64(5),
        },
    },
}
```

### Connection String Examples

#### Local MongoDB
```go
config := gpa.Config{
    Driver:        "mongodb",
    ConnectionURL: "mongodb://localhost:27017",
    Database:      "myapp",
}
```

#### MongoDB Atlas
```go
config := gpa.Config{
    Driver:        "mongodb",
    ConnectionURL: "mongodb+srv://username:password@cluster.mongodb.net",
    Database:      "myapp",
}
```

#### MongoDB with Authentication
```go
config := gpa.Config{
    Driver:        "mongodb",
    ConnectionURL: "mongodb://username:password@localhost:27017",
    Database:      "myapp",
}
```

## Code Structure

### Document Definitions
```go
type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name      string             `bson:"name" json:"name"`
    Email     string             `bson:"email" json:"email"`
    Profile   *UserProfile       `bson:"profile,omitempty" json:"profile,omitempty"`
    Location  *GeoLocation       `bson:"location,omitempty" json:"location,omitempty"`
}

type GeoLocation struct {
    Type        string    `bson:"type" json:"type"`
    Coordinates []float64 `bson:"coordinates" json:"coordinates"` // [longitude, latitude]
    City        string    `bson:"city" json:"city"`
    Country     string    `bson:"country" json:"country"`
}
```

### Repository Operations
```go
// Type-safe provider creation
provider, err := gpamongo.NewTypeSafeProvider[User](config)
repo := provider.Repository()

// Document operations
err := repo.Create(ctx, &user)
user, err := repo.FindByID(ctx, userID)
err := repo.Update(ctx, &user)
err := repo.Delete(ctx, userID)
```

### Document-Specific Operations
```go
if docRepo, ok := repo.(gpa.DocumentRepository[User]); ok {
    // MongoDB query syntax
    users, err := docRepo.FindByDocument(ctx, map[string]interface{}{
        "is_active": true,
        "age": map[string]interface{}{"$gte": 25},
    })
    
    // Text search
    results, err := docRepo.TextSearch(ctx, "golang developer", gpa.Limit(10))
    
    // Geospatial queries
    nearby, err := docRepo.FindNear(ctx, "location", 
        []float64{-122.4194, 37.7749}, 100000) // 100km from SF
}
```

### Aggregation Pipelines
```go
pipeline := []map[string]interface{}{
    {
        "$match": map[string]interface{}{
            "is_active": true,
        },
    },
    {
        "$group": map[string]interface{}{
            "_id": "$location.country",
            "count": map[string]interface{}{"$sum": 1},
            "avgAge": map[string]interface{}{"$avg": "$age"},
        },
    },
    {
        "$sort": map[string]interface{}{
            "count": -1,
        },
    },
}

results, err := docRepo.Aggregate(ctx, pipeline)
```

### Update Operations
```go
// Using MongoDB update operators
updateResult, err := docRepo.UpdateDocument(ctx, userID, map[string]interface{}{
    "$set": map[string]interface{}{
        "age": 31,
        "updated_at": time.Now(),
    },
    "$addToSet": map[string]interface{}{
        "tags": "senior-developer",
    },
})

// Update multiple documents
manyResult, err := docRepo.UpdateManyDocuments(ctx,
    map[string]interface{}{"location.country": "USA"},
    map[string]interface{}{
        "$inc": map[string]interface{}{"age": 1},
    },
)
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
Structured errors with MongoDB-specific information:
```go
if gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
    // Handle document not found
}
```

### Transactions
MongoDB transactions (requires replica set):
```go
err := repo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    if err := tx.Create(ctx, &user1); err != nil {
        return err // Automatic rollback
    }
    return tx.Update(ctx, &user2)
})
```

### Index Management
```go
if docRepo, ok := repo.(gpa.DocumentRepository[User]); ok {
    // Text index
    textIndex := map[string]interface{}{
        "name": "text",
        "profile.bio": "text",
    }
    err := docRepo.CreateIndex(ctx, textIndex, false)
    
    // Geospatial index
    geoIndex := map[string]interface{}{
        "location": "2dsphere",
    }
    err := docRepo.CreateIndex(ctx, geoIndex, false)
    
    // Compound index
    compoundIndex := map[string]interface{}{
        "is_active": 1,
        "age": -1,
        "location.country": 1,
    }
    err := docRepo.CreateIndex(ctx, compoundIndex, false)
}
```

## Performance Tips

1. **Index Strategically**: Create indexes on frequently queried fields
2. **Use Aggregation**: Leverage MongoDB's aggregation framework for complex queries
3. **Limit Results**: Use `Limit()` for pagination and performance
4. **Project Fields**: Select only needed fields in aggregation pipelines
5. **Batch Operations**: Use `CreateBatch()` for multiple inserts

## MongoDB-Specific Features

### Text Search
```go
// Create text index first
textIndex := map[string]interface{}{
    "title": "text",
    "content": "text",
}
err := docRepo.CreateIndex(ctx, textIndex, false)

// Perform text search
results, err := docRepo.TextSearch(ctx, "golang mongodb", gpa.Limit(10))
```

### Geospatial Queries
```go
// Create geospatial index
geoIndex := map[string]interface{}{
    "location": "2dsphere",
}
err := docRepo.CreateIndex(ctx, geoIndex, false)

// Find documents near a point
nearby, err := docRepo.FindNear(ctx, "location", 
    []float64{longitude, latitude}, maxDistanceMeters)
```

### Aggregation Analytics
```go
// Complex analytics pipeline
pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{"published": true}},
    {"$group": map[string]interface{}{
        "_id": "$category",
        "count": map[string]interface{}{"$sum": 1},
        "avgViews": map[string]interface{}{"$avg": "$views"},
    }},
    {"$sort": map[string]interface{}{"count": -1}},
}

analytics, err := docRepo.Aggregate(ctx, pipeline)
```

## Testing

The example includes comprehensive error handling:

- Connection failures
- Index creation conflicts
- Document validation errors
- Transaction rollbacks
- Aggregation errors

## Common Patterns

### Repository Pattern
```go
type UserService struct {
    repo gpa.Repository[User]
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    user.CreatedAt = time.Now()
    user.UpdatedAt = time.Now()
    return s.repo.Create(ctx, user)
}
```

### Document Validation
```go
func (s *UserService) ValidateUser(user *User) error {
    if user.Name == "" {
        return gpa.NewError(gpa.ErrorTypeValidation, "name is required")
    }
    if !strings.Contains(user.Email, "@") {
        return gpa.NewError(gpa.ErrorTypeValidation, "invalid email")
    }
    return nil
}
```

## Troubleshooting

### Common Issues

1. **Connection Errors**: Ensure MongoDB is running and accessible
2. **Index Conflicts**: Handle duplicate index creation gracefully
3. **Transaction Errors**: Requires MongoDB 4.0+ with replica set
4. **Memory Issues**: Use pagination for large result sets

### Debug Tips

1. Enable MongoDB logging for query analysis
2. Use MongoDB Compass for visual query debugging
3. Monitor index usage with `db.collection.getIndexes()`
4. Profile slow queries with MongoDB profiler

## Environment Variables

```bash
# MongoDB connection
export MONGODB_URL="mongodb://localhost:27017"

# Or with authentication
export MONGODB_URL="mongodb://username:password@host:port"

# MongoDB Atlas
export MONGODB_URL="mongodb+srv://username:password@cluster.mongodb.net"
```

## Next Steps

- Explore other providers: GORM, Redis, Bun
- Review the main documentation for architecture details
- Check the test files for additional usage patterns
- Consider implementing custom aggregation pipelines for your use case