package prov

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/stretchr/testify/assert"
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
		name           string
		defaultTTL     int64
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedToken  *core.APIToken
		expectedError  string
	}{
		{
			name:       "success",
			defaultTTL: 3600,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Decode request body
				var req generateTokenRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				assert.NoError(t, err)
				assert.Equal(t, int64(3600), req.TTL)

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				resp := generateTokenResponse{
					Token: "test-token",
					KeyID: "test-key-id",
					TTL:   3600,
				}
				_ = json.NewEncoder(w).Encode(resp)
			},
			expectedToken: &core.APIToken{
				Token:     "test-token",
				KeyID:     "test-key-id",
				ExpiresIn: 3600,
			},
			expectedError: "",
		},
		{
			name:       "server error",
			defaultTTL: 3600,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedToken: nil,
			expectedError: "failed to generate token, status code: 500",
		},
		{
			name:       "invalid response",
			defaultTTL: 3600,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid json"))
			},
			expectedToken: nil,
			expectedError: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Create MIT instance with test server URL
			mit := &MIT{
				defaultTTL: tt.defaultTTL,
				baseUrl:    server.URL,
				cl:         &http.Client{},
			}

			// Call the method
			token, err := mit.GenerateToken()

			// Verify results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Nil(t, token)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}
