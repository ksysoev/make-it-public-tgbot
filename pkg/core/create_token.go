package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
)

const (
	maxWebTokensPerUser = 3
	maxTCPTokensPerUser = 1
	secondsInDay        = 24 * 60 * 60
	tokenCreatedMessage = "🔑 Your New API Token\n\n%s\n\n⏱ Valid until: %s\n\nKeep this token secure and don't share it with others."
	keyIDDisplayLen     = 8   // Number of characters shown from key ID in buttons
	tokenFieldSep       = "|" // Separator between token type and key ID in conv.Question.Field
)

const (
	StateSelectTokenType         conv.State = "selectTokenType"
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

// encodeTokenField encodes a token type and key ID into a single Field string.
// For new tokens where keyID is empty, the field is "<type>|".
func encodeTokenField(tokenType TokenType, keyID string) string {
	return string(tokenType) + tokenFieldSep + keyID
}

// decodeTokenField decodes a Field string produced by encodeTokenField back into a TokenType and key ID.
// If the field has no separator, it returns TokenTypeWeb and the whole string as keyID (backward compat).
func decodeTokenField(field string) (TokenType, string) {
	idx := strings.Index(field, tokenFieldSep)
	if idx < 0 {
		// Backward compat: treat as web with the whole field as keyID.
		return TokenTypeWeb, field
	}

	return TokenType(field[:idx]), field[idx+len(tokenFieldSep):]
}

// maxTokensForType returns the per-user token limit for the given type.
func maxTokensForType(tokenType TokenType) int {
	if tokenType == TokenTypeTCP {
		return maxTCPTokensPerUser
	}

	return maxWebTokensPerUser
}

// filterKeysByType returns only the KeyInfo entries matching the given token type.
func filterKeysByType(keys []KeyInfo, tokenType TokenType) []KeyInfo {
	result := make([]KeyInfo, 0, len(keys))
	for _, k := range keys {
		if k.Type == tokenType {
			result = append(result, k)
		}
	}

	return result
}

// CreateToken starts a conversation asking the user what type of token they want to create (Web or TCP).
func (s *Service) CreateToken(ctx context.Context, userID string) (*Response, error) {
	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    "What type of token do you want to create?",
			Answers: []string{"Web", "TCP"},
		}},
	)

	if err := c.Start(StateSelectTokenType, questions); err != nil {
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

// handleSelectTokenTypeResult processes the type selection answer and branches into the appropriate flow.
// If under the per-type limit it asks for expiration; if at the limit it asks to regenerate.
func (s *Service) handleSelectTokenTypeResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for token type question, got %d", len(answers))
	}

	var tokenType TokenType

	switch answers[0].Answer {
	case "Web":
		tokenType = TokenTypeWeb
	case "TCP":
		tokenType = TokenTypeTCP
	default:
		return &Response{
			Message: "Invalid token type selected. Please choose Web or TCP.",
		}, nil
	}

	keys, err := s.repo.GetAPIKeysWithExpiration(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	typeKeys := filterKeysByType(keys, tokenType)
	limit := maxTokensForType(tokenType)

	if len(typeKeys) >= limit {
		return s.askToRegenerateToken(ctx, userID, tokenType)
	}

	return s.askForTokenExpiration(ctx, userID, StateNewToken, tokenType)
}

// askToRegenerateToken starts a conversation asking the user whether they want to regenerate
// one of their existing tokens of the given type. The token type is encoded in the question Field.
func (s *Service) askToRegenerateToken(ctx context.Context, userID string, tokenType TokenType) (*Response, error) {
	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	limit := maxTokensForType(tokenType)

	var text string

	switch tokenType {
	case TokenTypeTCP:
		text = fmt.Sprintf("You've reached the maximum of %d TCP token. Do you want to regenerate it?", limit)
	default:
		text = fmt.Sprintf("You've reached the maximum of %d web tokens. Do you want to regenerate an existing one?", limit)
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    text,
			Answers: []string{"Yes", "No"},
			Field:   string(tokenType),
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
// The token type is extracted from the answer's Field (set by askToRegenerateToken).
func (s *Service) handleTokenExistsResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for tokenExists question, got %d", len(answers))
	}

	if answers[0].Answer == "No" {
		return &Response{
			Message: "No changes made. You can continue using your existing API tokens.",
		}, nil
	}

	tokenType := TokenType(answers[0].Field)
	if tokenType == "" {
		tokenType = TokenTypeWeb // backward compat
	}

	return s.askToSelectTokenForRegeneration(ctx, userID, tokenType)
}

