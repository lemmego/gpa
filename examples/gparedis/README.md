# Redis Provider Example

This example demonstrates the Redis provider for the Go Persistence API (GPA), showcasing key-value operations, caching patterns, and Redis-specific features.

## Features Demonstrated

### Core Key-Value Operations
- **Basic CRUD**: Type-safe create, read, update, delete operations
- **TTL Management**: Setting expiration times and managing key lifecycles
- **Batch Operations**: Efficient multi-key get/set operations
- **Key Pattern Matching**: Finding keys by patterns
- **Atomic Operations**: Increment/decrement counters

### Redis-Specific Features
- **Hash Operations**: Field-based storage for complex objects
- **List Operations**: Queue and stack operations with FIFO/LIFO
- **Set Operations**: Unique collections with membership testing
- **Sorted Sets**: Ranked data structures for leaderboards
- **Pub/Sub**: Message publishing and subscription (basic patterns)

### Advanced Patterns
- **Caching Strategies**: TTL-based cache with automatic expiration
- **Session Management**: User session storage with automatic cleanup
- **Rate Limiting**: API rate limiting with sliding windows
- **Distributed Locking**: Simple distributed locks for coordination
- **Notification Queues**: Event-driven notification systems

## Prerequisites

- Go 1.18+
- Redis 6.0+ (running locally or remote)

### Redis Setup

#### Local Installation
```bash
# macOS with Homebrew
brew install redis

# Start Redis service
brew services start redis

# Ubuntu/Debian
sudo apt-get install redis-server

# Start Redis service
sudo systemctl start redis-server
```

#### Docker
```bash
# Run Redis in Docker
docker run -d -p 6379:6379 --name redis redis:latest

# Or with persistent storage
docker run -d -p 6379:6379 -v redis_data:/data --name redis redis:latest

# Run with configuration
docker run -d -p 6379:6379 -v ./redis.conf:/usr/local/etc/redis/redis.conf --name redis redis:latest redis-server /usr/local/etc/redis/redis.conf
```

## Quick Start

```bash
# Navigate to this directory
cd examples/gparedis

# Install dependencies
go mod tidy

# Run the example (Redis must be running)
go run main.go

# Or with custom Redis URL
REDIS_URL="redis://localhost:6379" go run main.go
```

## Database Configuration

The example uses local Redis by default:

```go
config := gpa.Config{
    Driver:        "redis",
    ConnectionURL: "redis://localhost:6379",
    Database:      "0", // Redis database number
    Options: map[string]interface{}{
        "redis": map[string]interface{}{
            "pool_size":      10,
            "min_idle_conns": 2,
            "max_retries":    3,
            "dial_timeout":   "5s",
            "read_timeout":   "3s",
            "write_timeout":  "3s",
            "pool_timeout":   "4s",
        },
    },
}
```

### Connection String Examples

#### Local Redis
```go
config := gpa.Config{
    Driver:        "redis",
    ConnectionURL: "redis://localhost:6379",
    Database:      "0",
}
```

#### Redis with Authentication
```go
config := gpa.Config{
    Driver:        "redis",
    ConnectionURL: "redis://username:password@localhost:6379",
    Database:      "0",
}
```

#### Redis Cluster
```go
config := gpa.Config{
    Driver:        "redis",
    ConnectionURL: "redis://node1:6379,node2:6379,node3:6379",
    Database:      "0",
}
```

#### Redis with SSL
```go
config := gpa.Config{
    Driver:        "redis",
    ConnectionURL: "rediss://username:password@secure-redis.com:6380",
    Database:      "0",
}
```

## Code Structure

### Entity Definitions
```go
type User struct {
    ID       string    `json:"id"`
    Name     string    `json:"name"`
    Email    string    `json:"email"`
    Metadata Metadata  `json:"metadata"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Metadata struct {
    LoginCount    int                    `json:"login_count"`
    LastLoginIP   string                 `json:"last_login_ip"`
    Preferences   map[string]interface{} `json:"preferences"`
}
```

### Repository Operations
```go
// Type-safe provider creation
provider, err := gparedis.NewTypeSafeProvider[User](config)
repo := provider.Repository()

