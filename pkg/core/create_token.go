package core

import (
	"context"
	"fmt"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
)

const (
	maxTokensPerUser    = 1
	secondsInDay        = 24 * 60 * 60
	tokenCreatedMessage = "🔑 Your New API Token\n\n%s\n\n⏱ Valid until: %s\n\nKeep this token secure and don't share it with others."
)

const (
	StateTokenRegenerate conv.State = "tokenRegenerate"
	StateTokenExists     conv.State = "tokenExists"
	StateNewToken        conv.State = "newToken"
)

// CreateToken generates a new API token for the specified user, storing it in the repository, if token limits are not exceeded.
// Returns an error if the token limit is reached, fails to generate the token, or fails to save the token in the repository.
func (s *Service) CreateToken(ctx context.Context, userID string) (*Response, error) {
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if len(keys) >= maxTokensPerUser {
		questions := conv.NewQuestions(
			[]conv.Question{{
				Text:    "You already have an active API token. Do you want to regenerate it?",
				Answers: []string{"Yes", "No"},
			}},
		)

		if err := c.Start(StateTokenExists, questions); err != nil {
			return nil, fmt.Errorf("failed to start questions: %w", err)
		}

		q, _ := c.Current()

		if err := s.repo.SaveConversation(ctx, c); err != nil {
			return nil, fmt.Errorf("failed to save conversation: %w", err)
		}

		return &Response{
			Message: q.Text,
			Answers: q.Answers,
		}, nil
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    "What is the expiration period for your new API token?",
			Answers: []string{"1 day", "7 days", "30 days", "90 days"},
		}},
	)

	if err := c.Start(StateNewToken, questions); err != nil {
		return nil, fmt.Errorf("failed to start questions: %w", err)
	}

	q, _ := c.Current()
	if err := s.repo.SaveConversation(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to save conversation: %w", err)
	}

	return &Response{
		Message: q.Text,
		Answers: q.Answers,
	}, nil
}

// handleTokenExistsResult processes the result of a "token exists" question and takes appropriate action based on the answer.
func (s *Service) handleTokenExistsResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for tokenExists question, got %d", len(answers))
	}

	if answers[0].Answer == "No" {
		return &Response{
			Message: "No changes made. You can continue using your existing API token.",
		}, nil
	}

	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    "What is the expiration period for your new API token?",
			Answers: []string{"1 day", "7 days", "30 days", "90 days"},
		}},
	)

	if err := c.Start(StateTokenRegenerate, questions); err != nil {
		return nil, fmt.Errorf("failed to start questions: %w", err)
	}

	q, _ := c.Current()
	if err := s.repo.SaveConversation(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to save conversation: %w", err)
	}

	return &Response{
		Message: q.Text,
		Answers: q.Answers,
	}, nil
}

func (s *Service) handleNewTokenResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for newToken question, got %d", len(answers))
	}

	var expiresIn int64
	switch answers[0].Answer {
	case "1 day":
		expiresIn = secondsInDay
	case "7 days":
		expiresIn = 7 * secondsInDay
	case "30 days":
		expiresIn = 30 * secondsInDay
	case "90 days":
		expiresIn = 90 * secondsInDay
	default:
		return &Response{
			Message: "Invalid expiration period selected. Please select one of the available options.",
		}, nil
	}

	token, err := s.prov.GenerateToken("", expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	if err = s.repo.AddAPIKey(ctx, userID, token.KeyID, token.ExpiresIn); err != nil {
		return nil, fmt.Errorf("failed to add API key: %w", err)
	}

	expiresAt := time.Now().Add(token.ExpiresIn).Format(time.DateTime)
	return &Response{
		Message: fmt.Sprintf(tokenCreatedMessage, token.Token, expiresAt),
	}, nil
}

func (s *Service) handleTokenRegenerateResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for newToken question, got %d", len(answers))
	}

	var expiresIn int64
	switch answers[0].Answer {
	case "1 day":
		expiresIn = secondsInDay
	case "7 days":
		expiresIn = 7 * secondsInDay
	case "30 days":
		expiresIn = 30 * secondsInDay
	case "90 days":
		expiresIn = 90 * secondsInDay
	default:
		return &Response{
			Message: "Invalid expiration period selected. Please select one of the available options.",
		}, nil
	}

	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) != 1 {
		return nil, fmt.Errorf("expected exactly one API key for user %s, got %d", userID, len(keys))
	}

	keyID := keys[0]
	if err := s.prov.RevokeToken(keyID); err != nil {
		return nil, fmt.Errorf("failed to revoke existing token: %w", err)
	}

	if err := s.repo.RevokeToken(ctx, userID, keyID); err != nil {
		return nil, fmt.Errorf("failed to remove API key from repository: %w", err)
	}

	token, err := s.prov.GenerateToken(keyID, expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	if err = s.repo.AddAPIKey(ctx, userID, token.KeyID, token.ExpiresIn); err != nil {
		return nil, fmt.Errorf("failed to add API key: %w", err)
	}

	expiresAt := time.Now().Add(token.ExpiresIn).Format(time.DateTime)
	return &Response{
		Message: fmt.Sprintf(tokenCreatedMessage, token.Token, expiresAt),
	}, nil
}
