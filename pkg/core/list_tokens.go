package core

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	listTokensHeader = "🔑 Your Active API Tokens (Web: %d/%d, TCP: %d/%d)\n\n"
	listTokensEntry  = "%d. [%s] %s...\n   ⏱ Expires: %s\n"
	listTokensFooter = "\nUse /new_token to create a new token or /revoke_token to revoke one."
	listTokensKeyLen = 12 // number of key ID characters shown in the listing
)

// ListTokens retrieves and formats all active API tokens for the specified user.
// Returns ErrTokenNotFound if the user has no active tokens.
func (s *Service) ListTokens(ctx context.Context, userID string) (*Response, error) {
	keys, err := s.repo.GetAPIKeysWithExpiration(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, ErrTokenNotFound
	}

	var webCount, tcpCount int

	for _, k := range keys {
		if k.Type == TokenTypeTCP {
			tcpCount++
		} else {
			webCount++
		}
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, listTokensHeader, webCount, maxWebTokensPerUser, tcpCount, maxTCPTokensPerUser)

	for i, k := range keys {
		keyDisplay := k.KeyID
		if len(keyDisplay) > listTokensKeyLen {
			keyDisplay = keyDisplay[:listTokensKeyLen]
		}

		expiresAt := k.ExpiresAt.Format(time.DateTime)
		fmt.Fprintf(&sb, listTokensEntry, i+1, string(k.Type), keyDisplay, expiresAt)
	}

	sb.WriteString(listTokensFooter)

	return &Response{
		Message: sb.String(),
	}, nil
}
