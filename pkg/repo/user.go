package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
	"github.com/redis/go-redis/v9"
)

const (
	ttlOffset     = 60 * time.Second
	apiKeyPrefix  = "USER_KEYS::"
	convKeyPrefix = "CONV::"
	convTTL       = 24 * time.Hour // Default TTL for conversations

	// memberPrefixWeb is the sorted-set member prefix for web tokens.
	memberPrefixWeb = "w:"
	// memberPrefixTCP is the sorted-set member prefix for TCP tokens.
	memberPrefixTCP = "t:"
)

// encodeKeyMember encodes a key ID and its token type into the Redis sorted-set member string.
// Format: "w:<keyID>" for web, "t:<keyID>" for TCP.
func encodeKeyMember(keyID string, tokenType core.TokenType) string {
	switch tokenType {
	case core.TokenTypeTCP:
		return memberPrefixTCP + keyID
	default:
		return memberPrefixWeb + keyID
	}
}

// decodeKeyMember decodes a Redis sorted-set member back into a key ID and token type.
// Bare members (no known prefix) are treated as web for backward compatibility.
func decodeKeyMember(member string) (keyID string, tokenType core.TokenType) {
	switch {
	case strings.HasPrefix(member, memberPrefixWeb):
		return member[len(memberPrefixWeb):], core.TokenTypeWeb
	case strings.HasPrefix(member, memberPrefixTCP):
		return member[len(memberPrefixTCP):], core.TokenTypeTCP
	default:
		// Backward compat: treat bare keyID (pre-type-support) as web.
		return member, core.TokenTypeWeb
	}
}

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

// AddAPIKey adds an API key with a token type and expiration time to the user's Redis store.
// The key is stored as a prefixed member ("w:<keyID>" or "t:<keyID>") in a sorted set.
// Returns an error if the operation fails.
func (u *User) AddAPIKey(ctx context.Context, userID string, apiKeyID string, tokenType core.TokenType, expiresIn time.Duration) error {
	redisKey := u.keyPrefix + apiKeyPrefix + userID
	member := encodeKeyMember(apiKeyID, tokenType)

	_, err := u.db.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(time.Now().Add(expiresIn - ttlOffset).Unix()),
		Member: member,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add API key: %w", err)
	}

	// If the result is 0, the member already exists — not an error.
	return nil
}

// GetAPIKeys retrieves all non-expired API key IDs for a user from the Redis store.
// Prefixes are stripped; bare legacy members are returned as-is (backward compat).
// Returns a slice of bare key IDs and an error if the operation fails.
func (u *User) GetAPIKeys(ctx context.Context, userID string) ([]string, error) {
	redisKey := u.keyPrefix + apiKeyPrefix + userID

	// Clean up expired keys.
	now := time.Now().Unix()
	_, err := u.db.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to remove expired API keys: %w", err)
	}

	// Get keys with scores greater than current time (not expired).
	members, err := u.db.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     redisKey,
		ByScore: true,
		Start:   fmt.Sprintf("%d", now),
		Stop:    "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	keys := make([]string, len(members))
	for i, m := range members {
		keyID, _ := decodeKeyMember(m)
		keys[i] = keyID
	}

	return keys, nil
}

// GetAPIKeysWithExpiration retrieves all active API keys for a user along with their expiration times
// and token types. Returns a slice of KeyInfo or an error if the operation fails.
func (u *User) GetAPIKeysWithExpiration(ctx context.Context, userID string) ([]core.KeyInfo, error) {
	redisKey := u.keyPrefix + apiKeyPrefix + userID

	now := time.Now().Unix()

	// Clean up expired keys.
	_, err := u.db.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to remove expired API keys: %w", err)
	}

	// Get keys with scores greater than current time, including scores.
	zSlice, err := u.db.ZRangeByScoreWithScores(ctx, redisKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", now),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys with scores: %w", err)
	}

	keys := make([]core.KeyInfo, len(zSlice))
	for i, z := range zSlice {
		// Score is (expiresAt - ttlOffset), so restore the original expiration.
		expiresAt := time.Unix(int64(z.Score), 0).Add(ttlOffset)
		keyID, tokenType := decodeKeyMember(z.Member.(string))
		keys[i] = core.KeyInfo{
			KeyID:     keyID,
			ExpiresAt: expiresAt,
			Type:      tokenType,
		}
	}

	return keys, nil
}

// RevokeToken removes the specified API key for a user from the Redis store.
// It handles both prefixed members (new format) and bare members (legacy format).
// Returns an error if the operation fails.
func (u *User) RevokeToken(ctx context.Context, userID string, apiKeyID string) error {
	redisKey := u.keyPrefix + apiKeyPrefix + userID

	// Try all possible encodings: prefixed web, prefixed TCP, and bare (legacy).
	candidates := []string{
		encodeKeyMember(apiKeyID, core.TokenTypeWeb),
		encodeKeyMember(apiKeyID, core.TokenTypeTCP),
		apiKeyID, // legacy bare member
	}

	for _, candidate := range candidates {
		removed, err := u.db.ZRem(ctx, redisKey, candidate).Result()
		if err != nil {
			return fmt.Errorf("failed to revoke API key: %w", err)
		}

		if removed > 0 {
			return nil
		}
	}

	// Member not found — not an error; it may have already expired or been removed.
	return nil
}

// SaveConversation stores a conversation object in the Redis database. Returns an error if the operation fails.
func (u *User) SaveConversation(ctx context.Context, conversation *conv.Conversation) error {
	redisKey := u.keyPrefix + convKeyPrefix + conversation.ID

	data, err := json.Marshal(conversation)
	if err != nil {
		return fmt.Errorf("failed to encode conversation: %w", err)
	}

	_, err = u.db.Set(ctx, redisKey, data, convTTL).Result()

	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	return nil
}

// GetConversation retrieves a conversation by its ID from the Redis store. Returns the conversation or an error if it fails.
func (u *User) GetConversation(ctx context.Context, conversationID string) (*conv.Conversation, error) {
	redisKey := u.keyPrefix + convKeyPrefix + conversationID

	data, err := u.db.Get(ctx, redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			return conv.New(conversationID), nil
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	var conversation conv.Conversation
	if err := json.Unmarshal([]byte(data), &conversation); err != nil {
		return nil, fmt.Errorf("failed to decode conversation: %w", err)
	}

	return &conversation, nil
}

// DeleteConversation removes a conversation from the Redis store by its ID.
func (u *User) DeleteConversation(ctx context.Context, conversationID string) error {
	redisKey := u.keyPrefix + convKeyPrefix + conversationID

	res := u.db.Del(ctx, redisKey)
	if res.Err() != nil {
		return fmt.Errorf("failed to delete conversation: %w", res.Err())
	}

	return nil
}
