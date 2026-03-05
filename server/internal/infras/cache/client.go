package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/s2"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/singleflight"
)

const (
	compressionThreshold = 64
	timeLen              = 4
)

const (
	noCompression = 0x0
	s2Compression = 0x1
)

var ErrCacheMiss = errors.New("cache: key is missing")

type rediser interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) *redis.StatusCmd
	SetXX(ctx context.Context, key string, value any, ttl time.Duration) *redis.BoolCmd
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) *redis.BoolCmd

	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type Item struct { //nolint:containedctx
	Key   string
	Value any

	// TTL is the cache expiration time.
	// Default TTL is 1 hour.
	TTL time.Duration

	// Do returns value to be cached.
	Do func(*Item) (any, error)

	// SetXX only sets the key if it already exists.
	SetXX bool

	// SetNX only sets the key if it does not already exist.
	SetNX bool
}

func (item *Item) value() (any, error) {
	if item.Do != nil {
		return item.Do(item)
	}
	if item.Value != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (item *Item) ttl() time.Duration {
	const defaultTTL = time.Hour

	if item.TTL < 0 {
		return 0
	}

	if item.TTL != 0 {
		if item.TTL < time.Second {
			log.Printf("too short TTL for key=%q: %s", item.Key, item.TTL)
			return defaultTTL
		}
		return item.TTL
	}

	return defaultTTL
}

// ------------------------------------------------------------------------------
type (
	MarshalFunc   func(any) ([]byte, error)
	UnmarshalFunc func([]byte, any) error
)

type Options struct {
	Redis        rediser
	StatsEnabled bool
	Marshal      MarshalFunc
	Unmarshal    UnmarshalFunc
}

type Client struct {
	opt *Options

	group singleflight.Group

	marshal   MarshalFunc
	unmarshal UnmarshalFunc

	hits   uint64
	misses uint64
}

func New(opt *Options) *Client {
	cacher := &Client{
		opt: opt,
	}

	if opt.Marshal == nil {
		cacher.marshal = cacher._marshal
	} else {
		cacher.marshal = opt.Marshal
	}

	if opt.Unmarshal == nil {
		cacher.unmarshal = cacher._unmarshal
	} else {
		cacher.unmarshal = opt.Unmarshal
	}
	return cacher
}

// Set caches the item.
func (cd *Client) Set(ctx context.Context, item *Item) error {
	_, _, err := cd.doSet(ctx, item)
	return err
}

func (cd *Client) doSet(ctx context.Context, item *Item) ([]byte, bool, error) {
	value, err := item.value()
	if err != nil {
		return nil, false, err
	}

	b, err := cd.Marshal(value)
	if err != nil {
		return nil, false, err
	}

	if cd.opt.Redis == nil {
		return b, true, nil
	}

	ttl := item.ttl()
	if ttl == 0 {
		return b, true, nil
	}

	if item.SetXX {
		return b, true, fmt.Errorf("redis setxx: %w", cd.opt.Redis.SetXX(ctx, item.Key, b, ttl).Err())
	}
	if item.SetNX {
		return b, true, fmt.Errorf("redis setnx: %w", cd.opt.Redis.SetNX(ctx, item.Key, b, ttl).Err())
	}
	return b, true, fmt.Errorf("redis set: %w", cd.opt.Redis.Set(ctx, item.Key, b, ttl).Err())
}

// Exists reports whether value for the given key exists.
func (cd *Client) Exists(ctx context.Context, key string) bool {
	_, err := cd.getBytes(ctx, key)
	return err == nil
}

// Get gets the value for the given key.
func (cd *Client) Get(ctx context.Context, key string, value any) error {
	return cd.doGet(ctx, key, value)
}

func (cd *Client) doGet(ctx context.Context, key string, value any) error {
	b, err := cd.getBytes(ctx, key)
	if err != nil {
		return err
	}
	return cd.unmarshal(b, value)
}

func (cd *Client) getBytes(ctx context.Context, key string) ([]byte, error) {
	if cd.opt.Redis == nil {
		return nil, ErrCacheMiss
	}

	b, err := cd.opt.Redis.Get(ctx, key).Bytes()
	if err != nil {
		if cd.opt.StatsEnabled {
			atomic.AddUint64(&cd.misses, 1)
		}
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}

	if cd.opt.StatsEnabled {
		atomic.AddUint64(&cd.hits, 1)
	}
	return b, nil
}

// Once gets the item.Value for the given item.Key from the cache or
// executes, caches, and returns the results of the given item.Func,
// making sure that only one execution is in-flight for a given item.Key
// at a time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
func (cd *Client) Once(ctx context.Context, item *Item) error {
	b, cached, err := cd.getSetItemBytesOnce(ctx, item)
	if err != nil {
		return err
	}

	if item.Value == nil || len(b) == 0 {
		return nil
	}

	if err := cd.unmarshal(b, item.Value); err != nil {
		if cached {
			_ = cd.Delete(ctx, item.Key)
			return cd.Once(ctx, item)
		}
		return err
	}

	return nil
}

func (cd *Client) getSetItemBytesOnce(ctx context.Context, item *Item) (b []byte, cached bool, err error) {
	v, err, _ := cd.group.Do(item.Key, func() (any, error) {
		b, err := cd.getBytes(ctx, item.Key)
		if err == nil {
			cached = true
			return b, nil
		}

		b, ok, err := cd.doSet(ctx, item)
		if ok {
			return b, nil
		}
		return nil, err
	})
	if err != nil {
		return nil, false, fmt.Errorf("singleflight: %w", err)
	}
	vb, ok := v.([]byte)
	if !ok {
		return nil, cached, fmt.Errorf("invalid type assertion: expected []byte, got %T", v)
	}
	return vb, cached, nil
}

func (cd *Client) Delete(ctx context.Context, keys ...string) error {
	if cd.opt.Redis == nil {
		return nil
	}

	_, err := cd.opt.Redis.Del(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}

func (cd *Client) Marshal(value any) ([]byte, error) {
	return cd.marshal(value)
}

func (cd *Client) _marshal(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}

	b, err := msgpack.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("msgpack marshal: %w", err)
	}

	return compress(b), nil
}

