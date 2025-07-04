package gparedis

import (
	"context"
	"testing"
	"time"

	"github.com/lemmego/gpa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test models
type TestUser struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Age      int       `json:"age"`
	Status   string    `json:"status"`
	Created  time.Time `json:"created"`
}

type TestSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
}

type TestCounter struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// RedisAdapterTestSuite provides integration tests for Redis adapter
type RedisAdapterTestSuite struct {
	suite.Suite
	provider gpa.Provider
	userRepo gpa.Repository
	ctx      context.Context
}

func (suite *RedisAdapterTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	
	// Skip tests if Redis is not available
	config := gpa.Config{
		Driver:   "redis",
		Host:     "localhost",
		Port:     6379,
		Database: "15", // Use DB 15 for testing
		Options: map[string]interface{}{
			"redis": map[string]interface{}{
				"dial_timeout":  time.Second * 5,
				"read_timeout":  time.Second * 3,
				"write_timeout": time.Second * 3,
			},
		},
	}

	provider, err := gpa.NewProvider("redis", config)
	if err != nil {
		suite.T().Skip("Redis not available for testing:", err)
		return
	}

	suite.provider = provider
	suite.userRepo = provider.RepositoryFor(&TestUser{})
	
	// Clean up test data
	suite.cleanupTestData()
}

func (suite *RedisAdapterTestSuite) TearDownSuite() {
	if suite.provider != nil {
		suite.cleanupTestData()
		suite.provider.Close()
	}
}

func (suite *RedisAdapterTestSuite) SetupTest() {
	// Clean up before each test
	suite.cleanupTestData()
}

func (suite *RedisAdapterTestSuite) cleanupTestData() {
	if redisRepo, ok := suite.userRepo.(*Repository); ok {
		// Delete all test keys
		keys, err := redisRepo.Keys(suite.ctx, "*")
		if err == nil && len(keys) > 0 {
			redisRepo.MDelete(suite.ctx, keys)
		}
	}
}

// =====================================
// Basic Repository Tests
// =====================================

func (suite *RedisAdapterTestSuite) TestProviderFactory() {
	factory := &Factory{}
	
	supportedDrivers := factory.SupportedDrivers()
	assert.Contains(suite.T(), supportedDrivers, "redis")
	
	config := gpa.Config{
		Driver: "redis",
		Host:   "localhost",
		Port:   6379,
	}
	
	provider, err := factory.Create(config)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), provider)
	
	defer provider.Close()
	
	// Test provider info
	info := provider.ProviderInfo()
	assert.Equal(suite.T(), "Redis", info.Name)
	assert.Equal(suite.T(), gpa.DatabaseTypeKV, info.DatabaseType)
	
	features := provider.SupportedFeatures()
	assert.Contains(suite.T(), features, gpa.FeaturePubSub)
	assert.Contains(suite.T(), features, gpa.FeatureIndexing)
}

func (suite *RedisAdapterTestSuite) TestProviderHealth() {
	err := suite.provider.Health()
	assert.NoError(suite.T(), err)
}

func (suite *RedisAdapterTestSuite) TestCreate() {
	user := &TestUser{
		ID:      "user1",
		Name:    "John Doe",
		Email:   "john@example.com",
		Age:     30,
		Status:  "active",
		Created: time.Now(),
	}

	err := suite.userRepo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)
	
	// Verify it was created
	var retrieved TestUser
	err = suite.userRepo.FindByID(suite.ctx, "user1", &retrieved)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Name, retrieved.Name)
	assert.Equal(suite.T(), user.Email, retrieved.Email)
}

func (suite *RedisAdapterTestSuite) TestCreateBatch() {
	users := []*TestUser{
		{ID: "user2", Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{ID: "user3", Name: "Bob", Email: "bob@example.com", Age: 35, Status: "inactive"},
		{ID: "user4", Name: "Charlie", Email: "charlie@example.com", Age: 28, Status: "active"},
	}

	err := suite.userRepo.CreateBatch(suite.ctx, users)
	assert.NoError(suite.T(), err)
	
	// Verify all were created
	for _, user := range users {
		var retrieved TestUser
		err = suite.userRepo.FindByID(suite.ctx, user.ID, &retrieved)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.Name, retrieved.Name)
	}
}

func (suite *RedisAdapterTestSuite) TestFindByID() {
	// Test finding non-existent entity
	var user TestUser
	err := suite.userRepo.FindByID(suite.ctx, "nonexistent", &user)
	assert.Error(suite.T(), err)
	assert.IsType(suite.T(), gpa.GPAError{}, err)
	
	// Create and find existing entity
	testUser := &TestUser{ID: "user5", Name: "Test User", Email: "test@example.com"}
	err = suite.userRepo.Create(suite.ctx, testUser)
	require.NoError(suite.T(), err)
	
	err = suite.userRepo.FindByID(suite.ctx, "user5", &user)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testUser.Name, user.Name)
}

