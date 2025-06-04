package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
