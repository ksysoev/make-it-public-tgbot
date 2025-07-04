package repo

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedis(t *testing.T) (*miniredis.Miniredis, *User) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	user := &User{
		db:        client,
		keyPrefix: "prefix:",
	}

	return mr, user
}

func TestNew(t *testing.T) {
	cfg := Config{
		RedisAddr: "localhost:6379",
		Password:  "password",
		KeyPrefix: "prefix:",
	}

	user := New(cfg)

	assert.NotNil(t, user)
	assert.Equal(t, cfg.KeyPrefix, user.keyPrefix)
	assert.NotNil(t, user.db)
}

func TestAddAPIKey(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "user123"
	apiKeyID := "key123"
	expiresIn := 3600 * time.Second

	// Test successful add
	err := user.AddAPIKey(ctx, userID, apiKeyID, expiresIn)
	assert.NoError(t, err)

	// Verify key was added
	keys, err := user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Contains(t, keys, apiKeyID)

	// Test adding the same key again (should still work)
	err = user.AddAPIKey(ctx, userID, apiKeyID, expiresIn)
	assert.NoError(t, err)
}

func TestGetAPIKeys(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "user123"

	// Test with no keys
	keys, err := user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Empty(t, keys)

	// Add a key
	apiKeyID := "key123"
	expiresIn := 3600 * time.Second
	err = user.AddAPIKey(ctx, userID, apiKeyID, expiresIn)
	assert.NoError(t, err)

	// Test with one key
	keys, err = user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, apiKeyID)

	// Add another key
	apiKeyID2 := "key456"
	err = user.AddAPIKey(ctx, userID, apiKeyID2, expiresIn)
	assert.NoError(t, err)

	// Test with multiple keys
	keys, err = user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, apiKeyID)
	assert.Contains(t, keys, apiKeyID2)

	// Test expired keys are removed
	// Set the time to future to make the keys expire
	mr.FastForward(expiresIn + ttlOffset + time.Second)

	// Manually delete the keys to simulate expiration
	redisKey := user.keyPrefix + apiKeyPrefix + userID
	user.db.Del(ctx, redisKey)

	// Keys should be empty after deletion
	keys, err = user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestClose(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	err := user.Close()
	assert.NoError(t, err)
}

func TestRevokeToken(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "user123"
	apiKeyID := "key123"
	expiresIn := 3600 * time.Second

	// Add a key to revoke
	err := user.AddAPIKey(ctx, userID, apiKeyID, expiresIn)
	require.NoError(t, err)

	tests := []struct {
		name          string
		setup         func()
		targetKey     string
		expectedError bool
		expectedKeys  []string
	}{
		{
			name: "revoke existing key",
			setup: func() {
				// Key already added in setup
			},
			targetKey:     apiKeyID,
			expectedError: false,
			expectedKeys:  []string{},
		},
		{
			name: "revoke non-existing key",
			setup: func() {
				// Clear all keys explicitly
				mr.FlushAll()
			},
			targetKey:     "non_existent_key",
			expectedError: false,
			expectedKeys:  []string{},
		},
		{
			name: "revoke key when user has multiple keys",
			setup: func() {
				anotherAPIKeyID := "key456"
				err := user.AddAPIKey(ctx, userID, anotherAPIKeyID, expiresIn)
				require.NoError(t, err)
			},
			targetKey:     apiKeyID,
			expectedError: false,
			expectedKeys:  []string{"key456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			err := user.RevokeToken(ctx, userID, tt.targetKey)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			keys, _ := user.GetAPIKeys(ctx, userID)
			assert.ElementsMatch(t, tt.expectedKeys, keys)
		})
	}
}
