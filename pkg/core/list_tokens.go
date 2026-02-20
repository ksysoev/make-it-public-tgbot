package core

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	listTokensHeader = "🔑 Your Active API Tokens (%d/%d)\n\n"
	listTokensEntry  = "%d. %s...\n   ⏱ Expires: %s\n"
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

	var sb strings.Builder

	fmt.Fprintf(&sb, listTokensHeader, len(keys), maxTokensPerUser)

	for i, k := range keys {
		keyDisplay := k.KeyID
		if len(keyDisplay) > listTokensKeyLen {
			keyDisplay = keyDisplay[:listTokensKeyLen]
		}

		expiresAt := k.ExpiresAt.Format(time.DateTime)
		fmt.Fprintf(&sb, listTokensEntry, i+1, keyDisplay, expiresAt)
	}

	sb.WriteString(listTokensFooter)

	return &Response{
		Message: sb.String(),
	}, nil
}
