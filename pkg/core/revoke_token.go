package core

import (
	"context"
	"fmt"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
)

// RevokeToken revokes a user's API token.
// If the user has exactly one token, it is revoked directly and nil is returned for the response.
// If the user has multiple tokens, a conversation is started to ask which token to revoke.
// Returns an error if no tokens exist or if any step in the process fails.
func (s *Service) RevokeToken(ctx context.Context, userID string) (*Response, error) {
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, ErrTokenNotFound
	}

	if len(keys) == 1 {
		if err := s.revokeKeyByID(ctx, userID, keys[0]); err != nil {
			return nil, err
		}

		return nil, nil //nolint:nilnil // nil response signals direct revocation with no follow-up conversation
	}

	return s.askToSelectTokenForRevocation(ctx, userID)
}

// askToSelectTokenForRevocation starts a conversation asking the user which of their tokens to revoke.
func (s *Service) askToSelectTokenForRevocation(ctx context.Context, userID string) (*Response, error) {
	keys, err := s.repo.GetAPIKeysWithExpiration(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	q := buildTokenSelectionQuestion(keys, "Which token do you want to revoke?")

	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions([]conv.Question{q})

	if err := c.Start(StateSelectTokenToRevoke, questions); err != nil {
		return nil, fmt.Errorf("failed to start questions: %w", err)
	}

	current, _ := c.Current()

	if err := s.repo.SaveConversation(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to save conversation: %w", err)
	}

	return &Response{
		Message: current.Text,
		Answers: current.Answers,
	}, nil
}

// handleSelectTokenToRevokeResult processes the user's token selection and performs the revocation.
func (s *Service) handleSelectTokenToRevokeResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for token selection question, got %d", len(answers))
	}

	selectedPrefix := answers[0].Answer

	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	keyID, err := resolveKeyIDFromPrefix(keys, selectedPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve key ID: %w", err)
	}

	if err := s.revokeKeyByID(ctx, userID, keyID); err != nil {
		return nil, err
	}

	return &Response{
		Message: "🔒 Your API token has been successfully revoked.\n\nYou can create a new one using /new_token command.",
	}, nil
}

// revokeKeyByID revokes the given key ID from both the provider and the repository.
func (s *Service) revokeKeyByID(ctx context.Context, userID string, keyID string) error {
	if err := s.prov.RevokeToken(keyID); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if err := s.repo.RevokeToken(ctx, userID, keyID); err != nil {
		return fmt.Errorf("failed to remove API key from repository: %w", err)
	}

	return nil
}
