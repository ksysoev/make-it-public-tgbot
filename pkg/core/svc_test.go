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
)

func TestNewService(t *testing.T) {
	repo := NewMockUserRepo(t)
	prov := NewMockMITProv(t)

	svc := New(repo, prov)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, prov, svc.prov)
}

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

func TestRevokeToken(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		existingKeys  []string
		getKeysErr    error
		revokeProvErr error
		revokeRepoErr error
		expectedErr   string
	}{
		{
			name:         "success",
			userID:       "user123",
			existingKeys: []string{"key123"},
			getKeysErr:   nil,
			expectedErr:  "",
		},
		{
			name:         "no API keys found",
			userID:       "user123",
			existingKeys: []string{},
			getKeysErr:   nil,
			expectedErr:  ErrTokenNotFound.Error(),
		},
		{
			name:         "multiple API keys found",
			userID:       "user123",
			existingKeys: []string{"key123", "key456"},
			getKeysErr:   nil,
			expectedErr:  "multiple API keys found for user user123, cannot revoke",
		},
		{
			name:         "get keys error",
			userID:       "user123",
			existingKeys: nil,
			getKeysErr:   errors.New("get keys error"),
			expectedErr:  "failed to get API keys: get keys error",
		},
		{
			name:          "revoke from repository error",
			userID:        "user123",
			existingKeys:  []string{"key123"},
			revokeRepoErr: errors.New("revoke repository error"),
			expectedErr:   "failed to remove API key from repository: revoke repository error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			repo.On("GetAPIKeys", mock.Anything, tt.userID).Return(tt.existingKeys, tt.getKeysErr)

			if len(tt.existingKeys) == 1 {
				prov.On("RevokeToken", tt.existingKeys[0]).Return(tt.revokeProvErr)
				repo.On("RevokeToken", mock.Anything, tt.userID, tt.existingKeys[0]).Return(tt.revokeRepoErr)
			}

			svc := New(repo, prov)

			err := svc.RevokeToken(context.Background(), tt.userID)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}
