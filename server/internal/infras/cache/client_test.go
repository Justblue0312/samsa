package cache

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUser is a sample struct for testing
type TestUser struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
}

// setupRedis creates a test Redis client
func setupRedis(t *testing.T) redis.UniversalClient {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use a separate DB for tests
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up test data
	t.Cleanup(func() {
		client.FlushDB(ctx)
		_ = client.Close()
	})
	client.FlushDB(ctx)

	return client
}

func TestCache_SetAndGet(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := New(&Options{Redis: rdb})

	t.Run("set and get simple value", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key1", Value: "value1"})
		require.NoError(t, err)

		var rs string
		err = cache.Get(ctx, "key1", &rs)
		assert.NoError(t, err)
		assert.Equal(t, "value1", rs)
	})

	t.Run("set and get struct", func(t *testing.T) {
		user := &TestUser{
			ID:        123,
			Name:      "John Doe",
			Email:     "john@example.com",
			CreatedAt: time.Now().Truncate(time.Second).UTC(),
		}

		err := cache.Set(ctx, &Item{Key: "user:123", Value: user})
		require.NoError(t, err)

		var retrievedUser TestUser
		err = cache.Get(ctx, "user:123", &retrievedUser)
		assert.NoError(t, err)

		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Name, retrievedUser.Name)
		assert.Equal(t, user.Email, retrievedUser.Email)
		assert.WithinDuration(t, user.CreatedAt, retrievedUser.CreatedAt, time.Second)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		var value string
		err := cache.Get(ctx, "non-existent", &value)
		assert.Error(t, err)
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("set with custom TTL", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "short-lived", Value: "value", TTL: 1 * time.Second})
		require.NoError(t, err)

		var value string
		err = cache.Get(ctx, "short-lived", &value)
		assert.NoError(t, err)
		assert.Equal(t, "value", value)

		time.Sleep(1100 * time.Millisecond)
		err = cache.Get(ctx, "short-lived", &value)
		assert.Error(t, err)
		assert.Equal(t, ErrCacheMiss, err)
	})
}

func TestCache_Compression(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := New(&Options{Redis: rdb})

	t.Run("small data no compression", func(t *testing.T) {
		smallData := "small"
		err := cache.Set(ctx, &Item{Key: "small", Value: smallData})
		require.NoError(t, err)

		raw, err := rdb.Get(ctx, "small").Bytes()
		require.NoError(t, err)
		assert.Equal(t, byte(noCompression), raw[len(raw)-1])

		var retrieved string
		err = cache.Get(ctx, "small", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, smallData, retrieved)
	})

	t.Run("large data with compression", func(t *testing.T) {
		largeData := strings.Repeat("a", 1000)
		err := cache.Set(ctx, &Item{Key: "large", Value: largeData})
		require.NoError(t, err)

		raw, err := rdb.Get(ctx, "large").Bytes()
		require.NoError(t, err)
		assert.Equal(t, byte(s2Compression), raw[len(raw)-1])

		var retrieved string
		err = cache.Get(ctx, "large", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, largeData, retrieved)
	})
}

func TestCache_Delete(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := New(&Options{Redis: rdb})

	t.Run("delete existing key", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key1", Value: "value1"})
		require.NoError(t, err)

		err = cache.Delete(ctx, "key1")
		require.NoError(t, err)

		var value string
		err = cache.Get(ctx, "key1", &value)
		assert.Error(t, err)
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		err := cache.Delete(ctx, "non-existent")
		assert.NoError(t, err)
	})
}

func TestCache_Exists(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := New(&Options{Redis: rdb})

	t.Run("exists returns true for existing key", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key1", Value: "value1"})
		require.NoError(t, err)

		exists := cache.Exists(ctx, "key1")
		assert.True(t, exists)
	})

	t.Run("exists returns false for non-existent key", func(t *testing.T) {
		exists := cache.Exists(ctx, "non-existent")
		assert.False(t, exists)
	})
}

