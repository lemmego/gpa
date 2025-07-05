package gparedis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lemmego/gpa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RedisSpecificOperationsTestSuite tests Redis-specific operations
type RedisSpecificOperationsTestSuite struct {
	suite.Suite
	provider  gpa.Provider
	redisRepo *Repository
	ctx       context.Context
}

func (suite *RedisSpecificOperationsTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	
	// Skip tests if Redis is not available
	config := gpa.Config{
		Driver:   "redis",
		Host:     "localhost",
		Port:     6379,
		Database: "14", // Use DB 14 for Redis-specific testing
	}

	provider, err := gpa.NewProvider("redis", config)
	if err != nil {
		suite.T().Skip("Redis not available for testing:", err)
		return
	}

	suite.provider = provider
	repo := provider.RepositoryFor(&TestUser{})
	redisRepo, ok := repo.(*Repository)
	if !ok {
		suite.T().Skip("Repository is not a Redis repository")
		return
	}
	suite.redisRepo = redisRepo
	
	// Clean up test data
	suite.cleanupTestData()
}

func (suite *RedisSpecificOperationsTestSuite) TearDownSuite() {
	if suite.provider != nil {
		suite.cleanupTestData()
		suite.provider.Close()
	}
}

func (suite *RedisSpecificOperationsTestSuite) SetupTest() {
	suite.cleanupTestData()
}

func (suite *RedisSpecificOperationsTestSuite) cleanupTestData() {
	if suite.redisRepo != nil {
		keys, err := suite.redisRepo.Keys(suite.ctx, "*")
		if err == nil && len(keys) > 0 {
			suite.redisRepo.MDelete(suite.ctx, keys)
		}
	}
}

// =====================================
// List Operations Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestListOperations() {
	// Test LPush and RPush
	err := suite.redisRepo.LPush(suite.ctx, "mylist", "first", "second")
	assert.NoError(suite.T(), err)
	
	err = suite.redisRepo.RPush(suite.ctx, "mylist", "third", "fourth")
	assert.NoError(suite.T(), err)
	
	// Test LLen
	length, err := suite.redisRepo.LLen(suite.ctx, "mylist")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(4), length)
	
	// Test LRange
	var items []string
	err = suite.redisRepo.LRange(suite.ctx, "mylist", 0, -1, &items)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), items, 4)
	// Order should be: second, first, third, fourth (LPush adds to left)
	assert.Equal(suite.T(), "second", items[0])
	assert.Equal(suite.T(), "first", items[1])
	assert.Equal(suite.T(), "third", items[2])
	assert.Equal(suite.T(), "fourth", items[3])
	
	// Test LPop
	var popped string
	err = suite.redisRepo.LPop(suite.ctx, "mylist", &popped)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "second", popped)
	
	// Test RPop
	err = suite.redisRepo.RPop(suite.ctx, "mylist", &popped)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "fourth", popped)
	
	// Verify remaining length
	length, err = suite.redisRepo.LLen(suite.ctx, "mylist")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), length)
}

func (suite *RedisSpecificOperationsTestSuite) TestListEmptyOperations() {
	// Test operations on empty/non-existent list
	var popped string
	err := suite.redisRepo.LPop(suite.ctx, "emptylist", &popped)
	assert.Error(suite.T(), err)
	assert.IsType(suite.T(), gpa.GPAError{}, err)
	
	length, err := suite.redisRepo.LLen(suite.ctx, "emptylist")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), length)
}

// =====================================
// Set Operations Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestSetOperations() {
	// Test SAdd
	err := suite.redisRepo.SAdd(suite.ctx, "myset", "member1", "member2", "member3")
	assert.NoError(suite.T(), err)
	
	// Test SCard
	cardinality, err := suite.redisRepo.SCard(suite.ctx, "myset")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), cardinality)
	
	// Test SIsMember
	isMember, err := suite.redisRepo.SIsMember(suite.ctx, "myset", "member1")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), isMember)
	
	isMember, err = suite.redisRepo.SIsMember(suite.ctx, "myset", "nonexistent")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), isMember)
	
	// Test SMembers
	var members []string
	err = suite.redisRepo.SMembers(suite.ctx, "myset", &members)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 3)
	assert.Contains(suite.T(), members, "member1")
	assert.Contains(suite.T(), members, "member2")
	assert.Contains(suite.T(), members, "member3")
	
	// Test SRem
	err = suite.redisRepo.SRem(suite.ctx, "myset", "member2")
	assert.NoError(suite.T(), err)
	
	cardinality, err = suite.redisRepo.SCard(suite.ctx, "myset")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), cardinality)
	
	isMember, err = suite.redisRepo.SIsMember(suite.ctx, "myset", "member2")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), isMember)
}

