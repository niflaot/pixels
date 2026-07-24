package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps Redis operations used by reusable components.
type Client struct {
	// client is the underlying Redis driver.
	client *goredis.Client
}

// New creates a Redis client.
func New(config Config) *Client {
	return &Client{
		client: goredis.NewClient(&goredis.Options{
			Addr:     config.Address,
			Username: config.Username,
			Password: config.Password,
			DB:       config.Database,
		}),
	}
}

// Close closes the Redis client.
func (client *Client) Close() error {
	return client.client.Close()
}

// Delete removes a Redis key.
func (client *Client) Delete(ctx context.Context, key string) error {
	return client.client.Del(ctx, key).Err()
}

// Expire updates the expiration duration for a Redis key.
func (client *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return client.client.Expire(ctx, key, ttl).Err()
}

// Find reads a Redis key and reports whether it exists.
func (client *Client) Find(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := client.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

// Increment atomically increments a key and sets its expiration only on first use.
func (client *Client) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := client.client.Pipeline()
	counter := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	return counter.Val(), nil
}

// ListLength returns the number of values stored in a Redis list.
func (client *Client) ListLength(ctx context.Context, key string) (int64, error) {
	return client.client.LLen(ctx, key).Result()
}

// ListRange returns an inclusive range of values from a Redis list.
func (client *Client) ListRange(ctx context.Context, key string, start int64, stop int64) ([][]byte, error) {
	values, err := client.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, err
	}

	result := make([][]byte, len(values))
	for index, value := range values {
		result[index] = []byte(value)
	}

	return result, nil
}

// ListAppend appends values to a Redis list.
func (client *Client) ListAppend(ctx context.Context, key string, values ...[]byte) error {
	arguments := make([]any, len(values))
	for index, value := range values {
		arguments[index] = value
	}

	return client.client.RPush(ctx, key, arguments...).Err()
}

// SetAdd adds members to a Redis set.
func (client *Client) SetAdd(ctx context.Context, key string, members ...string) error {
	arguments := make([]any, len(members))
	for index, member := range members {
		arguments[index] = member
	}

	return client.client.SAdd(ctx, key, arguments...).Err()
}

// SetMembers returns every member of a Redis set.
func (client *Client) SetMembers(ctx context.Context, key string) ([]string, error) {
	return client.client.SMembers(ctx, key).Result()
}

// SetRemove removes members from a Redis set.
func (client *Client) SetRemove(ctx context.Context, key string, members ...string) error {
	arguments := make([]any, len(members))
	for index, member := range members {
		arguments[index] = member
	}

	return client.client.SRem(ctx, key, arguments...).Err()
}

// Set writes a Redis key with an optional expiration duration.
func (client *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return client.client.Set(ctx, key, value, ttl).Err()
}

// SetIfAbsent writes a key only when it does not already exist.
func (client *Client) SetIfAbsent(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return client.client.SetNX(ctx, key, value, ttl).Result()
}

// Take reads and deletes a Redis key atomically.
func (client *Client) Take(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := client.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}
