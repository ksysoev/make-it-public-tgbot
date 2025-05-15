package repo

import (
	"context"
	"fmt"
	"sync"
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
	mu        sync.Mutex
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

	res, err := u.db.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(time.Now().Add(expiresIn - ttlOffset).Unix()),
		Member: apiKeyID,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add API key: %w", err)
	}

	if res == 0 {
		return fmt.Errorf("no API key added")
	}

	return nil
}

// GetAPIKeys retrieves all API keys for a user from the Redis store. Returns a slice of API keys and an error if the operation fails.
func (u *User) GetAPIKeys(ctx context.Context, userID string) ([]string, error) {
	redisKey := u.keyPrefix + userID

	// clean up expired keys
	_, err := u.db.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", time.Now().Unix())).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to remove expired API keys: %w", err)
	}

	keys, err := u.db.ZRange(ctx, redisKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	return keys, nil
}