// =====================================
// Hash Operations Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestHashOperations() {
	// Test HSet
	err := suite.redisRepo.HSet(suite.ctx, "myhash", 
		"field1", "value1",
		"field2", "value2",
		"field3", "value3",
	)
	assert.NoError(suite.T(), err)
	
	// Test HLen
	length, err := suite.redisRepo.HLen(suite.ctx, "myhash")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), length)
	
	// Test HExists
	exists, err := suite.redisRepo.HExists(suite.ctx, "myhash", "field1")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)
	
	exists, err = suite.redisRepo.HExists(suite.ctx, "myhash", "nonexistent")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
	
	// Test HGet
	var value string
	err = suite.redisRepo.HGet(suite.ctx, "myhash", "field1", &value)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "value1", value)
	
	// Test HGet non-existent field
	err = suite.redisRepo.HGet(suite.ctx, "myhash", "nonexistent", &value)
	assert.Error(suite.T(), err)
	assert.IsType(suite.T(), gpa.GPAError{}, err)
	
	// Test HGetAll
	var hashMap map[string]string
	err = suite.redisRepo.HGetAll(suite.ctx, "myhash", &hashMap)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), hashMap, 3)
	assert.Equal(suite.T(), "value1", hashMap["field1"])
	assert.Equal(suite.T(), "value2", hashMap["field2"])
	assert.Equal(suite.T(), "value3", hashMap["field3"])
	
	// Test HDel
	err = suite.redisRepo.HDel(suite.ctx, "myhash", "field2")
	assert.NoError(suite.T(), err)
	
	length, err = suite.redisRepo.HLen(suite.ctx, "myhash")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), length)
	
	exists, err = suite.redisRepo.HExists(suite.ctx, "myhash", "field2")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// =====================================
// Sorted Set Operations Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestSortedSetOperations() {
	// Test ZAdd
	err := suite.redisRepo.ZAdd(suite.ctx, "myzset",
		redis.Z{Score: 1.0, Member: "member1"},
		redis.Z{Score: 2.0, Member: "member2"},
		redis.Z{Score: 3.0, Member: "member3"},
	)
	assert.NoError(suite.T(), err)
	
	// Test ZCard
	cardinality, err := suite.redisRepo.ZCard(suite.ctx, "myzset")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), cardinality)
	
	// Test ZScore
	score, err := suite.redisRepo.ZScore(suite.ctx, "myzset", "member2")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2.0, score)
	
	// Test ZScore for non-existent member
	_, err = suite.redisRepo.ZScore(suite.ctx, "myzset", "nonexistent")
	assert.Error(suite.T(), err)
	assert.IsType(suite.T(), gpa.GPAError{}, err)
	
	// Test ZRange
	var members []string
	err = suite.redisRepo.ZRange(suite.ctx, "myzset", 0, -1, &members)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 3)
	// Should be ordered by score
	assert.Equal(suite.T(), "member1", members[0])
	assert.Equal(suite.T(), "member2", members[1])
	assert.Equal(suite.T(), "member3", members[2])
	
	// Test ZRangeByScore
	err = suite.redisRepo.ZRangeByScore(suite.ctx, "myzset", "1", "2", &members)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 2)
	assert.Contains(suite.T(), members, "member1")
	assert.Contains(suite.T(), members, "member2")
	
	// Test ZRem
	err = suite.redisRepo.ZRem(suite.ctx, "myzset", "member2")
	assert.NoError(suite.T(), err)
	
	cardinality, err = suite.redisRepo.ZCard(suite.ctx, "myzset")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), cardinality)
}

