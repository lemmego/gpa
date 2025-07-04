// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gparedis"
)

// =====================================
// Demo Models for Redis
// =====================================

// User represents a user entity optimized for Redis storage
type User struct {
	ID          string              `json:"id"`
	Username    string              `json:"username"`
	Email       string              `json:"email"`
	Profile     UserProfile         `json:"profile"`
	Preferences map[string]string   `json:"preferences"`
	Tags        []string            `json:"tags"`
	LastLogin   time.Time           `json:"last_login"`
	CreatedAt   time.Time           `json:"created_at"`
}

// UserProfile represents nested user profile data
type UserProfile struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Age         int    `json:"age"`
	Country     string `json:"country"`
	Avatar      string `json:"avatar"`
}

// Session represents user session data
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Active    bool      `json:"active"`
}

// Product represents a product in an e-commerce scenario
type Product struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Price       float64           `json:"price"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	InStock     bool              `json:"in_stock"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Analytics represents analytics event data
type AnalyticsEvent struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

func main() {
	fmt.Println("🚀 Redis Adapter Demo for Go Persistence API (GPA)")
	fmt.Println("==================================================")

	// Register the Redis provider
	gpa.RegisterProvider("redis", &gparedis.Factory{})

	// Configuration for Redis
	config := gpa.Config{
		Driver:   "redis",
		Host:     "localhost",
		Port:     6379,
		Database: "0",
		Options: map[string]interface{}{
			"redis": map[string]interface{}{
				"dial_timeout":  time.Second * 5,
				"read_timeout":  time.Second * 3,
				"write_timeout": time.Second * 3,
			},
		},
	}

	// Create provider
	provider, err := gpa.NewProvider("redis", config)
	if err != nil {
		log.Fatalf("Failed to create Redis provider: %v", err)
	}
	defer provider.Close()

	fmt.Printf("✅ Connected to Redis at %s:%d (DB: %s)\n", config.Host, config.Port, config.Database)

	// Test provider health
	if err := provider.Health(); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}
	fmt.Println("✅ Redis health check passed")

	ctx := context.Background()

	// Demo different aspects of Redis with GPA
	demoBasicOperations(ctx, provider)
	demoKeyValueOperations(ctx, provider)
	demoRedisDataStructures(ctx, provider)
	demoCaching(ctx, provider)
	demoSessionManagement(ctx, provider)
	demoRealTimeAnalytics(ctx, provider)
	demoPubSub(ctx, provider)
	demoStreams(ctx, provider)

	fmt.Println("\n🎉 Redis Demo completed successfully!")
}

// =====================================
// Basic GPA Repository Operations
// =====================================

func demoBasicOperations(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n📝 Demo: Basic Repository Operations")
	fmt.Println("-----------------------------------")

	userRepo := provider.RepositoryFor(&User{})

	// Create users
	users := []*User{
		{
			ID:       "user1",
			Username: "alice",
			Email:    "alice@example.com",
			Profile: UserProfile{
				FirstName: "Alice",
				LastName:  "Johnson",
				Age:       28,
				Country:   "USA",
				Avatar:    "https://example.com/avatars/alice.jpg",
			},
			Preferences: map[string]string{
				"theme":       "dark",
				"language":    "en",
				"newsletter":  "enabled",
			},
			Tags:      []string{"premium", "early-adopter"},
			LastLogin: time.Now().Add(-time.Hour * 2),
			CreatedAt: time.Now().Add(-time.Hour * 24 * 30),
		},
		{
			ID:       "user2",
			Username: "bob",
			Email:    "bob@example.com",
			Profile: UserProfile{
				FirstName: "Bob",
				LastName:  "Smith",
				Age:       35,
				Country:   "Canada",
				Avatar:    "https://example.com/avatars/bob.jpg",
			},
			Preferences: map[string]string{
				"theme":      "light",
				"language":   "en",
				"newsletter": "disabled",
			},
			Tags:      []string{"standard"},
			LastLogin: time.Now().Add(-time.Minute * 30),
			CreatedAt: time.Now().Add(-time.Hour * 24 * 15),
		},
	}

	// Batch create
	err := userRepo.CreateBatch(ctx, users)
	if err != nil {
		log.Printf("Error creating users: %v", err)
		return
	}
	fmt.Printf("✅ Created %d users\n", len(users))

	// Find by ID
	var retrievedUser User
	err = userRepo.FindByID(ctx, "user1", &retrievedUser)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return
	}
	fmt.Printf("✅ Retrieved user: %s (%s)\n", retrievedUser.Username, retrievedUser.Email)

	// Update user
	retrievedUser.LastLogin = time.Now()
	retrievedUser.Preferences["last_activity"] = "redis_demo"
	err = userRepo.Update(ctx, &retrievedUser)
	if err != nil {
		log.Printf("Error updating user: %v", err)
		return
	}
	fmt.Println("✅ Updated user's last login")

	// Query with conditions
	var activeUsers []User
	err = userRepo.Query(ctx, &activeUsers,
		gpa.Where("profile.country", gpa.OpEqual, "USA"),
		gpa.Limit(10),
	)
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return
	}
	fmt.Printf("✅ Found %d users from USA\n", len(activeUsers))

	// Count users
	count, err := userRepo.Count(ctx)
	if err != nil {
		log.Printf("Error counting users: %v", err)
		return
	}
	fmt.Printf("✅ Total users in database: %d\n", count)
}

