package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateToken(t *testing.T) {
	tests := []struct {
		getConvErr      error
		saveConvErr     error
		name            string
		userID          string
		expectedMsg     string
		expectedErr     string
		expectedAnswers []string
	}{
		{
			name:            "asks for token type selection",
			userID:          "user123",
			expectedMsg:     "What type of token do you want to create?",
			expectedAnswers: []string{"Web", "TCP"},
		},
		{
			name:        "get conversation error",
			userID:      "user123",
			getConvErr:  errors.New("redis error"),
			expectedErr: "failed to get conversation: redis error",
		},
		{
			name:        "save conversation error",
			userID:      "user123",
			saveConvErr: errors.New("save error"),
			expectedErr: "failed to save conversation: save error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			if tt.getConvErr != nil {
				repo.On("GetConversation", mock.Anything, tt.userID).Return(nil, tt.getConvErr)
			} else {
				repo.On("GetConversation", mock.Anything, tt.userID).Return(conv.New(tt.userID), nil)
				repo.On("SaveConversation", mock.Anything, mock.AnythingOfType("*conv.Conversation")).Return(tt.saveConvErr)
			}

			svc := New(repo, prov)

			resp, err := svc.CreateToken(context.Background(), tt.userID)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedMsg, resp.Message)
				assert.Equal(t, tt.expectedAnswers, resp.Answers)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}

func TestHandleSelectTokenTypeResult(t *testing.T) {
	userID := "user123"

	tests := []struct {
		getKeysErr      error
		saveConvErr     error
		name            string
		answer          string
		expectedMsg     string
		expectedErr     string
		existingKeys    []KeyInfo
		expectedAnswers []string
		expectSaveConv  bool
	}{
		{
			name:            "web under limit - asks for expiration",
			answer:          "Web",
			existingKeys:    []KeyInfo{},
			expectedMsg:     "What is the expiration period for your new API token?",
			expectedAnswers: []string{"1 day", "7 days", "30 days", "90 days"},
			expectSaveConv:  true,
		},
		{
			name:            "TCP under limit - asks for expiration",
			answer:          "TCP",
			existingKeys:    []KeyInfo{},
			expectedMsg:     "What is the expiration period for your new API token?",
			expectedAnswers: []string{"1 day", "7 days", "30 days", "90 days"},
			expectSaveConv:  true,
		},
		{
			name:   "web at limit - asks to regenerate",
			answer: "Web",
			existingKeys: []KeyInfo{
				{KeyID: "key1", Type: TokenTypeWeb, ExpiresAt: time.Now().Add(24 * time.Hour)},
				{KeyID: "key2", Type: TokenTypeWeb, ExpiresAt: time.Now().Add(24 * time.Hour)},
				{KeyID: "key3", Type: TokenTypeWeb, ExpiresAt: time.Now().Add(24 * time.Hour)},
			},
			expectedMsg:     "You've reached the maximum of 3 web tokens. Do you want to regenerate an existing one?",
			expectedAnswers: []string{"Yes", "No"},
			expectSaveConv:  true,
		},
		{
			name:   "TCP at limit - asks to regenerate",
			answer: "TCP",
			existingKeys: []KeyInfo{
				{KeyID: "tcpkey1", Type: TokenTypeTCP, ExpiresAt: time.Now().Add(24 * time.Hour)},
			},
			expectedMsg:     "You've reached the maximum of 1 TCP token. Do you want to regenerate it?",
			expectedAnswers: []string{"Yes", "No"},
			expectSaveConv:  true,
		},
		{
			name:         "invalid type selection",
			answer:       "FTP",
			existingKeys: []KeyInfo{},
			expectedMsg:  "Invalid token type selected. Please choose Web or TCP.",
		},
		{
			name:        "wrong number of answers",
			answer:      "",
			expectedErr: "expected exactly one answer for token type question, got 0",
		},
		{
			name:        "get keys error",
			answer:      "Web",
			getKeysErr:  errors.New("redis error"),
			expectedErr: "failed to get API keys: redis error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			var answers []conv.QuestionAnswer
			if tt.answer != "" {
				answers = []conv.QuestionAnswer{{Answer: tt.answer}}
			}

			if tt.answer != "" && tt.answer != "FTP" && tt.expectedErr == "" {
				repo.On("GetAPIKeysWithExpiration", mock.Anything, userID).Return(tt.existingKeys, tt.getKeysErr)
			} else if tt.getKeysErr != nil {
				repo.On("GetAPIKeysWithExpiration", mock.Anything, userID).Return(nil, tt.getKeysErr)
			}

			if tt.expectSaveConv {
				repo.On("GetConversation", mock.Anything, userID).Return(conv.New(userID), nil)
				repo.On("SaveConversation", mock.Anything, mock.AnythingOfType("*conv.Conversation")).Return(tt.saveConvErr)
			}

			svc := New(repo, prov)

			resp, err := svc.handleSelectTokenTypeResult(context.Background(), userID, answers)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedMsg, resp.Message)
				if tt.expectedAnswers != nil {
					assert.Equal(t, tt.expectedAnswers, resp.Answers)
				}
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}

func TestHandleTokenExistsResult(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		expectedMsg     string
		expectedErr     string
		answers         []conv.QuestionAnswer
		expectedAnswers []string
	}{
		{
			name:   "answer is No",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{Answer: "No"},
			},
			expectedMsg: "No changes made. You can continue using your existing API tokens.",
		},
		{
			name:   "invalid number of answers",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{Answer: "Yes"},
				{Answer: "No"},
			},
			expectedErr: "expected exactly one answer for tokenExists question, got 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			svc := New(repo, prov)

			resp, err := svc.handleTokenExistsResult(context.Background(), tt.userID, tt.answers)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedMsg, resp.Message)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}