func TestCache_SetXX_SetNX(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := New(&Options{Redis: rdb})

	// Set initial value
	err := cache.Set(ctx, &Item{Key: "key", Value: "value1"})
	require.NoError(t, err)

	t.Run("SetNX on existing key fails", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key", Value: "value2", SetNX: true})
		require.NoError(t, err)

		var val string
		err = cache.Get(ctx, "key", &val)
		require.NoError(t, err)
		assert.Equal(t, "value1", val)
	})

	t.Run("SetXX on existing key succeeds", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key", Value: "value3", SetXX: true})
		require.NoError(t, err)

		var val string
		err = cache.Get(ctx, "key", &val)
		require.NoError(t, err)
		assert.Equal(t, "value3", val)
	})

	// Delete key
	err = cache.Delete(ctx, "key")
	require.NoError(t, err)

	t.Run("SetXX on non-existing key fails", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key", Value: "value4", SetXX: true})
		require.NoError(t, err)

		var val string
		err = cache.Get(ctx, "key", &val)
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("SetNX on non-existing key succeeds", func(t *testing.T) {
		err := cache.Set(ctx, &Item{Key: "key", Value: "value5", SetNX: true})
		require.NoError(t, err)

		var val string
		err = cache.Get(ctx, "key", &val)
		require.NoError(t, err)
		assert.Equal(t, "value5", val)
	})
}

func TestCache_Once(t *testing.T) {
	ctx := context.Background()
	rdb := setupRedis(t)
	cache := New(&Options{Redis: rdb})

	var value string
	item := &Item{
		Key:   "once-key",
		Value: &value,
	}

	t.Run("first call executes Do", func(t *testing.T) {
		var callCount int32
		item.Do = func(_ *Item) (any, error) {
			atomic.AddInt32(&callCount, 1)
			return "value-from-do", nil
		}

		err := cache.Once(ctx, item)
		require.NoError(t, err)
		assert.Equal(t, "value-from-do", value)
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	})

	t.Run("second call uses cache", func(t *testing.T) {
		// Reset value and Do to ensure they aren't used
		value = ""
		var callCount int32
		item.Do = func(_ *Item) (any, error) {
			atomic.AddInt32(&callCount, 1)
			return "new-value", nil
		}

		err := cache.Once(ctx, item)
		require.NoError(t, err)
		assert.Equal(t, "value-from-do", value) // Should get the cached value
		assert.Equal(t, int32(0), atomic.LoadInt32(&callCount))
	})

	t.Run("concurrent calls only execute Do once", func(t *testing.T) {
		key := "concurrent-key"
		var callCount int32
		var wg sync.WaitGroup
		wg.Add(10)

		for range 10 {
			go func() {
				defer wg.Done()
				var v string
				err := cache.Once(ctx, &Item{
					Key:   key,
					Value: &v,
					Do: func(_ *Item) (any, error) {
						atomic.AddInt32(&callCount, 1)
						// Simulate work
						time.Sleep(10 * time.Millisecond)
						return "concurrent-value", nil
					},
				})
				require.NoError(t, err)
				assert.Equal(t, "concurrent-value", v)
			}()
		}

		wg.Wait()
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	})
}

func TestCache_Stats(t *testing.T) {
	rdb := setupRedis(t)
	cache := New(&Options{Redis: rdb, StatsEnabled: true})
	ctx := context.Background()

	var val string

	// Initial state
	stats := cache.Stats()
	require.NotNil(t, stats)
	assert.Equal(t, uint64(0), stats.Hits)
	assert.Equal(t, uint64(0), stats.Misses)

	// Cache miss
	err := cache.Get(ctx, "key1", &val)
	assert.Equal(t, ErrCacheMiss, err)
	stats = cache.Stats()
	assert.Equal(t, uint64(0), stats.Hits)
	assert.Equal(t, uint64(1), stats.Misses)

	// Cache hit
	err = cache.Set(ctx, &Item{Key: "key1", Value: "value1"})
	require.NoError(t, err)

	err = cache.Get(ctx, "key1", &val)
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
	stats = cache.Stats()
	assert.Equal(t, uint64(1), stats.Hits)
	assert.Equal(t, uint64(1), stats.Misses)

	// Another hit
	err = cache.Get(ctx, "key1", &val)
	require.NoError(t, err)
	stats = cache.Stats()
	assert.Equal(t, uint64(2), stats.Hits)
	assert.Equal(t, uint64(1), stats.Misses)

	t.Run("stats disabled", func(t *testing.T) {
		cacheDisabled := New(&Options{Redis: rdb, StatsEnabled: false})
		assert.Nil(t, cacheDisabled.Stats())
	})
}
