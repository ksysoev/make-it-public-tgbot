package core

import (
	"context"
	"fmt"
	"time"
)

var (
	ErrMaxTokensExceeded = fmt.Errorf("maximum tokens exceeded")
)

type UserRepo interface {
	AddAPIKey(ctx context.Context, userID string, apiKeyID string, expiresIn time.Duration) error
	GetAPIKeys(ctx context.Context, userID string) ([]string, error)
	RevokeToken(ctx context.Context, userID string, apiKeyID string) error
}

type MITProv interface {
	GenerateToken() (*APIToken, error)
	RevokeToken(keyID string) error
}

type Service struct {
	repo UserRepo
	prov MITProv
}

// New initializes and returns a new Service instance with the provided UserRepo and MITProv.
func New(repo UserRepo, prov MITProv) *Service {
	return &Service{
		repo: repo,
		prov: prov,
	}
}

// CreateToken generates a new API token for the specified user, storing it in the repository, if token limits are not exceeded.
// Returns an error if the token limit is reached, fails to generate the token, or fails to save the token in the repository.
func (s *Service) CreateToken(ctx context.Context, userID string) (*APIToken, error) {
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) > 0 {
		return nil, ErrMaxTokensExceeded
	}

	token, err := s.prov.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	if err = s.repo.AddAPIKey(ctx, userID, token.KeyID, token.ExpiresIn); err != nil {
		return nil, fmt.Errorf("failed to add API key: %w", err)
	}

	return token, nil
}

// RevokeToken revokes a user's single existing API token, removing it from both the provider and the repository.
// Returns an error if multiple or no tokens exist, or if any step in the revocation process fails.
func (s *Service) RevokeToken(ctx context.Context, userID string) error {
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) == 0 {
		return fmt.Errorf("no API keys found for user %s", userID)
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
