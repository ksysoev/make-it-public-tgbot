package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
)

var (
	ErrTokenNotFound = fmt.Errorf("token not found")
)

// UserRepo defines the storage operations required by the core service.
type UserRepo interface {
	AddAPIKey(ctx context.Context, userID string, apiKeyID string, tokenType TokenType, expiresIn time.Duration) error
	GetAPIKeys(ctx context.Context, userID string) ([]string, error)
	GetAPIKeysWithExpiration(ctx context.Context, userID string) ([]KeyInfo, error)
	RevokeToken(ctx context.Context, userID string, apiKeyID string) error
	SaveConversation(ctx context.Context, conversation *conv.Conversation) error
	GetConversation(ctx context.Context, conversationID string) (*conv.Conversation, error)
	DeleteConversation(ctx context.Context, conversationID string) error
}

// MITProv defines the external API operations for managing tokens.
// GenerateToken creates a new token of the given type; RevokeToken removes it.
type MITProv interface {
	GenerateToken(keyID string, tokenType TokenType, ttl int64) (*APIToken, error)
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

// ResetConversation deletes the conversation associated with a user.
func (s *Service) ResetConversation(ctx context.Context, userID string) error {
	if err := s.repo.DeleteConversation(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

// HandleMessage processes an incoming user message within a conversation context and returns a response or an error.
func (s *Service) HandleMessage(ctx context.Context, userID string, message string) (*Response, error) {
	cnv, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	state, err := cnv.Submit(message)
	if err != nil {
		return nil, fmt.Errorf("failed to submit message: %w", err)
	}

	res, err := cnv.Results()

	switch {
	case errors.Is(err, conv.ErrIsNotComplete):
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
	case err != nil:
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	if err := s.repo.SaveConversation(ctx, cnv); err != nil {
		return nil, fmt.Errorf("failed to save conversation: %w", err)
	}

	switch state {
	case StateSelectTokenType:
		return s.handleSelectTokenTypeResult(ctx, userID, res)
	case StateTokenExists:
		return s.handleTokenExistsResult(ctx, userID, res)
	case StateNewToken:
		return s.handleNewTokenResult(ctx, userID, res)
	case StateTokenRegenerate:
		return s.handleTokenRegenerateResult(ctx, userID, res)
	case StateSelectTokenToRegenerate:
		return s.handleSelectTokenToRegenerateResult(ctx, userID, res)
	case StateSelectTokenToRevoke:
		return s.handleSelectTokenToRevokeResult(ctx, userID, res)
	default:
		return nil, fmt.Errorf("unsupported conversation state: %s", state)
	}
}
