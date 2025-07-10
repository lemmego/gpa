package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gparedis"
)

// User represents a user entity for Redis storage
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	IsActive  bool      `json:"is_active"`
	Tags      []string  `json:"tags,omitempty"`
	Metadata  Metadata  `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Metadata represents additional user metadata
type Metadata struct {
	LoginCount    int                    `json:"login_count"`
	LastLoginIP   string                 `json:"last_login_ip"`
	Preferences   map[string]interface{} `json:"preferences"`
	SessionTokens []string               `json:"session_tokens,omitempty"`
}

// CacheEntry represents a cached data entry
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	TTL       int         `json:"ttl"`
	CreatedAt time.Time   `json:"created_at"`
}

// SessionData represents user session information
type SessionData struct {
	UserID    string                 `json:"user_id"`
	Token     string                 `json:"token"`
	ExpiresAt time.Time              `json:"expires_at"`
	Data      map[string]interface{} `json:"data"`
}

func main() {
	fmt.Println("🔴 Redis Provider Example")
	fmt.Println("Demonstrating Redis key-value operations and caching patterns")

	// Check if Redis is available
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Configure Redis connection
	// Note: You can also configure Redis using individual settings:
	config := gpa.Config{
		Driver:       "redis",
		Host:         "localhost",
		Port:         6379,
		Database:     "0", // Redis database number
		MaxOpenConns: 10,
		MaxIdleConns: 2,
		Options: map[string]interface{}{
			"redis": map[string]interface{}{
				"max_retries":   3,
				"read_timeout":  5 * time.Second,
				"write_timeout": 3 * time.Second,
			},
		},
	}

	// Create a single provider using the new unified API
	provider, err := gparedis.NewProvider(config)
	if err != nil {
		log.Fatalf("Failed to create provider (ensure Redis is running): %v", err)
	}
	defer provider.Close()

	// Create multiple repositories from the same provider using the new unified API
	userRepo := gparedis.GetRepository[User](provider)
	cacheRepo := gparedis.GetRepository[CacheEntry](provider)
	sessionRepo := gparedis.GetRepository[SessionData](provider)

	ctx := context.Background()

	// Check provider health
	err = provider.Health()
	if err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}
	fmt.Println("✓ Connected to Redis successfully")

	// ============================================
	// Basic Key-Value Operations
	// ============================================
	fmt.Println("\n=== Basic Key-Value Operations ===")

	// Create users
	users := []*User{
		{
			ID:       "user:1",
			Name:     "John Doe",
			Email:    "john@example.com",
			Age:      30,
			IsActive: true,
			Tags:     []string{"developer", "golang", "redis"},
			Metadata: Metadata{
				LoginCount:  15,
				LastLoginIP: "192.168.1.100",
				Preferences: map[string]interface{}{
					"theme":      "dark",
					"language":   "en",
					"newsletter": true,
				},
				SessionTokens: []string{"token123", "token456"},
			},
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "user:2",
			Name:     "Jane Smith",
			Email:    "jane@example.com",
			Age:      28,
			IsActive: true,
			Tags:     []string{"designer", "ui", "ux"},
			Metadata: Metadata{
				LoginCount:  8,
				LastLoginIP: "192.168.1.101",
				Preferences: map[string]interface{}{
					"theme":    "light",
					"language": "en",
					"notifications": map[string]bool{
						"email": true,
						"push":  false,
					},
				},
				SessionTokens: []string{"token789"},
			},
			CreatedAt: time.Now().Add(-48 * time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "user:3",
			Name:     "Bob Johnson",
			Email:    "bob@example.com",
			Age:      35,
			IsActive: false,
			Tags:     []string{"manager", "agile"},
			Metadata: Metadata{
				LoginCount:  45,
				LastLoginIP: "192.168.1.102",
				Preferences: map[string]interface{}{
					"theme":       "auto",
					"language":    "en",
					"dashboard":   "compact",
					"auto_logout": 3600,
				},
			},
			CreatedAt: time.Now().Add(-72 * time.Hour),
			UpdatedAt: time.Now().Add(-12 * time.Hour),
		},
	}

	// Store users in Redis
	for _, user := range users {
		err = userRepo.Create(ctx, user)
		if err != nil {
			log.Printf("Failed to create user %s: %v", user.ID, err)
		} else {
			fmt.Printf("✓ Created user: %s\n", user.Name)
		}
	}

	// Retrieve users
	retrievedUser, err := userRepo.FindByID(ctx, "user:1")
	if err != nil {
		log.Printf("Failed to find user: %v", err)
	} else {
		fmt.Printf("✓ Retrieved user: %s (age %d, %d logins)\n",
			retrievedUser.Name, retrievedUser.Age, retrievedUser.Metadata.LoginCount)
	}

	// ============================================
	// Redis-Specific Operations
	// ============================================
	fmt.Println("\n=== Redis-Specific Operations ===")

	if kvRepo, ok := userRepo.(gpa.TTLKeyValueRepository[User]); ok {
		// Set with TTL (expire in 1 hour)
		tempUser := &User{
			ID:        "user:temp",
			Name:      "Temporary User",
			Email:     "temp@example.com",
			Age:       25,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = kvRepo.SetWithTTL(ctx, tempUser.ID, tempUser, time.Hour)
		if err != nil {
			log.Printf("Failed to set user with TTL: %v", err)
		} else {
			fmt.Printf("✓ Created temporary user with 1-hour TTL\n")
		}

		// Get TTL
		ttl, err := kvRepo.GetTTL(ctx, "user:temp")
		if err != nil {
			log.Printf("Failed to get TTL: %v", err)
		} else {
			fmt.Printf("✓ Temporary user TTL: %v\n", ttl)
		}

		// Set expiration
		err = kvRepo.SetTTL(ctx, "user:1", 24*time.Hour)
		if err != nil {
			log.Printf("Failed to set expiration: %v", err)
		} else {
			fmt.Println("✓ Set user:1 to expire in 24 hours")
		}

		// Check if key exists
		exists, err := kvRepo.KeyExists(ctx, "user:1")
		if err != nil {
			log.Printf("Failed to check existence: %v", err)
		} else {
			fmt.Printf("✓ user:1 exists: %t\n", exists)
		}

		// Get multiple keys (requires batch interface)
		if batchRepo, ok := userRepo.(gpa.BatchKeyValueRepository[User]); ok {
			keys := []string{"user:1", "user:2", "user:3"}
			multiUsers, err := batchRepo.MGet(ctx, keys)
			if err != nil {
				log.Printf("Failed to get multiple users: %v", err)
			} else {
				fmt.Printf("✓ Retrieved %d users in batch operation\n", len(multiUsers))
			}

			// Set multiple keys
			userMap := make(map[string]*User)
			for _, user := range users {
				userMap[user.ID] = user
			}
			err = batchRepo.MSet(ctx, userMap)
			if err != nil {
				log.Printf("Failed to set multiple users: %v", err)
			} else {
				fmt.Printf("✓ Set %d users in batch operation\n", len(userMap))
			}
		} else {
			fmt.Println("✓ Batch operations not supported (would require BatchKeyValueRepository)")
		}
	}

	// ============================================
	// Redis Operations Interface
	// ============================================
	fmt.Println("\n=== Redis Operations Interface ===")

	// Note: Redis-specific operations would require a specialized interface
	// For now, we'll demonstrate basic key-value operations
	fmt.Println("✓ Redis-specific operations (HMSet, HGet, etc.) not implemented in this example")

	/*
		if redisRepo, ok := userRepo.(gpa.RedisRepository[User]); ok {
			// Hash operations
			hashKey := "user:1:profile"
			hashData := map[string]interface{}{
				"bio":      "Software engineer passionate about Redis",
				"website":  "https://johndoe.dev",
				"location": "San Francisco",
				"skills":   "Go,Redis,Microservices",
			}

			err = redisRepo.HMSet(ctx, hashKey, hashData)
			if err != nil {
				log.Printf("Failed to set hash: %v", err)
			} else {
				fmt.Printf("✓ Set hash data for user profile\n")
			}

			// Get hash field
			bio, err := redisRepo.HGet(ctx, hashKey, "bio")
			if err != nil {
				log.Printf("Failed to get hash field: %v", err)
			} else {
				fmt.Printf("✓ User bio: %s\n", bio)
			}

			// Get all hash fields
			allFields, err := redisRepo.HGetAll(ctx, hashKey)
			if err != nil {
				log.Printf("Failed to get all hash fields: %v", err)
			} else {
				fmt.Printf("✓ Retrieved %d profile fields\n", len(allFields))
			}

			// Increment counters
			loginCountKey := "user:1:login_count"
			newCount, err := redisRepo.Incr(ctx, loginCountKey)
			if err != nil {
				log.Printf("Failed to increment counter: %v", err)
			} else {
				fmt.Printf("✓ Incremented login count to: %d\n", newCount)
			}

			// Increment by value
			pageViewsKey := "user:1:page_views"
			newViews, err := redisRepo.IncrBy(ctx, pageViewsKey, 5)
			if err != nil {
				log.Printf("Failed to increment by value: %v", err)
			} else {
				fmt.Printf("✓ Incremented page views by 5 to: %d\n", newViews)
			}

			// List operations
			recentActionsKey := "user:1:recent_actions"
			actions := []string{
				"login",
				"view_profile",
				"update_settings",
				"logout",
			}

			for _, action := range actions {
				err = redisRepo.LPush(ctx, recentActionsKey, action)
				if err != nil {
					log.Printf("Failed to push to list: %v", err)
				}
			}
			fmt.Printf("✓ Added %d actions to recent actions list\n", len(actions))

			// Get list length
			listLen, err := redisRepo.LLen(ctx, recentActionsKey)
			if err != nil {
				log.Printf("Failed to get list length: %v", err)
			} else {
				fmt.Printf("✓ Recent actions list length: %d\n", listLen)
			}

			// Get list range
			recentActions, err := redisRepo.LRange(ctx, recentActionsKey, 0, 2)
			if err != nil {
				log.Printf("Failed to get list range: %v", err)
			} else {
				fmt.Printf("✓ Last 3 actions: %v\n", recentActions)
			}

			// Set operations
			skillsSetKey := "user:1:skills"
			skills := []string{"Go", "Redis", "Docker", "Kubernetes", "MongoDB"}
			for _, skill := range skills {
				err = redisRepo.SAdd(ctx, skillsSetKey, skill)
				if err != nil {
					log.Printf("Failed to add to set: %v", err)
				}
			}
			fmt.Printf("✓ Added %d skills to set\n", len(skills))

			// Check set membership
			hasGo, err := redisRepo.SIsMember(ctx, skillsSetKey, "Go")
			if err != nil {
				log.Printf("Failed to check set membership: %v", err)
			} else {
				fmt.Printf("✓ User has Go skill: %t\n", hasGo)
			}

			// Get all set members
			allSkills, err := redisRepo.SMembers(ctx, skillsSetKey)
			if err != nil {
				log.Printf("Failed to get set members: %v", err)
			} else {
				fmt.Printf("✓ User skills: %v\n", allSkills)
			}

			// Sorted set operations (leaderboard example)
			leaderboardKey := "leaderboard:developers"
			developers := map[string]float64{
				"user:1": 95.5,  // John Doe
				"user:2": 87.2,  // Jane Smith
				"user:3": 92.1,  // Bob Johnson
			}

			for userID, score := range developers {
				err = redisRepo.ZAdd(ctx, leaderboardKey, score, userID)
				if err != nil {
					log.Printf("Failed to add to sorted set: %v", err)
				}
			}
			fmt.Printf("✓ Added %d developers to leaderboard\n", len(developers))

			// Get top performers
			topPerformers, err := redisRepo.ZRevRange(ctx, leaderboardKey, 0, 2)
			if err != nil {
				log.Printf("Failed to get top performers: %v", err)
			} else {
				fmt.Printf("✓ Top 3 performers: %v\n", topPerformers)
			}

			// Get score
			johnScore, err := redisRepo.ZScore(ctx, leaderboardKey, "user:1")
			if err != nil {
				log.Printf("Failed to get score: %v", err)
			} else {
				fmt.Printf("✓ John's score: %.1f\n", johnScore)
			}
		}
	*/

	// ============================================
	// Caching Patterns
	// ============================================
	fmt.Println("\n=== Caching Patterns ===")

	// Cache expensive computations
	cacheEntries := []*CacheEntry{
		{
			Key:       "computation:fibonacci:50",
			Value:     12586269025,
			TTL:       3600, // 1 hour
			CreatedAt: time.Now(),
		},
		{
			Key: "api:weather:sf",
			Value: map[string]interface{}{
				"temperature": 22,
				"humidity":    65,
				"conditions":  "partly cloudy",
				"wind_speed":  15,
			},
			TTL:       300, // 5 minutes
			CreatedAt: time.Now(),
		},
		{
			Key: "query:user_stats",
			Value: map[string]interface{}{
				"total_users":  1250,
				"active_users": 987,
				"new_today":    23,
				"average_age":  32.5,
			},
			TTL:       1800, // 30 minutes
			CreatedAt: time.Now(),
		},
	}

	if kvCacheRepo, ok := cacheRepo.(gpa.TTLKeyValueRepository[CacheEntry]); ok {
		for _, entry := range cacheEntries {
			err = kvCacheRepo.SetWithTTL(ctx, entry.Key, entry, time.Duration(entry.TTL)*time.Second)
			if err != nil {
				log.Printf("Failed to cache entry %s: %v", entry.Key, err)
			} else {
				fmt.Printf("✓ Cached: %s (TTL: %ds)\n", entry.Key, entry.TTL)
			}
		}

		// Retrieve cached data
		weatherData, err := kvCacheRepo.Get(ctx, "api:weather:sf")
		if err != nil {
			log.Printf("Failed to get cached weather: %v", err)
		} else {
			weatherValue := weatherData.Value.(map[string]interface{})
			fmt.Printf("✓ Weather cache: %s, %v°C\n",
				weatherValue["conditions"], weatherValue["temperature"])
		}
	}

	// ============================================
	// Session Management
	// ============================================
	fmt.Println("\n=== Session Management ===")

	// Create user sessions
	sessions := []*SessionData{
		{
			UserID:    "user:1",
			Token:     "session_token_abc123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Data: map[string]interface{}{
				"login_time":    time.Now(),
				"ip_address":    "192.168.1.100",
				"user_agent":    "Mozilla/5.0 Chrome/96.0",
				"permissions":   []string{"read", "write", "admin"},
				"last_activity": time.Now(),
			},
		},
		{
			UserID:    "user:2",
			Token:     "session_token_def456",
			ExpiresAt: time.Now().Add(8 * time.Hour),
			Data: map[string]interface{}{
				"login_time":    time.Now().Add(-2 * time.Hour),
				"ip_address":    "192.168.1.101",
				"user_agent":    "Safari/15.0",
				"permissions":   []string{"read", "write"},
				"last_activity": time.Now().Add(-30 * time.Minute),
			},
		},
	}

	if kvSessionRepo, ok := sessionRepo.(gpa.TTLKeyValueRepository[SessionData]); ok {
		for _, session := range sessions {
			sessionKey := fmt.Sprintf("session:%s", session.Token)
			sessionTTL := time.Until(session.ExpiresAt)

			err = kvSessionRepo.SetWithTTL(ctx, sessionKey, session, sessionTTL)
			if err != nil {
				log.Printf("Failed to create session: %v", err)
			} else {
				fmt.Printf("✓ Created session for user %s (expires in %v)\n",
					session.UserID, sessionTTL.Round(time.Hour))
			}
		}

		// Validate session
		sessionKey := "session:session_token_abc123"
		session, err := kvSessionRepo.Get(ctx, sessionKey)
		if err != nil {
			log.Printf("Session validation failed: %v", err)
		} else {
			if time.Now().Before(session.ExpiresAt) {
				fmt.Printf("✓ Valid session for user %s\n", session.UserID)
			} else {
				fmt.Printf("✗ Session expired for user %s\n", session.UserID)
			}
		}

		// Session cleanup (remove expired sessions)
		// Note: Session cleanup would require pattern matching interface
		fmt.Println("✓ Session cleanup not implemented in this example (would require pattern matching)")

		/*
			if patternRepo, ok := sessionRepo.(gpa.PatternKeyValueRepository); ok {
				allSessionKeys, err := patternRepo.Keys(ctx, "session:*")
				if err != nil {
					log.Printf("Failed to get session keys: %v", err)
				} else {
					expiredCount := 0
					for _, key := range allSessionKeys {
						ttl, err := kvSessionRepo.GetTTL(ctx, key)
						if err == nil && ttl <= 0 {
							kvSessionRepo.DeleteKey(ctx, key)
							expiredCount++
						}
					}
					fmt.Printf("✓ Cleaned up %d expired sessions\n", expiredCount)
				}
			}
		*/
	}

	// ============================================
	// Advanced Redis Patterns
	// ============================================
	fmt.Println("\n=== Advanced Redis Patterns ===")

	// Note: Advanced Redis patterns would require Redis-specific interfaces
	fmt.Println("✓ Advanced Redis patterns (rate limiting, distributed locking, etc.) not implemented in this example")

	/*
		if redisRepo, ok := userRepo.(gpa.RedisRepository[User]); ok {
			// Rate limiting
			rateLimitKey := "rate_limit:user:1:api_calls"

			// Allow 10 API calls per minute
			currentCalls, err := redisRepo.Incr(ctx, rateLimitKey)
			if err != nil {
				log.Printf("Failed to increment rate limit: %v", err)
			} else {
				if currentCalls == 1 {
					// First call in this minute, set expiration
					redisRepo.Expire(ctx, rateLimitKey, time.Minute)
				}

				if currentCalls <= 10 {
					fmt.Printf("✓ API call allowed (%d/10 this minute)\n", currentCalls)
				} else {
					fmt.Printf("✗ Rate limit exceeded (%d/10 this minute)\n", currentCalls)
				}
			}

			// Distributed locking
			lockKey := "lock:user:1:update"
			lockAcquired, err := redisRepo.SetNX(ctx, lockKey, "process_123", 30*time.Second)
			if err != nil {
				log.Printf("Failed to acquire lock: %v", err)
			} else if lockAcquired {
				fmt.Println("✓ Acquired distributed lock for user update")

				// Simulate work
				time.Sleep(100 * time.Millisecond)

				// Release lock
				err = redisRepo.Del(ctx, lockKey)
				if err != nil {
					log.Printf("Failed to release lock: %v", err)
				} else {
					fmt.Println("✓ Released distributed lock")
				}
			} else {
				fmt.Println("✗ Failed to acquire lock (already held)")
			}

			// Pub/Sub example (simplified)
			notificationKey := "notifications:user:1"
			notifications := []string{
				"New message from Jane",
				"Your post received 5 likes",
				"Meeting reminder: 3 PM today",
			}

			for _, notification := range notifications {
				err = redisRepo.LPush(ctx, notificationKey, notification)
				if err != nil {
					log.Printf("Failed to add notification: %v", err)
				}
			}

			// Trim to keep only last 10 notifications
			err = redisRepo.LTrim(ctx, notificationKey, 0, 9)
			if err != nil {
				log.Printf("Failed to trim notifications: %v", err)
			} else {
				fmt.Printf("✓ Added %d notifications (keeping last 10)\n", len(notifications))
			}
		}
	*/

	// ============================================
	// Performance and Statistics
	// ============================================
	fmt.Println("\n=== Performance and Statistics ===")

	// Count operations
	if kvRepo, ok := userRepo.(gpa.PatternKeyValueRepository); ok {
		allUserKeys, err := kvRepo.Keys(ctx, "user:*")
		if err != nil {
			log.Printf("Failed to get user keys: %v", err)
		} else {
			fmt.Printf("✓ Total users in Redis: %d\n", len(allUserKeys))
		}

		// Get memory usage info (Redis-specific)
		fmt.Println("✓ Redis operations completed successfully")
	}

	// ============================================
	// Cleanup Operations
	// ============================================
	fmt.Println("\n=== Cleanup Operations ===")

	// Remove temporary data
	if kvRepo, ok := userRepo.(gpa.TTLKeyValueRepository[User]); ok {
		tempKeys := []string{
			"user:temp",
			"user:1:login_count",
			"user:1:page_views",
			"rate_limit:user:1:api_calls",
		}

		for _, key := range tempKeys {
			err = kvRepo.DeleteKey(ctx, key)
			if err != nil {
				log.Printf("Failed to delete key %s: %v", key, err)
			}
		}
		fmt.Printf("✓ Cleaned up %d temporary keys\n", len(tempKeys))
	}

	// ============================================
	// Provider Information
	// ============================================
	fmt.Println("\n=== Provider Information ===")

	providerInfo := provider.ProviderInfo()
	fmt.Printf("✓ Provider information:\n")
	fmt.Printf("  Name: %s\n", providerInfo.Name)
	fmt.Printf("  Version: %s\n", providerInfo.Version)
	fmt.Printf("  Database Type: %s\n", providerInfo.DatabaseType)
	fmt.Printf("  Features: %v\n", providerInfo.Features)

	fmt.Println("\n🎉 Redis provider example completed!")
}
