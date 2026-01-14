package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Encoder interface for caching complex objects
type Encoder interface {
	MarshalBinary() (data []byte, err error)
}

// Client wraps redis client
type Client struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewRedisClient creates a new cache client
func NewRedisClient(addr string, ttl time.Duration) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{
		rdb: rdb,
		ttl: ttl,
	}, nil
}

// Get retrieves a value from cache and unmarshals it into v
func (c *Client) Get(ctx context.Context, key string, v interface{}) (bool, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Cache miss
		}
		return false, err
	}

	if err := json.Unmarshal(val, v); err != nil {
		return false, err
	}

	return true, nil // Cache hit
}

// Set stores a value in cache with default TTL
func (c *Client) Set(ctx context.Context, key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return c.rdb.Set(ctx, key, data, c.ttl).Err()
}

// Close closes the redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}