func TestHandleTokenExistsResult_Yes(t *testing.T) {
	userID := "user123"
	keys := []KeyInfo{
		{KeyID: "abcdef123456", Type: TokenTypeWeb, ExpiresAt: time.Now().Add(24 * time.Hour)},
		{KeyID: "xyz987654321", Type: TokenTypeWeb, ExpiresAt: time.Now().Add(48 * time.Hour)},
	}

	repo := NewMockUserRepo(t)
	prov := NewMockMITProv(t)

	repo.On("GetAPIKeysWithExpiration", mock.Anything, userID).Return(keys, nil)
	repo.On("GetConversation", mock.Anything, userID).Return(conv.New(userID), nil)
	repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
		return c.ID == userID && c.State == StateSelectTokenToRegenerate
	})).Return(nil)

	svc := New(repo, prov)

	answers := []conv.QuestionAnswer{
		{Answer: "Yes", Field: string(TokenTypeWeb)},
	}

	resp, err := svc.handleTokenExistsResult(context.Background(), userID, answers)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Which token do you want to regenerate?", resp.Message)
	assert.Len(t, resp.Answers, 2)

	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

func TestHandleTokenExistsResult_Yes_TCP(t *testing.T) {
	userID := "user123"
	keys := []KeyInfo{
		{KeyID: "tcpkey123456", Type: TokenTypeTCP, ExpiresAt: time.Now().Add(24 * time.Hour)},
	}

	repo := NewMockUserRepo(t)
	prov := NewMockMITProv(t)

	repo.On("GetAPIKeysWithExpiration", mock.Anything, userID).Return(keys, nil)
	// TCP with 1 token at limit: skips selection, goes directly to expiration
	repo.On("GetConversation", mock.Anything, userID).Return(conv.New(userID), nil)
	repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
		return c.ID == userID && c.State == StateTokenRegenerate
	})).Return(nil)

	svc := New(repo, prov)

	answers := []conv.QuestionAnswer{
		{Answer: "Yes", Field: string(TokenTypeTCP)},
	}

	resp, err := svc.handleTokenExistsResult(context.Background(), userID, answers)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "What is the expiration period for your new API token?", resp.Message)
	assert.Equal(t, []string{"1 day", "7 days", "30 days", "90 days"}, resp.Answers)

	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

func TestHandleNewTokenResult(t *testing.T) {
	tests := []struct {
		generateErr error
		addKeyErr   error
		token       *APIToken
		name        string
		userID      string
		expectedMsg string
		expectedErr string
		answers     []conv.QuestionAnswer
	}{
		{
			name:   "success - 7 days",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{Answer: "7 days"},
			},
			token: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: 7 * 24 * time.Hour,
			},
			expectedMsg: "Your New API Token",
		},
		{
			name:   "invalid expiration period",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{Answer: "invalid"},
			},
			expectedMsg: "Invalid expiration period selected.",
		},
		{
			name:        "empty answers",
			userID:      "user123",
			answers:     []conv.QuestionAnswer{},
			expectedErr: "expected at least one answer for expiration question, got 0",
		},
		{
			name:   "generate token error",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{Answer: "1 day"},
			},
			generateErr: errors.New("generate error"),
			expectedErr: "failed to generate token: generate error",
		},
		{
			name:   "add key error",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{Answer: "30 days"},
			},
			token: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: 30 * 24 * time.Hour,
			},
			addKeyErr:   errors.New("add key error"),
			expectedErr: "failed to add API key: add key error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			if tt.token != nil || tt.generateErr != nil {
				prov.On("GenerateToken", "", TokenTypeWeb, mock.AnythingOfType("int64")).Return(tt.token, tt.generateErr)
			}

			if tt.token != nil && tt.generateErr == nil {
				repo.On("AddAPIKey", mock.Anything, tt.userID, tt.token.KeyID, TokenTypeWeb, tt.token.ExpiresIn).Return(tt.addKeyErr)
			}

			svc := New(repo, prov)

			resp, err := svc.handleNewTokenResult(context.Background(), tt.userID, tt.answers)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Contains(t, resp.Message, tt.expectedMsg)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}