// =====================================
// Key-Value Operations
// =====================================

func demoKeyValueOperations(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n🔑 Demo: Key-Value Operations")
	fmt.Println("-----------------------------")

	repo := provider.RepositoryFor(&User{})
	kvRepo, ok := repo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Simple key-value operations
	config := map[string]interface{}{
		"app_name":    "Redis GPA Demo",
		"version":     "1.0.0",
		"environment": "development",
		"features": map[string]bool{
			"dark_mode":     true,
			"notifications": true,
			"analytics":     false,
		},
	}

	// Set configuration
	err := kvRepo.Set(ctx, "app:config", config, 0)
	if err != nil {
		log.Printf("Error setting config: %v", err)
		return
	}
	fmt.Println("✅ Stored application configuration")

	// Get configuration
	var retrievedConfig map[string]interface{}
	err = kvRepo.Get(ctx, "app:config", &retrievedConfig)
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return
	}
	fmt.Printf("✅ Retrieved config: %s v%s\n", 
		retrievedConfig["app_name"], retrievedConfig["version"])

	// Batch operations
	userStats := map[string]interface{}{
		"user:stats:user1": map[string]int{"login_count": 42, "posts": 15},
		"user:stats:user2": map[string]int{"login_count": 28, "posts": 8},
		"user:preferences:user1": map[string]string{"theme": "dark", "lang": "en"},
	}

	err = kvRepo.MSet(ctx, userStats, time.Hour*24)
	if err != nil {
		log.Printf("Error batch setting: %v", err)
		return
	}
	fmt.Println("✅ Batch stored user statistics with 24h TTL")

	// Increment counter
	dailyLogins, err := kvRepo.Increment(ctx, "stats:daily_logins", 1)
	if err != nil {
		log.Printf("Error incrementing: %v", err)
		return
	}
	fmt.Printf("✅ Daily logins count: %d\n", dailyLogins)

	// Pattern matching
	keys, err := kvRepo.Keys(ctx, "user:*")
	if err != nil {
		log.Printf("Error getting keys: %v", err)
		return
	}
	fmt.Printf("✅ Found %d user-related keys\n", len(keys))
}

// =====================================
// Redis Data Structures Demo
// =====================================

