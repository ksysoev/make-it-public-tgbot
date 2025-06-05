package core

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateToken(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		existingKeys   []string
		getKeysErr     error
		token          *APIToken
		generateErr    error
		addKeyErr      error
		saveConvErr    error
		expectedResp   *Response
		expectedErr    error
		expectAddKey   bool
		expectSaveConv bool
	}{
		{
			name:         "success",
			userID:       "user123",
			existingKeys: []string{},
			getKeysErr:   nil,
			token: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: time.Hour,
			},
			generateErr:    nil,
			addKeyErr:      nil,
			saveConvErr:    nil,
			expectedResp:   &Response{Message: fmt.Sprintf(tokenCreatedMessage, "token123", time.Now().Add(time.Hour).Format(time.DateTime))},
			expectedErr:    nil,
			expectAddKey:   true,
			expectSaveConv: false,
		},
		{
			name:           "token exists",
			userID:         "user123",
			existingKeys:   []string{"existing-key"},
			getKeysErr:     nil,
			token:          nil,
			generateErr:    nil,
			addKeyErr:      nil,
			saveConvErr:    nil,
			expectedResp:   &Response{Message: "You already have an active API token. Do you want to regenerate it?", Answers: []string{"Yes", "No"}},
			expectedErr:    nil,
			expectAddKey:   false,
			expectSaveConv: true,
		},
		{
			name:           "get keys error",
			userID:         "user123",
			existingKeys:   nil,
			getKeysErr:     errors.New("get keys error"),
			token:          nil,
			generateErr:    nil,
			addKeyErr:      nil,
			saveConvErr:    nil,
			expectedResp:   nil,
			expectedErr:    errors.New("failed to get API keys: get keys error"),
			expectAddKey:   false,
			expectSaveConv: false,
		},
		{
			name:           "generate token error",
			userID:         "user123",
			existingKeys:   []string{},
			getKeysErr:     nil,
			token:          nil,
			generateErr:    errors.New("generate token error"),
			addKeyErr:      nil,
			saveConvErr:    nil,
			expectedResp:   nil,
			expectedErr:    errors.New("failed to generate token: generate token error"),
			expectAddKey:   false,
			expectSaveConv: false,
		},
		{
			name:         "add key error",
			userID:       "user123",
			existingKeys: []string{},
			getKeysErr:   nil,
			token: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: time.Hour,
			},
			generateErr:    nil,
			addKeyErr:      errors.New("add key error"),
			saveConvErr:    nil,
			expectedResp:   nil,
			expectedErr:    errors.New("failed to add API key: add key error"),
			expectAddKey:   true,
			expectSaveConv: false,
		},
		{
			name:           "save conversation error",
			userID:         "user123",
			existingKeys:   []string{"existing-key"},
			getKeysErr:     nil,
			token:          nil,
			generateErr:    nil,
			addKeyErr:      nil,
			saveConvErr:    errors.New("save conversation error"),
			expectedResp:   nil,
			expectedErr:    errors.New("failed to save conversation: save conversation error"),
			expectAddKey:   false,
			expectSaveConv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			repo.On("GetAPIKeys", mock.Anything, tt.userID).Return(tt.existingKeys, tt.getKeysErr)

			if tt.expectAddKey {
				repo.On("AddAPIKey", mock.Anything, tt.userID, tt.token.KeyID, tt.token.ExpiresIn).Return(tt.addKeyErr)
			}

			if tt.existingKeys != nil && len(tt.existingKeys) == 0 {
				prov.On("GenerateToken").Return(tt.token, tt.generateErr)
			}

			if tt.expectSaveConv {
				// Mock GetConversation to return a new conversation
				repo.On("GetConversation", mock.Anything, tt.userID).Return(conv.New(tt.userID), nil)

				// Create a matcher function that validates the conversation object
				repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
					// Verify that the conversation has the correct user ID and state
					return c.ID == tt.userID && c.State == "tokenExists"
				})).Return(tt.saveConvErr)
			}

			svc := New(repo, prov)

			resp, err := svc.CreateToken(context.Background(), tt.userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.expectedErr.Error() != "" {
					assert.Equal(t, tt.expectedErr.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
				if tt.expectedResp != nil {
					assert.Equal(t, tt.expectedResp.Message, resp.Message)
					assert.Equal(t, tt.expectedResp.Answers, resp.Answers)
				} else {
					assert.Nil(t, resp)
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
		answers         []conv.QuestionAnswer
		revokeErr       error
		createTokenResp *Response
		createTokenErr  error
		expectedResp    *Response
		expectedErr     string
	}{
		{
			name:   "answer is No",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{
					Question: conv.Question{
						Text:    "You already have an active API token. Do you want to regenerate it?",
						Answers: []string{"Yes", "No"},
					},
					Answer: "No",
				},
			},
			expectedResp: &Response{
				Message: "No changes made. You can continue using your existing API token.",
			},
		},
		{
			name:   "answer is Yes - success",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{
					Question: conv.Question{
						Text:    "You already have an active API token. Do you want to regenerate it?",
						Answers: []string{"Yes", "No"},
					},
					Answer: "Yes",
				},
			},
			createTokenResp: &Response{
				Message: fmt.Sprintf(tokenCreatedMessage, "new-token", time.Now().Add(time.Hour).Format(time.DateTime)),
			},
			expectedResp: &Response{
				Message: fmt.Sprintf(tokenCreatedMessage, "new-token", time.Now().Add(time.Hour).Format(time.DateTime)),
			},
		},
		{
			name:   "answer is Yes - revoke error",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{
					Question: conv.Question{
						Text:    "You already have an active API token. Do you want to regenerate it?",
						Answers: []string{"Yes", "No"},
					},
					Answer: "Yes",
				},
			},
			revokeErr:   errors.New("revoke error"),
			expectedErr: "failed to revoke existing token: failed to remove API key from repository: revoke error",
		},
		{
			name:   "answer is Yes - create token error",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{
					Question: conv.Question{
						Text:    "You already have an active API token. Do you want to regenerate it?",
						Answers: []string{"Yes", "No"},
					},
					Answer: "Yes",
				},
			},
			createTokenErr: errors.New("create token error"),
			expectedErr:    "failed to generate token: create token error",
		},
		{
			name:   "invalid number of answers",
			userID: "user123",
			answers: []conv.QuestionAnswer{
				{
					Question: conv.Question{
						Text:    "Question 1",
						Answers: []string{"Answer 1", "Answer 2"},
					},
					Answer: "Answer 1",
				},
				{
					Question: conv.Question{
						Text:    "Question 2",
						Answers: []string{"Answer 1", "Answer 2"},
					},
					Answer: "Answer 2",
				},
			},
			expectedErr: "expected exactly one answer for tokenExists question, got 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			// Create a partial mock of Service to mock CreateToken
			svc := New(repo, prov)

			// Setup mocks for RevokeToken if needed
			if tt.answers[0].Answer == "Yes" {
				// First GetAPIKeys call is to check if token exists before revoking
				repo.On("GetAPIKeys", mock.Anything, tt.userID).Return([]string{"existing-key"}, nil).Once()

				// Mock RevokeToken
				repo.On("RevokeToken", mock.Anything, tt.userID, "existing-key").Return(tt.revokeErr)

				// Mock the provider's RevokeToken method
				prov.On("RevokeToken", "existing-key").Return(nil)

				if tt.revokeErr == nil {
					// If RevokeToken succeeds, we need to mock CreateToken
					// Second GetAPIKeys call is after token is revoked, should return empty list
					repo.On("GetAPIKeys", mock.Anything, tt.userID).Return([]string{}, nil).Once()

					if tt.createTokenErr == nil {
						// Mock successful token generation
						prov.On("GenerateToken").Return(&APIToken{
							KeyID:     "new-key",
							Token:     "new-token",
							ExpiresIn: time.Hour,
						}, nil)

						repo.On("AddAPIKey", mock.Anything, tt.userID, "new-key", time.Hour).Return(nil)
					} else {
						// Mock failed token generation
						prov.On("GenerateToken").Return(nil, tt.createTokenErr)
					}
				}
			}

			resp, err := svc.handleTokenExistsResult(context.Background(), tt.userID, tt.answers)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedResp.Message, resp.Message)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}
