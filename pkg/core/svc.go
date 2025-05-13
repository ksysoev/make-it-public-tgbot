package core

import (
	"context"
	"fmt"
	"time"
)

type UserRepo interface {
	AddAPIKey(ctx context.Context, userID string, apiKeyID string, expiresIn time.Duration) error
}

type MITProv interface {
	GenerateToken() (*APIToken, error)
}

type Service struct {
	repo UserRepo
	prov MITProv
}

func New(repo UserRepo, prov MITProv) *Service {
	return &Service{
		repo: repo,
		prov: prov,
	}
}

func (s *Service) CreateToken(_ context.Context, _ string) (*APIToken, error) {
	return nil, fmt.Errorf("not implemented")
}
