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

func (s *Service) CreateToken(_ context.Context, _ string) error {
	return fmt.Errorf("not implemented")
}