func (suite *RedisAdapterTestSuite) TestUpdate() {
	// Create initial user
	user := &TestUser{ID: "user6", Name: "Initial Name", Email: "initial@example.com"}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)
	
	// Update user
	user.Name = "Updated Name"
	user.Email = "updated@example.com"
	err = suite.userRepo.Update(suite.ctx, user)
	assert.NoError(suite.T(), err)
	
	// Verify update
	var retrieved TestUser
	err = suite.userRepo.FindByID(suite.ctx, "user6", &retrieved)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Name", retrieved.Name)
	assert.Equal(suite.T(), "updated@example.com", retrieved.Email)
}

func (suite *RedisAdapterTestSuite) TestUpdatePartial() {
	// Create initial user
	user := &TestUser{ID: "user7", Name: "Original Name", Email: "original@example.com", Age: 25}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)
	
	// Partial update
	updates := map[string]interface{}{
		"name": "Partially Updated",
		"age":  30,
	}
	err = suite.userRepo.UpdatePartial(suite.ctx, "user7", updates)
	assert.NoError(suite.T(), err)
	
	// Verify partial update
	var retrieved TestUser
	err = suite.userRepo.FindByID(suite.ctx, "user7", &retrieved)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Partially Updated", retrieved.Name)
	assert.Equal(suite.T(), 30, retrieved.Age)
	assert.Equal(suite.T(), "original@example.com", retrieved.Email) // Should remain unchanged
}

func (suite *RedisAdapterTestSuite) TestDelete() {
	// Create user
	user := &TestUser{ID: "user8", Name: "To Delete", Email: "delete@example.com"}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)
	
	// Delete user
	err = suite.userRepo.Delete(suite.ctx, "user8")
	assert.NoError(suite.T(), err)
	
	// Verify deletion
	var retrieved TestUser
	err = suite.userRepo.FindByID(suite.ctx, "user8", &retrieved)
	assert.Error(suite.T(), err)
	assert.IsType(suite.T(), gpa.GPAError{}, err)
}

func (suite *RedisAdapterTestSuite) TestFindAll() {
	// Create test data
	users := []*TestUser{
		{ID: "user9", Name: "User 9", Status: "active"},
		{ID: "user10", Name: "User 10", Status: "inactive"},
		{ID: "user11", Name: "User 11", Status: "active"},
	}
	
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)
	
	// Find all
	var allUsers []TestUser
	err = suite.userRepo.FindAll(suite.ctx, &allUsers)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), allUsers, 3)
}

func (suite *RedisAdapterTestSuite) TestQueryWithConditions() {
	// Create test data
	users := []*TestUser{
		{ID: "user12", Name: "Active User 1", Status: "active", Age: 25},
		{ID: "user13", Name: "Active User 2", Status: "active", Age: 35},
		{ID: "user14", Name: "Inactive User", Status: "inactive", Age: 30},
	}
	
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)
	
	// Query with conditions
	var activeUsers []TestUser
	err = suite.userRepo.Query(suite.ctx, &activeUsers,
		gpa.Where("status", gpa.OpEqual, "active"),
	)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), activeUsers, 2)
	
	// Verify results
	for _, user := range activeUsers {
		assert.Equal(suite.T(), "active", user.Status)
	}
}

func (suite *RedisAdapterTestSuite) TestQueryOne() {
	// Create test data
	user := &TestUser{ID: "user15", Name: "Single User", Status: "active"}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)
	
	// Query one
	var retrieved TestUser
	err = suite.userRepo.QueryOne(suite.ctx, &retrieved,
		gpa.Where("status", gpa.OpEqual, "active"),
	)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Single User", retrieved.Name)
}

func (suite *RedisAdapterTestSuite) TestCount() {
	// Create test data
	users := []*TestUser{
		{ID: "user16", Name: "User 16", Status: "active"},
		{ID: "user17", Name: "User 17", Status: "active"},
		{ID: "user18", Name: "User 18", Status: "inactive"},
	}
	
	err := suite.userRepo.CreateBatch(suite.ctx, users)
	require.NoError(suite.T(), err)
	
	// Count all
	count, err := suite.userRepo.Count(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), count)
	
	// Count with condition
	count, err = suite.userRepo.Count(suite.ctx, gpa.Where("status", gpa.OpEqual, "active"))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), count)
}