// askToSelectTokenForRegeneration starts a conversation asking the user which token of the given type to regenerate.
// If there is exactly one token of that type, it skips the selection step and proceeds directly to expiration.
func (s *Service) askToSelectTokenForRegeneration(ctx context.Context, userID string, tokenType TokenType) (*Response, error) {
	keys, err := s.repo.GetAPIKeysWithExpiration(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	typeKeys := filterKeysByType(keys, tokenType)

	// If only one token of this type exists, skip selection and ask for expiration directly.
	if len(typeKeys) == 1 {
		return s.askForTokenExpirationWithKeyID(ctx, userID, StateTokenRegenerate, tokenType, typeKeys[0].KeyID)
	}

	q := buildTokenSelectionQuestion(typeKeys, "Which token do you want to regenerate?")
	q.Field = string(tokenType) // carry type forward for handleSelectTokenToRegenerateResult

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
// The token type is extracted from the answer's Field to scope the key resolution correctly.
func (s *Service) handleSelectTokenToRegenerateResult(ctx context.Context, userID string, answers []conv.QuestionAnswer) (*Response, error) {
	if len(answers) != 1 {
		return nil, fmt.Errorf("expected exactly one answer for token selection question, got %d", len(answers))
	}

	selectedPrefix := answers[0].Answer
	tokenType := TokenType(answers[0].Field)

	if tokenType == "" {
		tokenType = TokenTypeWeb // backward compat
	}

	// Resolve the full key ID from the prefix, scoped to the correct token type.
	keys, err := s.repo.GetAPIKeysWithExpiration(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	typeKeys := filterKeysByType(keys, tokenType)
	keyIDs := make([]string, len(typeKeys))

	for i, k := range typeKeys {
		keyIDs[i] = k.KeyID
	}

	keyID, err := resolveKeyIDFromPrefix(keyIDs, selectedPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve key ID: %w", err)
	}

	return s.askForTokenExpirationWithKeyID(ctx, userID, StateTokenRegenerate, tokenType, keyID)
}

// askForTokenExpirationWithKeyID starts a conversation asking the user for an expiration period.
// Both tokenType and keyID are encoded into the question's Field for retrieval by the result handler.
// Pass an empty keyID when creating a new token (as opposed to regenerating).
func (s *Service) askForTokenExpirationWithKeyID(ctx context.Context, userID string, state conv.State, tokenType TokenType, keyID string) (*Response, error) {
	c, err := s.repo.GetConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	questions := conv.NewQuestions(
		[]conv.Question{{
			Text:    "What is the expiration period for your new API token?",
			Answers: []string{"1 day", "7 days", "30 days", "90 days"},
			Field:   encodeTokenField(tokenType, keyID),
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
// The token type is decoded from the Field of the answer.
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

	if len(answers) == 0 {
		return nil, fmt.Errorf("expected at least one answer, got none")
	}

	tokenType, _ := decodeTokenField(answers[0].Field)

	token, err := s.prov.GenerateToken("", tokenType, expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	if err = s.repo.AddAPIKey(ctx, userID, token.KeyID, tokenType, token.ExpiresIn); err != nil {
		return nil, fmt.Errorf("failed to add API key: %w", err)
	}

	expiresAt := time.Now().Add(token.ExpiresIn).Format(time.DateTime)

	return &Response{
		Message: fmt.Sprintf(tokenCreatedMessage, token.Token, expiresAt),
	}, nil
}

// handleTokenRegenerateResult revokes the previously selected token and generates a new one.
// The token type and key ID to revoke are decoded from the Field of the expiration question answer.
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

	tokenType, keyID := decodeTokenField(answers[0].Field)
	if keyID == "" {
		return nil, fmt.Errorf("missing key ID in regenerate answer field")
	}

	if err := s.prov.RevokeToken(keyID); err != nil {
		return nil, fmt.Errorf("failed to revoke existing token: %w", err)
	}

	if err := s.repo.RevokeToken(ctx, userID, keyID); err != nil {
		return nil, fmt.Errorf("failed to remove API key from repository: %w", err)
	}

	token, err := s.prov.GenerateToken(keyID, tokenType, expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	if err = s.repo.AddAPIKey(ctx, userID, token.KeyID, tokenType, token.ExpiresIn); err != nil {
		return nil, fmt.Errorf("failed to add API key: %w", err)
	}

	expiresAt := time.Now().Add(token.ExpiresIn).Format(time.DateTime)

	return &Response{
		Message: fmt.Sprintf(tokenCreatedMessage, token.Token, expiresAt),
	}, nil
}

// askForTokenExpiration starts a conversation asking for token expiration for a new (non-regenerate) token.
func (s *Service) askForTokenExpiration(ctx context.Context, userID string, state conv.State, tokenType TokenType) (*Response, error) {
	return s.askForTokenExpirationWithKeyID(ctx, userID, state, tokenType, "")
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

// buildTokenSelectionQuestion creates a Question listing all provided keys as selectable buttons.
// The button text is the first keyIDDisplayLen characters of the key ID plus the expiration date.
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
