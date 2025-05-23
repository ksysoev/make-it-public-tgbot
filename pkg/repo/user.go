package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ttlOffset = 60 * time.Second
)

type Config struct {
	RedisAddr string `mapstructure:"redis_addr"`
	Password  string `mapstructure:"redis_password"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

type User struct {
	db        *redis.Client
	keyPrefix string
}

// New initializes and returns a new User instance configured with the provided Config.
func New(cfg Config) *User {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.Password,
	})

	return &User{
		db:        rdb,
		keyPrefix: cfg.KeyPrefix,
	}
}

// Close terminates the connection to the Redis database and returns an error if the operation fails.
func (u *User) Close() error {
	return u.db.Close()
}

// AddAPIKey adds an API key with an expiration time to the user's Redis store. Returns an error if the operation fails.
func (u *User) AddAPIKey(ctx context.Context, userID string, apiKeyID string, expiresIn time.Duration) error {
	redisKey := u.keyPrefix + userID

	_, err := u.db.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(time.Now().Add(expiresIn - ttlOffset).Unix()),
		Member: apiKeyID,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add API key: %w", err)
	}

	// If the result is 0, it means the member already exists in the sorted set
	// This is not an error, so we don't need to return one

	return nil
}

// GetAPIKeys retrieves all API keys for a user from the Redis store. Returns a slice of API keys and an error if the operation fails.
func (u *User) GetAPIKeys(ctx context.Context, userID string) ([]string, error) {
	redisKey := u.keyPrefix + userID

	// clean up expired keys
	now := time.Now().Unix()
	_, err := u.db.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to remove expired API keys: %w", err)
	}

	// Get keys with scores greater than current time (not expired)
	keys, err := u.db.ZRangeByScore(ctx, redisKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", now),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	return keys, nil
}

// RevokeToken removes the specified API key for a user from the Redis store. Returns an error if the operation fails.
func (u *User) RevokeToken(ctx context.Context, userID string, apiKeyID string) error {
	redisKey := u.keyPrefix + userID

	_, err := u.db.ZRem(ctx, redisKey, apiKeyID).Result()
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	return nil
}
