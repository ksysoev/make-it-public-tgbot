package core

import (
	"context"
	"fmt"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
)

var (
	ErrMaxTokensExceeded = fmt.Errorf("maximum tokens exceeded")
	ErrTokenNotFound     = fmt.Errorf("token not found")
)

type UserRepo interface {
	AddAPIKey(ctx context.Context, userID string, apiKeyID string, expiresIn time.Duration) error
	GetAPIKeys(ctx context.Context, userID string) ([]string, error)
	RevokeToken(ctx context.Context, userID string, apiKeyID string) error
	SaveConversation(ctx context.Context, conversation *conv.Conversation) error
	GetConversation(ctx context.Context, conversationID string) (*conv.Conversation, error)
}

type MITProv interface {
	GenerateToken() (*APIToken, error)
	RevokeToken(keyID string) error
}

type Response struct {
	Message string   `json:"message"` // Main response message
	Answers []string `json:"answers"` // Possible answers for the follow-up question
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

func (s *Service) HandleMessage(ctx context.Context, userID string, message string) (*Response, error) {
	cnv, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	startState := cnv.State

	err = cnv.Submit(message)
	if err != nil {
		return nil, fmt.Errorf("failed to submit message: %w", err)


	if cnv.State != conv.StateComplete {
		q, err := cnv.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get current question: %w", err)
		}

		if err := s.repo.SaveConversation(ctx, cnv); err != nil {
			return nil, fmt.Errorf("failed to save conversation: %w", err)
		}

		return &Response{
			Message: q.Text,
			Answers: q.Answers,
		}, nil
	}

	switch startState {
	case StateTokenExists:
		s.
	}



}