func demoRedisDataStructures(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n📊 Demo: Redis Data Structures")
	fmt.Println("-------------------------------")

	repo := provider.RepositoryFor(&User{})
	redisRepo, ok := repo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Lists - Recent activity feed
	fmt.Println("📝 Lists: Activity Feed")
	activities := []interface{}{
		"user1 logged in",
		"user2 created a post",
		"user1 liked a photo", 
		"user3 joined the platform",
	}

	for _, activity := range activities {
		err := redisRepo.LPush(ctx, "activity:feed", activity)
		if err != nil {
			log.Printf("Error adding activity: %v", err)
			continue
		}
	}

	var recentActivities []string
	err := redisRepo.LRange(ctx, "activity:feed", 0, 2, &recentActivities)
	if err != nil {
		log.Printf("Error getting activities: %v", err)
	} else {
		fmt.Printf("✅ Recent activities: %v\n", recentActivities)
	}

	// Sets - User tags/categories
	fmt.Println("🏷️  Sets: User Tags")
	premiumUsers := []interface{}{"user1", "user3", "user5"}
	err = redisRepo.SAdd(ctx, "users:premium", premiumUsers...)
	if err != nil {
		log.Printf("Error adding to set: %v", err)
	} else {
		fmt.Println("✅ Added premium users to set")
	}

	isPremium, err := redisRepo.SIsMember(ctx, "users:premium", "user1")
	if err != nil {
		log.Printf("Error checking membership: %v", err)
	} else {
		fmt.Printf("✅ Is user1 premium? %t\n", isPremium)
	}

	// Hashes - User profiles
	fmt.Println("🗃️  Hashes: User Profiles")
	err = redisRepo.HSet(ctx, "profile:user1",
		"name", "Alice Johnson",
		"email", "alice@example.com",
		"country", "USA",
		"premium", "true",
	)
	if err != nil {
		log.Printf("Error setting hash: %v", err)
	} else {
		fmt.Println("✅ Stored user profile in hash")
	}

	var profileData map[string]string
	err = redisRepo.HGetAll(ctx, "profile:user1", &profileData)
	if err != nil {
		log.Printf("Error getting hash: %v", err)
	} else {
		fmt.Printf("✅ Retrieved profile: %s from %s\n", 
			profileData["name"], profileData["country"])
	}

	// Sorted Sets - Leaderboard
	fmt.Println("🏆 Sorted Sets: User Scores")
	scores := []redis.Z{
		{Score: 1500, Member: "user1"},
		{Score: 1200, Member: "user2"},
		{Score: 1800, Member: "user3"},
		{Score: 900, Member: "user4"},
	}

	err = redisRepo.ZAdd(ctx, "leaderboard:scores", scores...)
	if err != nil {
		log.Printf("Error adding scores: %v", err)
	} else {
		fmt.Println("✅ Added user scores to leaderboard")
	}

	var topUsers []string
	err = redisRepo.ZRangeByScore(ctx, "leaderboard:scores", "1400", "+inf", &topUsers)
	if err != nil {
		log.Printf("Error getting top scores: %v", err)
	} else {
		fmt.Printf("✅ Top performers (>1400): %v\n", topUsers)
	}
}

// =====================================
// Caching Demo
// =====================================

func demoCaching(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n💾 Demo: Caching with TTL")
	fmt.Println("-------------------------")

	repo := provider.RepositoryFor(&Product{})
	kvRepo, ok := repo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Cache expensive computation results
	expensiveResult := map[string]interface{}{
		"computation_result": 42,
		"processing_time_ms": 1500,
		"algorithm_version":  "v2.1",
		"cached_at":          time.Now(),
	}

	// Cache for 30 seconds
	err := kvRepo.Set(ctx, "cache:expensive_computation", expensiveResult, time.Second*30)
	if err != nil {
		log.Printf("Error caching result: %v", err)
		return
	}
	fmt.Println("✅ Cached computation result with 30s TTL")

	// Check TTL
	ttl, err := kvRepo.TTL(ctx, "cache:expensive_computation")
	if err != nil {
		log.Printf("Error checking TTL: %v", err)
	} else {
		fmt.Printf("✅ Cache TTL: %v\n", ttl.Round(time.Second))
	}

	// Cache product details
	products := map[string]interface{}{
		"product:1": Product{
			ID:          "1",
			Name:        "Redis Book",
			Description: "Learn Redis with GPA",
			Price:       29.99,
			Category:    "books",
			Tags:        []string{"redis", "database", "nosql"},
			InStock:     true,
			CreatedAt:   time.Now(),
		},
		"product:2": Product{
			ID:          "2", 
			Name:        "GPA T-Shirt",
			Description: "Official GPA merchandise",
			Price:       19.99,
			Category:    "apparel",
			Tags:        []string{"gpa", "merchandise"},
			InStock:     true,
			CreatedAt:   time.Now(),
		},
	}

	// Cache products for 1 hour
	err = kvRepo.MSet(ctx, products, time.Hour)
	if err != nil {
		log.Printf("Error caching products: %v", err)
	} else {
		fmt.Printf("✅ Cached %d products with 1h TTL\n", len(products))
	}
}

// =====================================
// Session Management Demo
// =====================================