func TestHandleTokenRegenerateResult(t *testing.T) {
	userID := "user123"
	keyID := "key123"

	t.Run("success", func(t *testing.T) {
		repo := NewMockUserRepo(t)
		prov := NewMockMITProv(t)

		token := &APIToken{
			KeyID:     keyID,
			Token:     "newtoken",
			ExpiresIn: 7 * 24 * time.Hour,
		}

		prov.On("RevokeToken", keyID).Return(nil)
		repo.On("RevokeToken", mock.Anything, userID, keyID).Return(nil)
		prov.On("GenerateToken", keyID, TokenTypeWeb, mock.AnythingOfType("int64")).Return(token, nil)
		repo.On("AddAPIKey", mock.Anything, userID, keyID, TokenTypeWeb, token.ExpiresIn).Return(nil)

		svc := New(repo, prov)

		answers := []conv.QuestionAnswer{
			{Answer: "7 days", Field: encodeTokenField(TokenTypeWeb, keyID)},
		}

		resp, err := svc.handleTokenRegenerateResult(context.Background(), userID, answers)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Your New API Token")

		repo.AssertExpectations(t)
		prov.AssertExpectations(t)
	})

	t.Run("missing key ID in field", func(t *testing.T) {
		svc := New(NewMockUserRepo(t), NewMockMITProv(t))

		answers := []conv.QuestionAnswer{
			{Answer: "7 days", Field: encodeTokenField(TokenTypeWeb, "")},
		}

		_, err := svc.handleTokenRegenerateResult(context.Background(), userID, answers)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing key ID")
	})

	t.Run("invalid expiration period", func(t *testing.T) {
		svc := New(NewMockUserRepo(t), NewMockMITProv(t))

		answers := []conv.QuestionAnswer{
			{Answer: "invalid", Field: encodeTokenField(TokenTypeWeb, keyID)},
		}

		resp, err := svc.handleTokenRegenerateResult(context.Background(), userID, answers)
		require.NoError(t, err)
		assert.Contains(t, resp.Message, "Invalid expiration period")
	})
}

func TestHandleSelectTokenToRegenerateResult(t *testing.T) {
	userID := "user123"
	keyID := "abcdef123456"

	t.Run("success", func(t *testing.T) {
		repo := NewMockUserRepo(t)
		prov := NewMockMITProv(t)

		repo.On("GetAPIKeysWithExpiration", mock.Anything, userID).Return([]KeyInfo{
			{KeyID: keyID, Type: TokenTypeWeb, ExpiresAt: time.Now().Add(24 * time.Hour)},
		}, nil)
		repo.On("GetConversation", mock.Anything, userID).Return(conv.New(userID), nil)
		repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
			return c.State == StateTokenRegenerate
		})).Return(nil)

		svc := New(repo, prov)

		answers := []conv.QuestionAnswer{
			{Answer: keyID[:keyIDDisplayLen] + " (exp: 2026-03-01)", Field: string(TokenTypeWeb)},
		}

		resp, err := svc.handleSelectTokenToRegenerateResult(context.Background(), userID, answers)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "What is the expiration period for your new API token?", resp.Message)

		repo.AssertExpectations(t)
		prov.AssertExpectations(t)
	})

	t.Run("wrong number of answers", func(t *testing.T) {
		svc := New(NewMockUserRepo(t), NewMockMITProv(t))
		_, err := svc.handleSelectTokenToRegenerateResult(context.Background(), userID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected exactly one answer")
	})
}

func TestBuildTokenSelectionQuestion(t *testing.T) {
	keys := []KeyInfo{
		{KeyID: "abcdef12345678", ExpiresAt: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)},
		{KeyID: "xyz9876543210", ExpiresAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)},
	}

	q := buildTokenSelectionQuestion(keys, "Pick one")

	assert.Equal(t, "Pick one", q.Text)
	assert.Len(t, q.Answers, 2)
	assert.Contains(t, q.Answers[0], "abcdef12")
	assert.Contains(t, q.Answers[0], "2026-03-15")
	assert.Contains(t, q.Answers[1], "xyz98765")
	assert.Contains(t, q.Answers[1], "2026-04-20")
}

func TestResolveKeyIDFromPrefix(t *testing.T) {
	keys := []string{"abcdef12345678", "xyz9876543210"}

	t.Run("match first key", func(t *testing.T) {
		result, err := resolveKeyIDFromPrefix(keys, "abcdef12 (exp: 2026-03-15)")
		require.NoError(t, err)
		assert.Equal(t, "abcdef12345678", result)
	})

	t.Run("match second key", func(t *testing.T) {
		result, err := resolveKeyIDFromPrefix(keys, "xyz98765 (exp: 2026-04-20)")
		require.NoError(t, err)
		assert.Equal(t, "xyz9876543210", result)
	})

	t.Run("no match", func(t *testing.T) {
		_, err := resolveKeyIDFromPrefix(keys, "nomatch1 (exp: 2026-03-15)")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("short button text", func(t *testing.T) {
		_, err := resolveKeyIDFromPrefix(keys, "short")
		require.Error(t, err)
	})
}