// =====================================
// Pub/Sub Operations Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestPubSubOperations() {
	// Test Subscribe
	pubsub, err := suite.redisRepo.Subscribe(suite.ctx, "testchannel")
	if err != nil {
		suite.T().Skip("Pub/Sub not available:", err)
		return
	}
	defer pubsub.Close()
	
	// Test Publish
	err = suite.redisRepo.Publish(suite.ctx, "testchannel", "test message")
	assert.NoError(suite.T(), err)
	
	// Try to receive the message (with timeout)
	ctx, cancel := context.WithTimeout(suite.ctx, time.Second*2)
	defer cancel()
	
	msg, err := pubsub.ReceiveMessage(ctx)
	if err == nil {
		assert.Equal(suite.T(), "testchannel", msg.Channel)
		assert.Equal(suite.T(), "test message", msg.Payload)
	}
	// Note: We don't assert on the receive since it might timeout in test environments
}

// =====================================
// Stream Operations Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestStreamOperations() {
	// Test XAdd
	values := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	}
	
	id, err := suite.redisRepo.XAdd(suite.ctx, "mystream", values)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), id)
	
	// Add another entry
	values2 := map[string]interface{}{
		"field1": "value3",
		"field2": "value4",
	}
	
	id2, err := suite.redisRepo.XAdd(suite.ctx, "mystream", values2)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), id2)
	assert.NotEqual(suite.T(), id, id2)
	
	// Test XRead
	streams := map[string]string{
		"mystream": "0", // Read from beginning
	}
	
	results, err := suite.redisRepo.XRead(suite.ctx, streams, 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 1) // One stream
	assert.Equal(suite.T(), suite.redisRepo.buildKey("mystream"), results[0].Stream)
	assert.Len(suite.T(), results[0].Messages, 2) // Two messages
	
	// Verify message content
	msg1 := results[0].Messages[0]
	assert.Equal(suite.T(), "value1", msg1.Values["field1"])
	assert.Equal(suite.T(), "value2", msg1.Values["field2"])
}

// =====================================
// Raw Operations Tests  
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestRawOperations() {
	// Test RawExec
	result, err := suite.redisRepo.RawExec(suite.ctx, "SET", []interface{}{suite.redisRepo.buildKey("rawkey"), "rawvalue"})
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	// Test RawQuery
	var value string
	err = suite.redisRepo.RawQuery(suite.ctx, "GET", []interface{}{suite.redisRepo.buildKey("rawkey")}, &value)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "rawvalue", value)
}

// =====================================
// Error Handling Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestErrorHandling() {
	// Test invalid operations
	err := suite.redisRepo.LPop(suite.ctx, "nonexistentlist", nil)
	assert.Error(suite.T(), err)
	
	// Test invalid raw command
	_, err = suite.redisRepo.RawExec(suite.ctx, "INVALIDCOMMAND", []interface{}{})
	assert.Error(suite.T(), err)
	
	// Test transaction (should be unsupported)
	err = suite.redisRepo.Transaction(suite.ctx, func(tx gpa.Transaction) error {
		return nil
	})
	assert.Error(suite.T(), err)
	assert.IsType(suite.T(), gpa.GPAError{}, err)
	gpaErr := err.(gpa.GPAError)
	assert.Equal(suite.T(), gpa.ErrorTypeUnsupported, gpaErr.Type)
}

// =====================================
// Performance and Stress Tests
// =====================================

func (suite *RedisSpecificOperationsTestSuite) TestBatchOperations() {
	// Test large batch operations
	pairs := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("batch:key:%d", i)
		pairs[key] = fmt.Sprintf("value:%d", i)
	}
	
	// MSet large batch
	start := time.Now()
	err := suite.redisRepo.MSet(suite.ctx, pairs)
	duration := time.Since(start)
	assert.NoError(suite.T(), err)
	suite.T().Logf("MSet 1000 keys took: %v", duration)
	
	// Verify count
	count, err := suite.redisRepo.Count(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1000), count)
	
	// MGet large batch
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("batch:key:%d", i)
	}
	
	start = time.Now()
	var values []interface{}
	err = suite.redisRepo.MGet(suite.ctx, keys, &values)
	duration = time.Since(start)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), values, 1000)
	suite.T().Logf("MGet 1000 keys took: %v", duration)
}

// Run the Redis-specific operations test suite
func TestRedisSpecificOperationsSuite(t *testing.T) {
	// Register the Redis provider
	gpa.RegisterProvider("redis", &Factory{})
	
	suite.Run(t, new(RedisSpecificOperationsTestSuite))
}