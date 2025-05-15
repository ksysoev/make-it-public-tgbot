package prov

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
)

type Config struct {
	Url        string `mapstructure:"url"`
	DefaultTTL int64  `mapstructure:"default_ttl"`
}

type MIT struct {
	defaultTTL int64
	baseUrl    string
	cl         *http.Client
}

// New creates and returns a new instance of the MIT struct initialized with the provided configuration.
func New(cfg Config) *MIT {
	return &MIT{
		defaultTTL: cfg.DefaultTTL,
		baseUrl:    cfg.Url,
		cl: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type generateTokenRequest struct {
	KeyID string `json:"key_id"`
	TTL   int64  `json:"ttl"`
}

type generateTokenResponse struct {
	Token string `json:"token"`
	KeyID string `json:"key_id"`
	TTL   int64  `json:"ttl"`
}

// GenerateToken sends a request to generate an API token and returns the token along with its metadata or an error.
func (m *MIT) GenerateToken() (*core.APIToken, error) {
	req := generateTokenRequest{
		TTL: m.defaultTTL,
	}

	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := m.cl.Post(m.baseUrl+"/generateToken", "application/json", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to generate token, status code: %d", resp.StatusCode)
	}

	var tokenResp generateTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	token := core.APIToken{
		Token:     tokenResp.Token,
		KeyID:     tokenResp.KeyID,
		ExpiresIn: tokenResp.TTL,
	}

	return &token, fmt.Errorf("not implemented")
}
