package repo

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedis(t *testing.T) (*miniredis.Miniredis, *User) {
	t.Helper()

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

func TestEncodeDecodeKeyMember(t *testing.T) {
	tests := []struct {
		tokenType      core.TokenType
		name           string
		keyID          string
		expectedMember string
		expectedType   core.TokenType
	}{
		{
			name:           "web token encodes with w: prefix",
			keyID:          "key123",
			tokenType:      core.TokenTypeWeb,
			expectedMember: "w:key123",
			expectedType:   core.TokenTypeWeb,
		},
		{
			name:           "tcp token encodes with t: prefix",
			keyID:          "key456",
			tokenType:      core.TokenTypeTCP,
			expectedMember: "t:key456",
			expectedType:   core.TokenTypeTCP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member := encodeKeyMember(tt.keyID, tt.tokenType)
			assert.Equal(t, tt.expectedMember, member)

			decodedID, decodedType := decodeKeyMember(member)
			assert.Equal(t, tt.keyID, decodedID)
			assert.Equal(t, tt.expectedType, decodedType)
		})
	}
}

func TestDecodeKeyMember_BackwardCompat(t *testing.T) {
	// Bare members (no prefix) existed before type support — treat as web.
	keyID, tokenType := decodeKeyMember("legacykeyid")
	assert.Equal(t, "legacykeyid", keyID)
	assert.Equal(t, core.TokenTypeWeb, tokenType)
}

func TestAddAPIKey(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "user123"
	apiKeyID := "key123"
	expiresIn := 3600 * time.Second

	// Test successful add (web)
	err := user.AddAPIKey(ctx, userID, apiKeyID, core.TokenTypeWeb, expiresIn)
	assert.NoError(t, err)

	// Verify key is retrievable
	keys, err := user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Contains(t, keys, apiKeyID)

	// Test adding the same key again (idempotent)
	err = user.AddAPIKey(ctx, userID, apiKeyID, core.TokenTypeWeb, expiresIn)
	assert.NoError(t, err)
}

func TestAddAPIKey_TypesAreStored(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "userTyped"

	err := user.AddAPIKey(ctx, userID, "webkey1", core.TokenTypeWeb, 3600*time.Second)
	require.NoError(t, err)

	err = user.AddAPIKey(ctx, userID, "tcpkey1", core.TokenTypeTCP, 3600*time.Second)
	require.NoError(t, err)

	keys, err := user.GetAPIKeysWithExpiration(ctx, userID)
	require.NoError(t, err)
	require.Len(t, keys, 2)

	// Verify types are correctly decoded.
	typesByID := map[string]core.TokenType{}
	for _, k := range keys {
		typesByID[k.KeyID] = k.Type
	}

	assert.Equal(t, core.TokenTypeWeb, typesByID["webkey1"])
	assert.Equal(t, core.TokenTypeTCP, typesByID["tcpkey1"])
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

	// Add a web key
	apiKeyID := "key123"
	expiresIn := 3600 * time.Second
	err = user.AddAPIKey(ctx, userID, apiKeyID, core.TokenTypeWeb, expiresIn)
	assert.NoError(t, err)

	// Returns bare key ID (prefix stripped)
	keys, err = user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, apiKeyID)
	assert.NotContains(t, keys, memberPrefixWeb+apiKeyID, "prefix should be stripped")

	// Add a TCP key
	apiKeyID2 := "key456"
	err = user.AddAPIKey(ctx, userID, apiKeyID2, core.TokenTypeTCP, expiresIn)
	assert.NoError(t, err)

	// Both keys returned as bare IDs
	keys, err = user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, apiKeyID)
	assert.Contains(t, keys, apiKeyID2)

	// Test expired keys are removed
	mr.FastForward(expiresIn + ttlOffset + time.Second)

	redisKey := user.keyPrefix + apiKeyPrefix + userID
	user.db.Del(ctx, redisKey)

	keys, err = user.GetAPIKeys(ctx, userID)
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestGetAPIKeys_BackwardCompat(t *testing.T) {
	// Bare keyIDs (legacy format) stored without prefix should be returned as-is.
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "legacyUser"
	redisKey := user.keyPrefix + apiKeyPrefix + userID

	// Insert a bare member directly (simulating legacy data)
	score := float64(time.Now().Add(time.Hour).Unix())
	err := user.db.ZAdd(ctx, redisKey, redis.Z{Score: score, Member: "barekey123"}).Err()
	require.NoError(t, err)

	keys, err := user.GetAPIKeys(ctx, userID)
	require.NoError(t, err)
	assert.Contains(t, keys, "barekey123")
}

