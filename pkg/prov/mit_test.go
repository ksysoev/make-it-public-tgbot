package prov

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := Config{
		Url:        "https://example.com",
		DefaultTTL: 3600,
	}

	mit := New(cfg)

	assert.NotNil(t, mit)
	assert.Equal(t, cfg.Url, mit.baseUrl)
	assert.Equal(t, cfg.DefaultTTL, mit.defaultTTL)
	assert.NotNil(t, mit.cl)
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedToken  *core.APIToken
		name           string
		tokenType      core.TokenType
		expectedError  string
		defaultTTL     int64
	}{
		{
			name:       "success web token",
			defaultTTL: 3600,
			tokenType:  core.TokenTypeWeb,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var req generateTokenRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				assert.NoError(t, err)
				assert.Equal(t, int64(3600), req.TTL)
				assert.Equal(t, "web", req.Type)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				resp := generateTokenResponse{
					Token: "test-token",
					KeyID: "test-key-id",
					Type:  "web",
					TTL:   3600,
				}
				_ = json.NewEncoder(w).Encode(resp)
			},
			expectedToken: &core.APIToken{
				Token:     "test-token",
				KeyID:     "test-key-id",
				Type:      core.TokenTypeWeb,
				ExpiresIn: time.Hour,
			},
		},
		{
			name:       "success tcp token",
			defaultTTL: 3600,
			tokenType:  core.TokenTypeTCP,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var req generateTokenRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				assert.NoError(t, err)
				assert.Equal(t, "tcp", req.Type)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				resp := generateTokenResponse{
					Token: "tcp-token",
					KeyID: "tcp-key-id",
					Type:  "tcp",
					TTL:   3600,
				}
				_ = json.NewEncoder(w).Encode(resp)
			},
			expectedToken: &core.APIToken{
				Token:     "tcp-token",
				KeyID:     "tcp-key-id",
				Type:      core.TokenTypeTCP,
				ExpiresIn: time.Hour,
			},
		},
		{
			name:      "server error",
			tokenType: core.TokenTypeWeb,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "failed to generate token, status code: 500",
		},
		{
			name:      "invalid response",
			tokenType: core.TokenTypeWeb,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("invalid json"))
			},
			expectedError: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			mit := &MIT{
				defaultTTL: tt.defaultTTL,
				baseUrl:    server.URL,
				cl:         &http.Client{},
			}

			token, err := mit.GenerateToken("", tt.tokenType, tt.defaultTTL)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Nil(t, token)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestRevokeToken(t *testing.T) {
	tests := []struct {
		name           string
		keyID          string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:  "success",
			keyID: "valid-key",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/token/valid-key", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			name:  "not found",
			keyID: "invalid-key",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/token/invalid-key", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			},
		},
		{
			name:  "server error",
			keyID: "error-key",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/token/error-key", r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "failed to revoke token, status code: 500",
		},
		{
			name:  "bad request",
			keyID: "bad-request-key",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/token/bad-request-key", r.URL.Path)
				w.WriteHeader(http.StatusBadRequest)
			},
			expectedError: "failed to revoke token, status code: 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			mit := &MIT{
				baseUrl: server.URL,
				cl:      &http.Client{},
			}

			err := mit.RevokeToken(tt.keyID)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
