package core

import (
	"context"
	"errors"
	"testing"
	"time"

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
		name          string
		userID        string
		existingKeys  []string
		getKeysErr    error
		token         *APIToken
		generateErr   error
		addKeyErr     error
		expectedToken *APIToken
		expectedErr   error
		expectAddKey  bool
	}{
		{
			name:         "success",
			userID:       "user123",
			existingKeys: []string{},
			getKeysErr:   nil,
			token: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: 3600,
			},
			generateErr: nil,
			addKeyErr:   nil,
			expectedToken: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: 3600,
			},
			expectedErr:  nil,
			expectAddKey: true,
		},
		{
			name:          "max tokens exceeded",
			userID:        "user123",
			existingKeys:  []string{"existing-key"},
			getKeysErr:    nil,
			token:         nil,
			generateErr:   nil,
			addKeyErr:     nil,
			expectedToken: nil,
			expectedErr:   ErrMaxTokensExceeded,
			expectAddKey:  false,
		},
		{
			name:          "get keys error",
			userID:        "user123",
			existingKeys:  nil,
			getKeysErr:    errors.New("get keys error"),
			token:         nil,
			generateErr:   nil,
			addKeyErr:     nil,
			expectedToken: nil,
			expectedErr:   errors.New("failed to get API keys: get keys error"),
			expectAddKey:  false,
		},
		{
			name:          "generate token error",
			userID:        "user123",
			existingKeys:  []string{},
			getKeysErr:    nil,
			token:         nil,
			generateErr:   errors.New("generate token error"),
			addKeyErr:     nil,
			expectedToken: nil,
			expectedErr:   errors.New("failed to generate token: generate token error"),
			expectAddKey:  false,
		},
		{
			name:         "add key error",
			userID:       "user123",
			existingKeys: []string{},
			getKeysErr:   nil,
			token: &APIToken{
				KeyID:     "key123",
				Token:     "token123",
				ExpiresIn: 3600,
			},
			generateErr:   nil,
			addKeyErr:     errors.New("add key error"),
			expectedToken: nil,
			expectedErr:   errors.New("failed to add API key: add key error"),
			expectAddKey:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			repo.On("GetAPIKeys", mock.Anything, tt.userID).Return(tt.existingKeys, tt.getKeysErr)

			if tt.expectAddKey {
				repo.On("AddAPIKey", mock.Anything, tt.userID, tt.token.KeyID, time.Duration(tt.token.ExpiresIn)*time.Second).Return(tt.addKeyErr)
			}

			if tt.existingKeys != nil && len(tt.existingKeys) == 0 {
				prov.On("GenerateToken").Return(tt.token, tt.generateErr)
			}

			svc := New(repo, prov)

			token, err := svc.CreateToken(context.Background(), tt.userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.expectedErr.Error() != "" {
					assert.Equal(t, tt.expectedErr.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}