func TestGetAPIKeysWithExpiration(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "userExp"

	// Empty state
	keys, err := user.GetAPIKeysWithExpiration(ctx, userID)
	assert.NoError(t, err)
	assert.Empty(t, keys)

	// Add a web key
	err = user.AddAPIKey(ctx, userID, "keyB", core.TokenTypeWeb, 48*time.Hour)
	require.NoError(t, err)

	// Directly insert an expired entry to simulate expiry
	redisKey := user.keyPrefix + apiKeyPrefix + userID
	expiredScore := float64(time.Now().Add(-time.Hour).Unix())
	err = user.db.ZAdd(ctx, redisKey, redis.Z{Score: expiredScore, Member: encodeKeyMember("keyA", core.TokenTypeWeb)}).Err()
	require.NoError(t, err)

	// Only keyB should be returned; keyA is expired
	keys, err = user.GetAPIKeysWithExpiration(ctx, userID)
	assert.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "keyB", keys[0].KeyID)
	assert.Equal(t, core.TokenTypeWeb, keys[0].Type)
	assert.True(t, keys[0].ExpiresAt.After(time.Now()), "expiry should be in the future")
}

func TestGetAPIKeysWithExpiration_BackwardCompat(t *testing.T) {
	// Bare members (no prefix) should be returned with TokenTypeWeb.
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "legacyUser2"
	redisKey := user.keyPrefix + apiKeyPrefix + userID

	score := float64(time.Now().Add(time.Hour).Unix())
	err := user.db.ZAdd(ctx, redisKey, redis.Z{Score: score, Member: "legacykey"}).Err()
	require.NoError(t, err)

	keys, err := user.GetAPIKeysWithExpiration(ctx, userID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "legacykey", keys[0].KeyID)
	assert.Equal(t, core.TokenTypeWeb, keys[0].Type)
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
	expiresIn := 3600 * time.Second

	// Add a key to revoke (as web)
	err := user.AddAPIKey(ctx, userID, "key123", core.TokenTypeWeb, expiresIn)
	require.NoError(t, err)

	tests := []struct {
		setup         func()
		name          string
		targetKey     string
		expectedKeys  []string
		expectedError bool
	}{
		{
			name:          "revoke existing web key",
			setup:         func() {},
			targetKey:     "key123",
			expectedError: false,
			expectedKeys:  []string{},
		},
		{
			name: "revoke non-existing key",
			setup: func() {
				mr.FlushAll()
			},
			targetKey:     "non_existent_key",
			expectedError: false,
			expectedKeys:  []string{},
		},
		{
			name: "revoke key when user has multiple keys",
			setup: func() {
				_ = user.AddAPIKey(ctx, userID, "key123", core.TokenTypeWeb, expiresIn)
				_ = user.AddAPIKey(ctx, userID, "key456", core.TokenTypeWeb, expiresIn)
			},
			targetKey:     "key123",
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

func TestRevokeToken_TCPKey(t *testing.T) {
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "userTCP"

	err := user.AddAPIKey(ctx, userID, "tcpkey1", core.TokenTypeTCP, 3600*time.Second)
	require.NoError(t, err)

	err = user.RevokeToken(ctx, userID, "tcpkey1")
	require.NoError(t, err)

	keys, err := user.GetAPIKeys(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestRevokeToken_LegacyBareKey(t *testing.T) {
	// Revoking a key stored as a bare (legacy) member should work.
	mr, user := setupRedis(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "legacyRevokeUser"
	redisKey := user.keyPrefix + apiKeyPrefix + userID

	score := float64(time.Now().Add(time.Hour).Unix())
	err := user.db.ZAdd(ctx, redisKey, redis.Z{Score: score, Member: "barekey"}).Err()
	require.NoError(t, err)

	err = user.RevokeToken(ctx, userID, "barekey")
	require.NoError(t, err)

	keys, err := user.GetAPIKeys(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, keys)
}