func demoSessionManagement(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n🔐 Demo: Session Management")
	fmt.Println("---------------------------")

	sessionRepo := provider.RepositoryFor(&Session{})
	kvRepo, ok := sessionRepo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Create user sessions
	sessions := []*Session{
		{
			ID:        "sess_" + generateID(),
			UserID:    "user1",
			Token:     "tok_" + generateID(),
			ExpiresAt: time.Now().Add(time.Hour * 2),
			IPAddress: "192.168.1.100",
			UserAgent: "Mozilla/5.0 (Chrome)",
			Active:    true,
		},
		{
			ID:        "sess_" + generateID(),
			UserID:    "user2", 
			Token:     "tok_" + generateID(),
			ExpiresAt: time.Now().Add(time.Hour * 4),
			IPAddress: "10.0.0.5",
			UserAgent: "Mozilla/5.0 (Firefox)",
			Active:    true,
		},
	}

	// Store sessions with auto-expiry
	for _, session := range sessions {
		ttl := time.Until(session.ExpiresAt)
		err := kvRepo.Set(ctx, "session:"+session.ID, session, ttl)
		if err != nil {
			log.Printf("Error storing session: %v", err)
			continue
		}

		// Also store user->session mapping
		err = kvRepo.Set(ctx, "user_session:"+session.UserID, session.ID, ttl)
		if err != nil {
			log.Printf("Error storing user session mapping: %v", err)
		}
	}
	fmt.Printf("✅ Created %d user sessions with auto-expiry\n", len(sessions))

	// Retrieve session by user
	var sessionID string
	err := kvRepo.Get(ctx, "user_session:user1", &sessionID)
	if err != nil {
		log.Printf("Error getting user session: %v", err)
	} else {
		var userSession Session
		err = kvRepo.Get(ctx, "session:"+sessionID, &userSession)
		if err != nil {
			log.Printf("Error getting session: %v", err)
		} else {
			fmt.Printf("✅ Retrieved session for user1: %s (expires in %v)\n", 
				userSession.ID, time.Until(userSession.ExpiresAt).Round(time.Minute))
		}
	}

	// Session activity tracking using sets
	err = kvRepo.SAdd(ctx, "active_sessions:"+time.Now().Format("2006-01-02"), sessions[0].ID, sessions[1].ID)
	if err != nil {
		log.Printf("Error tracking session activity: %v", err)
	} else {
		fmt.Println("✅ Tracked daily active sessions")
	}
}

// =====================================
// Real-time Analytics Demo
// =====================================

func demoRealTimeAnalytics(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n📈 Demo: Real-time Analytics")
	fmt.Println("----------------------------")

	repo := provider.RepositoryFor(&AnalyticsEvent{})
	kvRepo, ok := repo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Page view counters
	pages := []string{"home", "products", "about", "contact"}
	for _, page := range pages {
		views, err := kvRepo.Increment(ctx, "analytics:pageviews:"+page, int64(10+len(page)))
		if err != nil {
			log.Printf("Error incrementing page views: %v", err)
		} else {
			fmt.Printf("✅ Page '%s' views: %d\n", page, views)
		}
	}

	// User engagement metrics using sorted sets
	engagementScores := []redis.Z{
		{Score: 85.5, Member: "user1"},
		{Score: 92.3, Member: "user2"},
		{Score: 78.1, Member: "user3"},
		{Score: 95.7, Member: "user4"},
	}

	err := kvRepo.ZAdd(ctx, "analytics:engagement:weekly", engagementScores...)
	if err != nil {
		log.Printf("Error storing engagement scores: %v", err)
	} else {
		fmt.Println("✅ Stored weekly engagement scores")
	}

	// Get top engaged users
	var topEngaged []string
	err = kvRepo.ZRangeByScore(ctx, "analytics:engagement:weekly", "90", "+inf", &topEngaged)
	if err != nil {
		log.Printf("Error getting top engaged users: %v", err)
	} else {
		fmt.Printf("✅ Highly engaged users (>90%%): %v\n", topEngaged)
	}

	// Store real-time events
	events := []AnalyticsEvent{
		{
			ID:        generateID(),
			EventType: "click",
			UserID:    "user1",
			Data: map[string]interface{}{
				"element": "signup_button",
				"page":    "home",
				"x":       150,
				"y":       300,
			},
			Timestamp: time.Now(),
		},
		{
			ID:        generateID(),
			EventType: "purchase",
			UserID:    "user2",
			Data: map[string]interface{}{
				"product_id": "prod_123",
				"amount":     29.99,
				"currency":   "USD",
			},
			Timestamp: time.Now(),
		},
	}

	for _, event := range events {
		err := kvRepo.Set(ctx, "events:"+event.ID, event, time.Hour*24)
		if err != nil {
			log.Printf("Error storing event: %v", err)
		}
	}
	fmt.Printf("✅ Stored %d analytics events\n", len(events))
}