func (suite *RedisAdapterTestSuite) TestExists() {
	// Create test data
	user := &TestUser{ID: "user19", Name: "Exists User", Status: "active"}
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)
	
	// Test exists
	exists, err := suite.userRepo.Exists(suite.ctx, gpa.Where("status", gpa.OpEqual, "active"))
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)
	
	exists, err = suite.userRepo.Exists(suite.ctx, gpa.Where("status", gpa.OpEqual, "nonexistent"))
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *RedisAdapterTestSuite) TestGetEntityInfo() {
	info, err := suite.userRepo.GetEntityInfo(&TestUser{})
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "TestUser", info.Name)
	assert.Equal(suite.T(), []string{"ID"}, info.PrimaryKey)
	assert.True(suite.T(), len(info.Fields) > 0)
}

// =====================================
// Key-Value Operations Tests
// =====================================

func (suite *RedisAdapterTestSuite) TestKeyValueOperations() {
	redisRepo, ok := suite.userRepo.(*Repository)
	require.True(suite.T(), ok, "Repository should be a Redis repository")
	
	// Test Set and Get
	testData := map[string]interface{}{
		"name": "Test Value",
		"age":  25,
	}
	
	err := redisRepo.Set(suite.ctx, "test:key1", testData, 0)
	assert.NoError(suite.T(), err)
	
	var retrieved map[string]interface{}
	err = redisRepo.Get(suite.ctx, "test:key1", &retrieved)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test Value", retrieved["name"])
	
	// Test Exists (using KeyValue interface)
	kv := redisRepo.AsKeyValue()
	exists, err := kv.Exists(suite.ctx, "test:key1")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)
	
	// Test Delete (using KeyValue interface)
	err = kv.Delete(suite.ctx, "test:key1")
	assert.NoError(suite.T(), err)
	
	exists, err = kv.Exists(suite.ctx, "test:key1")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *RedisAdapterTestSuite) TestMGetMSet() {
	redisRepo, ok := suite.userRepo.(*Repository)
	require.True(suite.T(), ok)
	
	// Test MSet
	pairs := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	err := redisRepo.MSet(suite.ctx, pairs, 0)
	assert.NoError(suite.T(), err)
	
	// Test MGet
	keys := []string{"key1", "key2", "key3"}
	var values []interface{}
	err = redisRepo.MGet(suite.ctx, keys, &values)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), values, 3)
	
	// Test MDelete
	err = redisRepo.MDelete(suite.ctx, keys)
	assert.NoError(suite.T(), err)
	
	// Verify deletion
	var deletedValues []interface{}
	err = redisRepo.MGet(suite.ctx, keys, &deletedValues)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), deletedValues)
}

func (suite *RedisAdapterTestSuite) TestIncrement() {
	redisRepo, ok := suite.userRepo.(*Repository)
	require.True(suite.T(), ok)
	
	// Test increment
	result, err := redisRepo.Increment(suite.ctx, "counter1", 1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), result)
	
	result, err = redisRepo.Increment(suite.ctx, "counter1", 5)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(6), result)
}

func (suite *RedisAdapterTestSuite) TestTTL() {
	redisRepo, ok := suite.userRepo.(*Repository)
	require.True(suite.T(), ok)
	
	// Set with TTL
	err := redisRepo.Set(suite.ctx, "ttl:key", "value", time.Second*10)
	assert.NoError(suite.T(), err)
	
	// Check TTL
	ttl, err := redisRepo.TTL(suite.ctx, "ttl:key")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), ttl > 0 && ttl <= time.Second*10)
	
	// Set expire
	err = redisRepo.Expire(suite.ctx, "ttl:key", time.Second*5)
	assert.NoError(suite.T(), err)
	
	ttl, err = redisRepo.TTL(suite.ctx, "ttl:key")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), ttl > 0 && ttl <= time.Second*5)
}

func (suite *RedisAdapterTestSuite) TestKeysAndScan() {
	redisRepo, ok := suite.userRepo.(*Repository)
	require.True(suite.T(), ok)
	
	// Create test keys
	pairs := map[string]interface{}{
		"test:1": "value1",
		"test:2": "value2",
		"other": "value3",
	}
	
	err := redisRepo.MSet(suite.ctx, pairs, 0)
	require.NoError(suite.T(), err)
	
	// Test Keys
	keys, err := redisRepo.Keys(suite.ctx, "test:*")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), keys, 2)
	
	// Test Scan
	scanKeys, cursor, err := redisRepo.Scan(suite.ctx, 0, "*", 10)
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(scanKeys), 3)
	assert.Equal(suite.T(), uint64(0), cursor) // Should complete in one scan
}

// Run the test suite
func TestRedisAdapterSuite(t *testing.T) {
	// Register the Redis provider
	gpa.RegisterProvider("redis", &Factory{})
	
	suite.Run(t, new(RedisAdapterTestSuite))
}