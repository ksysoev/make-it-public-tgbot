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

	resp, err := m.cl.Post(m.baseUrl+"/token", "application/json", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to generate token, status code: %d", resp.StatusCode)
	}

	var tkn generateTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&tkn); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &core.APIToken{
		Token:     tkn.Token,
		KeyID:     tkn.KeyID,
		ExpiresIn: time.Duration(tkn.TTL) * time.Second,
	}, nil
}

// RevokeToken sends a request to revoke an API token based on the provided key ID and returns an error if the request fails.
func (m *MIT) RevokeToken(keyID string) error {
	req, err := http.NewRequest("DELETE", m.baseUrl+"/token/"+keyID, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.cl.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to revoke token, status code: %d", resp.StatusCode)
	}

	return nil
}
