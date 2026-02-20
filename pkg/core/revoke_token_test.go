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

func TestRevokeToken(t *testing.T) {
	tests := []struct {
		getKeysErr    error
		revokeProvErr error
		revokeRepoErr error
		expectedResp  *Response
		name          string
		userID        string
		expectedErr   string
		existingKeys  []string
	}{
		{
			name:         "single token - success",
			userID:       "user123",
			existingKeys: []string{"key123"},
			expectedResp: nil, // direct revoke, no conversational response
			expectedErr:  "",
		},
		{
			name:         "no API keys found",
			userID:       "user123",
			existingKeys: []string{},
			expectedErr:  ErrTokenNotFound.Error(),
		},
		{
			name:         "get keys error",
			userID:       "user123",
			existingKeys: nil,
			getKeysErr:   errors.New("get keys error"),
			expectedErr:  "failed to get API keys: get keys error",
		},
		{
			name:          "single token - revoke provider error",
			userID:        "user123",
			existingKeys:  []string{"key123"},
			revokeProvErr: errors.New("revoke provider error"),
			expectedErr:   "failed to revoke token: revoke provider error",
		},
		{
			name:          "single token - revoke repository error",
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

			if len(tt.existingKeys) == 1 && tt.getKeysErr == nil {
				prov.On("RevokeToken", tt.existingKeys[0]).Return(tt.revokeProvErr)

				if tt.revokeProvErr == nil {
					repo.On("RevokeToken", mock.Anything, tt.userID, tt.existingKeys[0]).Return(tt.revokeRepoErr)
				}
			}

			svc := New(repo, prov)

			resp, err := svc.RevokeToken(context.Background(), tt.userID)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}

func TestRevokeToken_MultipleKeys(t *testing.T) {
	userID := "user123"
	keys := []KeyInfo{
		{KeyID: "abcdef1234", ExpiresAt: time.Now().Add(24 * time.Hour)},
		{KeyID: "xyz9876543", ExpiresAt: time.Now().Add(48 * time.Hour)},
	}

	repo := NewMockUserRepo(t)
	prov := NewMockMITProv(t)

	repo.On("GetAPIKeys", mock.Anything, userID).Return([]string{"abcdef1234", "xyz9876543"}, nil)
	repo.On("GetAPIKeysWithExpiration", mock.Anything, userID).Return(keys, nil)
	repo.On("GetConversation", mock.Anything, userID).Return(conv.New(userID), nil)
	repo.On("SaveConversation", mock.Anything, mock.MatchedBy(func(c *conv.Conversation) bool {
		return c.ID == userID && c.State == StateSelectTokenToRevoke
	})).Return(nil)

	svc := New(repo, prov)

	resp, err := svc.RevokeToken(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Which token do you want to revoke?", resp.Message)
	assert.Len(t, resp.Answers, 2)

	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

func TestHandleSelectTokenToRevokeResult(t *testing.T) {
	userID := "user123"
	keyID := "abcdef123456"

	t.Run("success", func(t *testing.T) {
		repo := NewMockUserRepo(t)
		prov := NewMockMITProv(t)

		repo.On("GetAPIKeys", mock.Anything, userID).Return([]string{keyID}, nil)
		prov.On("RevokeToken", keyID).Return(nil)
		repo.On("RevokeToken", mock.Anything, userID, keyID).Return(nil)

		svc := New(repo, prov)

		answers := []conv.QuestionAnswer{
			{Answer: keyID[:keyIDDisplayLen] + " (exp: 2026-03-01)"},
		}

		resp, err := svc.handleSelectTokenToRevokeResult(context.Background(), userID, answers)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "revoked")

		repo.AssertExpectations(t)
		prov.AssertExpectations(t)
	})

	t.Run("wrong number of answers", func(t *testing.T) {
		svc := New(NewMockUserRepo(t), NewMockMITProv(t))

		_, err := svc.handleSelectTokenToRevokeResult(context.Background(), userID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected exactly one answer")
	})

	t.Run("key not found", func(t *testing.T) {
		repo := NewMockUserRepo(t)
		prov := NewMockMITProv(t)

		repo.On("GetAPIKeys", mock.Anything, userID).Return([]string{"different123456"}, nil)

		svc := New(repo, prov)

		answers := []conv.QuestionAnswer{
			{Answer: "nomatch1 (exp: 2026-03-01)"},
		}

		_, err := svc.handleSelectTokenToRevokeResult(context.Background(), userID, answers)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve key ID")

		repo.AssertExpectations(t)
	})
}
