package core

import (
	"context"
	"fmt"
)

// RevokeToken revokes a user's single existing API token, removing it from both the provider and the repository.
// Returns an error if multiple or no tokens exist, or if any step in the revocation process fails.
func (s *Service) RevokeToken(ctx context.Context, userID string) error {
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) == 0 {
		return ErrTokenNotFound
	}

	if len(keys) > 1 {
		return fmt.Errorf("multiple API keys found for user %s, cannot revoke", userID)
	}

	keyID := keys[0]
	if err := s.prov.RevokeToken(keyID); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if err := s.repo.RevokeToken(ctx, userID, keyID); err != nil {
		return fmt.Errorf("failed to remove API key from repository: %w", err)
	}

	return nil
}