func compress(data []byte) []byte {
	if len(data) < compressionThreshold {
		n := len(data) + 1
		b := make([]byte, n, n+timeLen)
		copy(b, data)
		b[len(b)-1] = noCompression
		return b
	}

	n := s2.MaxEncodedLen(len(data)) + 1
	b := make([]byte, n, n+timeLen)
	b = s2.Encode(b, data)
	b = append(b, s2Compression)
	return b
}

func (cd *Client) Unmarshal(b []byte, value any) error {
	return cd.unmarshal(b, value)
}

func (cd *Client) _unmarshal(b []byte, value any) error {
	if len(b) == 0 {
		return nil
	}

	if value == nil {
		return nil
	}

	switch c := b[len(b)-1]; c {
	case noCompression:
		b = b[:len(b)-1]
	case s2Compression:
		b = b[:len(b)-1]

		var err error
		b, err = s2.Decode(nil, b)
		if err != nil {
			return fmt.Errorf("s2 decode: %w", err)
		}
	default:
		return fmt.Errorf("unknown compression method: %x", c)
	}

	return fmt.Errorf("msgpack unmarshal: %w", msgpack.Unmarshal(b, value))
}

// Generic helpers

func Get[T any](ctx context.Context, c *Client, key string) (T, error) {
	var dest T
	err := c.Get(ctx, key, &dest)
	return dest, err
}

func Set[T any](ctx context.Context, c *Client, key string, value T, ttl time.Duration) error {
	return c.Set(ctx, &Item{
		Key:   key,
		Value: value,
		TTL:   ttl,
	})
}

func BuildKey(parts ...any) string {
	s := make([]string, 0, len(parts))
	for _, p := range parts {
		s = append(s, fmt.Sprint(p))
	}
	return strings.Join(s, ":")
}

// ------------------------------------------------------------------------------

type Stats struct {
	Hits   uint64
	Misses uint64
}

// Stats returns cache statistics.
func (cd *Client) Stats() *Stats {
	if !cd.opt.StatsEnabled {
		return nil
	}
	return &Stats{
		Hits:   atomic.LoadUint64(&cd.hits),
		Misses: atomic.LoadUint64(&cd.misses),
	}
}
