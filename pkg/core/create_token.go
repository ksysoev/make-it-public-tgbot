package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
)

const (
	maxTokensPerUser    = 3
	secondsInDay        = 24 * 60 * 60
	tokenCreatedMessage = "🔑 Your New API Token\n\n%s\n\n⏱ Valid until: %s\n\nKeep this token secure and don't share it with others."
	keyIDDisplayLen     = 8 // Number of characters shown from key ID in buttons
)

const (
	StateTokenRegenerate         conv.State = "tokenRegenerate"
	StateTokenExists             conv.State = "tokenExists"
	StateNewToken                conv.State = "newToken"
	StateSelectTokenToRegenerate conv.State = "selectTokenToRegenerate"
	StateSelectTokenToRevoke     conv.State = "selectTokenToRevoke"
)

var (
	ErrInvalidExpirationPeriod = fmt.Errorf("invalid expiration period selected")
	ErrKeyNotFound             = fmt.Errorf("selected key not found")
)

// CreateToken generates a new API token for the specified user, storing it in the repository.
// If the user is at the token limit, it starts a conversation to choose an existing token to regenerate.
// Returns an error if token fetching fails or if starting the conversation fails.
func (s *Service) CreateToken(ctx context.Context, userID string) (*Response, error) {
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) >= maxTokensPerUser {
		return s.askToRegenerateToken(ctx, userID)
	}

	return s.askForTokenExpiration(ctx, userID, StateNewToken)
}

// askToRegenerateToken starts a conversation asking the user whether they want to regenerate one of their existing tokens.
func (s *Service) askToRegenerateToken(ctx context.Context, userID string) (*Response, error) {
	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    fmt.Sprintf("You've reached the maximum of %d API tokens. Do you want to regenerate an existing one?", maxTokensPerUser),
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

// handleTokenExistsResult processes the answer to "do you want to regenerate?" and takes appropriate action.
func (s *Service) handleTokenExistsResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for tokenExists question, got %d", len(answers))
	}

	if answers[0].Answer == "No" {
		return &Response{
			Message: "No changes made. You can continue using your existing API tokens.",
		}, nil
	}

	return s.askToSelectTokenForRegeneration(ctx, userID)
}

// askToSelectTokenForRegeneration starts a conversation asking the user which token to regenerate.
func (s *Service) askToSelectTokenForRegeneration(ctx context.Context, userID string) (*Response, error) {
	keys, err := s.repo.GetAPIKeysWithExpiration(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	q := buildTokenSelectionQuestion(keys, "Which token do you want to regenerate?")

	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions([]conv.Question{q})

	if err := c.Start(StateSelectTokenToRegenerate, questions); err != nil {
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

// handleSelectTokenToRegenerateResult stores the selected key ID context and asks for the new expiration period.
func (s *Service) handleSelectTokenToRegenerateResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for token selection question, got %d", len(answers))
	}

	selectedPrefix := answers[0].Answer

	// Resolve the full key ID from the prefix
	keys, err := s.repo.GetAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	keyID, err := resolveKeyIDFromPrefix(keys, selectedPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve key ID: %w", err)
	}

	return s.askForTokenExpirationWithKeyID(ctx, userID, StateTokenRegenerate, keyID)
}

// askForTokenExpirationWithKeyID starts a conversation asking the user for an expiration period,
// embedding the target keyID in the question field for later use.
func (s *Service) askForTokenExpirationWithKeyID(ctx context.Context, userID string, state conv.State, keyID string) (*Response, error) {
	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    "What is the expiration period for your new API token?",
			Answers: []string{"1 day", "7 days", "30 days", "90 days"},
			Field:   keyID,
		}},
	)

	if err := c.Start(state, questions); err != nil {
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

// handleNewTokenResult creates a brand-new token with the chosen expiration period.
func (s *Service) handleNewTokenResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	expiresIn, err := s.parseExpirationAnswer(answers)

	switch {
	case errors.Is(err, ErrInvalidExpirationPeriod):
		return &Response{
			Message: "Invalid expiration period selected. Please select one of the available options.",
		}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to parse expiration answer: %w", err)
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

// handleTokenRegenerateResult revokes the previously selected token and generates a new one.
// The key ID to revoke is stored in the Field of the expiration question answer.
func (s *Service) handleTokenRegenerateResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	expiresIn, err := s.parseExpirationAnswer(answers)

	switch {
	case errors.Is(err, ErrInvalidExpirationPeriod):
		return &Response{
			Message: "Invalid expiration period selected. Please select one of the available options.",
		}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to parse expiration answer: %w", err)
	}

	if len(answers) == 0 {
		return nil, fmt.Errorf("expected at least one answer, got none")
	}

	keyID := answers[0].Field
	if keyID == "" {
		return nil, fmt.Errorf("missing key ID in regenerate answer field")
	}

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

// askForTokenExpiration starts a conversation asking for token expiration period (no associated key ID).
func (s *Service) askForTokenExpiration(ctx context.Context, userID string, state conv.State) (*Response, error) {
	return s.askForTokenExpirationWithKeyID(ctx, userID, state, "")
}

// parseExpirationAnswer converts the user's textual expiration answer to a seconds value.
func (s *Service) parseExpirationAnswer(answers []conv.QuestionAnswer) (int64, error) {
	if len(answers) == 0 {
		return 0, fmt.Errorf("expected at least one answer for expiration question, got 0")
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
		return 0, ErrInvalidExpirationPeriod
	}

	return expiresIn, nil
}

// buildTokenSelectionQuestion creates a Question that lists all provided keys as selectable buttons.
// The button text is the first keyIDDisplayLen characters of the key ID, plus expiration date.
func buildTokenSelectionQuestion(keys []KeyInfo, questionText string) conv.Question {
	answers := make([]string, len(keys))
	for i, k := range keys {
		prefix := k.KeyID
		if len(prefix) > keyIDDisplayLen {
			prefix = prefix[:keyIDDisplayLen]
		}

		answers[i] = fmt.Sprintf("%s (exp: %s)", prefix, k.ExpiresAt.Format("2006-01-02"))
	}

	return conv.Question{
		Text:    questionText,
		Answers: answers,
	}
}

// resolveKeyIDFromPrefix finds the full key ID whose prefix matches the button text selected by the user.
// The button text format is "<prefix> (exp: <date>)".
func resolveKeyIDFromPrefix(keys []string, buttonText string) (string, error) {
	for _, k := range keys {
		prefix := k
		if len(prefix) > keyIDDisplayLen {
			prefix = prefix[:keyIDDisplayLen]
		}

		if len(buttonText) >= keyIDDisplayLen && buttonText[:keyIDDisplayLen] == prefix {
			return k, nil
		}
	}

	return "", ErrKeyNotFound
}