// Key-value operations
err := repo.Create(ctx, &user)
user, err := repo.FindByID(ctx, "user:123")
err := repo.Update(ctx, &user)
err := repo.Delete(ctx, "user:123")
```

### Key-Value Specific Operations
```go
if kvRepo, ok := repo.(gpa.KeyValueRepository[User]); ok {
    // Set with TTL
    err := kvRepo.SetWithTTL(ctx, "user:temp", &user, time.Hour)
    
    // Get multiple keys
    users, err := kvRepo.GetMultiple(ctx, []string{"user:1", "user:2"})
    
    // Set multiple keys
    userMap := map[string]*User{"user:1": &user1, "user:2": &user2}
    err := kvRepo.SetMultiple(ctx, userMap)
    
    // Check existence
    exists, err := kvRepo.Exists(ctx, "user:123")
    
    // Get TTL
    ttl, err := kvRepo.GetTTL(ctx, "user:123")
}
```

### Redis Operations
```go
if redisRepo, ok := repo.(gpa.RedisRepository[User]); ok {
    // Hash operations
    err := redisRepo.HMSet(ctx, "user:1:profile", map[string]interface{}{
        "bio": "Software engineer",
        "location": "San Francisco",
    })
    
    bio, err := redisRepo.HGet(ctx, "user:1:profile", "bio")
    allFields, err := redisRepo.HGetAll(ctx, "user:1:profile")
    
    // Counter operations
    count, err := redisRepo.Incr(ctx, "user:1:login_count")
    views, err := redisRepo.IncrBy(ctx, "user:1:page_views", 5)
    
    // List operations
    err := redisRepo.LPush(ctx, "user:1:actions", "login")
    length, err := redisRepo.LLen(ctx, "user:1:actions")
    actions, err := redisRepo.LRange(ctx, "user:1:actions", 0, 10)
    
    // Set operations
    err := redisRepo.SAdd(ctx, "user:1:skills", "Go")
    isMember, err := redisRepo.SIsMember(ctx, "user:1:skills", "Go")
    skills, err := redisRepo.SMembers(ctx, "user:1:skills")
    
    // Sorted set operations (leaderboards)
    err := redisRepo.ZAdd(ctx, "leaderboard", 95.5, "user:1")
    score, err := redisRepo.ZScore(ctx, "leaderboard", "user:1")
    topUsers, err := redisRepo.ZRevRange(ctx, "leaderboard", 0, 9)
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
Structured errors with Redis-specific information:
```go
if gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
    // Handle key not found
}
```

### TTL Management
Automatic expiration for cache-like behavior:
```go
// Set with automatic expiration
err := kvRepo.SetWithTTL(ctx, "session:abc123", &session, 30*time.Minute)

// Check remaining time
ttl, err := kvRepo.GetTTL(ctx, "session:abc123")
```

### Atomic Operations
Thread-safe operations for counters and flags:
```go
// Atomic increment
count, err := redisRepo.Incr(ctx, "api_calls")

// Conditional set (only if key doesn't exist)
success, err := redisRepo.SetNX(ctx, "lock:user:123", "process_id", time.Minute)
```

## Use Cases and Patterns

### 1. Caching
```go
type CacheEntry struct {
    Key       string      `json:"key"`
    Value     interface{} `json:"value"`
    TTL       int         `json:"ttl"`
    CreatedAt time.Time   `json:"created_at"`
}

// Cache expensive computation
err := kvRepo.SetWithTTL(ctx, "computation:result", &entry, time.Hour)

// Check cache first
result, err := kvRepo.Get(ctx, "computation:result")
if gpa.IsErrorType(err, gpa.ErrorTypeNotFound) {
    // Cache miss - compute and store
    result = computeExpensiveResult()
    kvRepo.SetWithTTL(ctx, "computation:result", result, time.Hour)
}
```

### 2. Session Management
```go
type SessionData struct {
    UserID    string                 `json:"user_id"`
    Token     string                 `json:"token"`
    ExpiresAt time.Time              `json:"expires_at"`
    Data      map[string]interface{} `json:"data"`
}

// Create session
sessionKey := fmt.Sprintf("session:%s", token)
sessionTTL := time.Until(session.ExpiresAt)
err := kvRepo.SetWithTTL(ctx, sessionKey, &session, sessionTTL)

// Validate session
session, err := kvRepo.Get(ctx, sessionKey)
if err == nil && time.Now().Before(session.ExpiresAt) {
    // Valid session
}
```

### 3. Rate Limiting
```go
// Simple rate limiting (10 requests per minute)
rateLimitKey := fmt.Sprintf("rate_limit:user:%s", userID)
currentCount, err := redisRepo.Incr(ctx, rateLimitKey)
if currentCount == 1 {
    // First request in this window
    redisRepo.Expire(ctx, rateLimitKey, time.Minute)
}

if currentCount > 10 {
    return errors.New("rate limit exceeded")
}
```

### 4. Distributed Locking
```go
// Acquire lock
lockKey := fmt.Sprintf("lock:resource:%s", resourceID)
acquired, err := redisRepo.SetNX(ctx, lockKey, processID, 30*time.Second)
if !acquired {
    return errors.New("resource is locked")
}

defer func() {
    // Release lock
    redisRepo.Del(ctx, lockKey)
}()

// Critical section
performCriticalOperation()
```

### 5. Leaderboards
```go
// Update score
err := redisRepo.ZAdd(ctx, "game:leaderboard", newScore, playerID)

// Get player rank
rank, err := redisRepo.ZRevRank(ctx, "game:leaderboard", playerID)

// Get top 10 players
topPlayers, err := redisRepo.ZRevRange(ctx, "game:leaderboard", 0, 9)

// Get players around current player
around, err := redisRepo.ZRevRange(ctx, "game:leaderboard", rank-5, rank+5)
```

### 6. Real-time Notifications
```go
// Add notification to user's queue
notificationKey := fmt.Sprintf("notifications:%s", userID)
err := redisRepo.LPush(ctx, notificationKey, notification)

// Keep only last 50 notifications
err = redisRepo.LTrim(ctx, notificationKey, 0, 49)

// Get unread notifications
notifications, err := redisRepo.LRange(ctx, notificationKey, 0, -1)
```

## Performance Tips

1. **Use Pipelining**: Batch multiple operations for better performance
2. **Expire Unused Keys**: Always set TTL for temporary data
3. **Choose Right Data Structure**: Use appropriate Redis data types
4. **Connection Pooling**: Configure pool size based on workload
5. **Memory Management**: Monitor memory usage and configure eviction policies

## Testing

The example includes comprehensive scenarios:

- Connection failures and retries
- Key expiration and TTL management
- Data type validation
- Memory optimization
- Error handling patterns

## Common Patterns

### Repository Pattern
```go
type UserService struct {
    repo gpa.Repository[User]
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    return s.repo.FindByID(ctx, id)
}

func (s *UserService) CacheUser(ctx context.Context, user *User, ttl time.Duration) error {
    if kvRepo, ok := s.repo.(gpa.KeyValueRepository[User]); ok {
        return kvRepo.SetWithTTL(ctx, user.ID, user, ttl)
    }
    return s.repo.Create(ctx, user)
}
```

### Cache-Aside Pattern
```go
func (s *UserService) GetUserWithCache(ctx context.Context, id string) (*User, error) {
    // Try cache first
    if kvRepo, ok := s.repo.(gpa.KeyValueRepository[User]); ok {
        user, err := kvRepo.Get(ctx, id)
        if err == nil {
            return user, nil
        }
    }
    
    // Cache miss - get from primary storage
    user, err := s.primaryDB.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Update cache
    if kvRepo, ok := s.repo.(gpa.KeyValueRepository[User]); ok {
        kvRepo.SetWithTTL(ctx, id, user, time.Hour)
    }
    
    return user, nil
}
```

## Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure Redis server is running
2. **Authentication Failed**: Check username/password in connection string
3. **Memory Issues**: Configure max memory and eviction policies
4. **Performance**: Tune connection pool settings

### Debug Tips

1. Use Redis CLI: `redis-cli monitor` to watch commands
2. Check memory usage: `redis-cli info memory`
3. Analyze slow queries: `redis-cli --latency-history`
4. Monitor key patterns: `redis-cli --scan --pattern "user:*"`

## Environment Variables

```bash
# Redis connection
export REDIS_URL="redis://localhost:6379"

# With authentication
export REDIS_URL="redis://username:password@host:port"

# Redis Cluster
export REDIS_URL="redis://node1:6379,node2:6379,node3:6379"

# SSL connection
export REDIS_URL="rediss://username:password@secure-redis.com:6380"
```

## Known Issues

- Advanced Redis patterns (hash operations, list operations, etc.) require specialized Redis interfaces that are not yet implemented in the core GPA framework
- Some Redis-specific features like pub/sub, distributed locking, and complex data structures are commented out in the example
- The example currently focuses on basic key-value operations with TTL support
- Interface assertions for specific repository types (BatchKeyValueRepository, TTLKeyValueRepository, etc.) may show compiler warnings but are functionally correct

## Next Steps

- Explore other providers: GORM, MongoDB, Bun
- Review the main documentation for architecture details
- Check the test files for additional usage patterns
- Consider Redis Cluster for high availability scenarios