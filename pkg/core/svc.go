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
}

type MITProv interface {
	GenerateToken() (*APIToken, error)
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
