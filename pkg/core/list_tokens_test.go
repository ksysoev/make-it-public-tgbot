package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListTokens(t *testing.T) {
	tests := []struct {
		getKeysErr  error
		expectedErr error
		checkResp   func(t *testing.T, resp *Response)
		name        string
		userID      string
		keys        []KeyInfo
	}{
		{
			name:        "no tokens",
			userID:      "user123",
			keys:        []KeyInfo{},
			expectedErr: ErrTokenNotFound,
		},
		{
			name:   "single token",
			userID: "user123",
			keys: []KeyInfo{
				{KeyID: "abcdef123456789", ExpiresAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)},
			},
			checkResp: func(t *testing.T, resp *Response) {
				t.Helper()
				assert.Contains(t, resp.Message, "1/3")
				assert.Contains(t, resp.Message, "abcdef123456")
				assert.Contains(t, resp.Message, "2026-03-15")
			},
		},
		{
			name:   "multiple tokens",
			userID: "user123",
			keys: []KeyInfo{
				{KeyID: "aaabbb123456789", ExpiresAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)},
				{KeyID: "cccddd987654321", ExpiresAt: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)},
				{KeyID: "eeefff111222333", ExpiresAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
			},
			checkResp: func(t *testing.T, resp *Response) {
				t.Helper()
				assert.Contains(t, resp.Message, "3/3")
				assert.Contains(t, resp.Message, "aaabbb123456")
				assert.Contains(t, resp.Message, "cccddd987654")
				assert.Contains(t, resp.Message, "eeefff111222")
				assert.Contains(t, resp.Message, "/new_token")
				assert.Contains(t, resp.Message, "/revoke_token")
			},
		},
		{
			name:        "get keys error",
			userID:      "user123",
			getKeysErr:  errors.New("redis error"),
			expectedErr: errors.New("failed to get API keys: redis error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			prov := NewMockMITProv(t)

			repo.On("GetAPIKeysWithExpiration", mock.Anything, tt.userID).Return(tt.keys, tt.getKeysErr)

			svc := New(repo, prov)

			resp, err := svc.ListTokens(context.Background(), tt.userID)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResp != nil {
					tt.checkResp(t, resp)
				}
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}