// =====================================
// Pub/Sub Demo
// =====================================

func demoPubSub(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n📡 Demo: Pub/Sub Messaging")
	fmt.Println("--------------------------")

	repo := provider.RepositoryFor(&User{})
	redisRepo, ok := repo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Subscribe to channels
	pubsub, err := redisRepo.Subscribe(ctx, "notifications", "alerts", "user_events")
	if err != nil {
		log.Printf("Error subscribing: %v", err)
		return
	}
	defer pubsub.Close()
	fmt.Println("✅ Subscribed to notification channels")

	// Publish messages
	messages := map[string]interface{}{
		"notifications": map[string]interface{}{
			"type":    "welcome",
			"user_id": "user1",
			"message": "Welcome to the platform!",
		},
		"alerts": map[string]interface{}{
			"type":     "system",
			"severity": "info",
			"message":  "System maintenance scheduled for tonight",
		},
		"user_events": map[string]interface{}{
			"event":   "login",
			"user_id": "user2",
			"time":    time.Now(),
		},
	}

	for channel, message := range messages {
		jsonMsg, _ := json.Marshal(message)
		err := redisRepo.Publish(ctx, channel, string(jsonMsg))
		if err != nil {
			log.Printf("Error publishing to %s: %v", channel, err)
		} else {
			fmt.Printf("✅ Published message to '%s'\n", channel)
		}
	}

	// Try to receive a message (with short timeout for demo)
	msgCtx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancel()

	msg, err := pubsub.ReceiveMessage(msgCtx)
	if err == nil {
		fmt.Printf("✅ Received message on '%s': %s\n", msg.Channel, msg.Payload[:50]+"...")
	} else {
		fmt.Println("ℹ️  No messages received (this is normal in a demo)")
	}
}

// =====================================
// Streams Demo
// =====================================

func demoStreams(ctx context.Context, provider gpa.Provider) {
	fmt.Println("\n🌊 Demo: Redis Streams")
	fmt.Println("----------------------")

	repo := provider.RepositoryFor(&User{})
	redisRepo, ok := repo.(*gparedis.Repository)
	if !ok {
		fmt.Println("❌ Repository is not a Redis repository")
		return
	}

	// Add events to stream
	streamEvents := []map[string]interface{}{
		{
			"event":   "user_signup",
			"user_id": "user1",
			"email":   "alice@example.com",
			"plan":    "premium",
		},
		{
			"event":      "page_view",
			"user_id":    "user1", 
			"page":       "/dashboard",
			"session_id": "sess_123",
		},
		{
			"event":    "purchase",
			"user_id":  "user2",
			"product":  "redis_book",
			"amount":   "29.99",
			"currency": "USD",
		},
	}

	var streamIDs []string
	for _, event := range streamEvents {
		id, err := redisRepo.XAdd(ctx, "user_activity", event)
		if err != nil {
			log.Printf("Error adding to stream: %v", err)
		} else {
			streamIDs = append(streamIDs, id)
			fmt.Printf("✅ Added event to stream: %s\n", id)
		}
	}

	// Read from stream
	streams := map[string]string{
		"user_activity": "0", // Read from beginning
	}

	results, err := redisRepo.XRead(ctx, streams, 10, 0)
	if err != nil {
		log.Printf("Error reading stream: %v", err)
	} else {
		fmt.Printf("✅ Read %d entries from stream\n", len(results[0].Messages))
		
		// Display first event
		if len(results) > 0 && len(results[0].Messages) > 0 {
			firstMsg := results[0].Messages[0]
			fmt.Printf("   First event: %s - User %s\n", 
				firstMsg.Values["event"], firstMsg.Values["user_id"])
		}
	}
}

// Helper function to generate simple IDs
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}