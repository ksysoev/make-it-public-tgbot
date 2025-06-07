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
			expectedResp:   &Response{Message: "What is the expiration period for your new API token?", Answers: []string{"1 day", "7 days", "30 days", "90 days"}},
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
			expectedResp:   &Response{Message: "What is the expiration period for your new API token?", Answers: []string{"1 day", "7 days", "30 days", "90 days"}},
			expectedErr:    nil,
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
			expectedResp:   &Response{Message: "What is the expiration period for your new API token?", Answers: []string{"1 day", "7 days", "30 days", "90 days"}},
			expectedErr:    nil,
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

			// Mock GetConversation for all test cases
			// We don't care how many times it's called
			repo.On("GetConversation", mock.Anything, tt.userID).Return(conv.New(tt.userID), nil).Maybe()

			// Mock SaveConversation for all test cases where we create a new token
			if tt.existingKeys != nil && len(tt.existingKeys) == 0 {
				repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
					return c.ID == tt.userID && c.State == "newToken"
				})).Return(tt.saveConvErr)
			}

			if tt.expectSaveConv {
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
				Message: "What is the expiration period for your new API token?",
				Answers: []string{"1 day", "7 days", "30 days", "90 days"},
			},
			expectedResp: &Response{
				Message: "What is the expiration period for your new API token?",
				Answers: []string{"1 day", "7 days", "30 days", "90 days"},
			},
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
			createTokenResp: &Response{
				Message: "What is the expiration period for your new API token?",
				Answers: []string{"1 day", "7 days", "30 days", "90 days"},
			},
			expectedResp: &Response{
				Message: "What is the expiration period for your new API token?",
				Answers: []string{"1 day", "7 days", "30 days", "90 days"},
			},
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

			// Setup conversation mocking for all test cases
			conversation := conv.New(tt.userID)

			// Setup mocks for tokenRegenerate state
			if tt.answers[0].Answer == "Yes" {
				repo.On("GetConversation", mock.Anything, tt.userID).Return(conversation, nil)

				// Mock SaveConversation for tokenRegenerate state
				repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
					return c.ID == tt.userID && c.State == StateTokenRegenerate
				})).Return(nil)
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
