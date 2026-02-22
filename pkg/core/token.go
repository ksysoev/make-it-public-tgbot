package core

import "time"

// TokenType represents the type of an API token.
type TokenType string

const (
	// TokenTypeWeb is a web-tunneling token (max 3 per user).
	TokenTypeWeb TokenType = "web"
	// TokenTypeTCP is a raw TCP-tunneling token (max 1 per user).
	TokenTypeTCP TokenType = "tcp"
)

// APIToken holds the details of a newly generated API token.
type APIToken struct {
	KeyID     string
	Token     string
	Type      TokenType
	ExpiresIn time.Duration
}

// KeyInfo holds the display information for an existing API key.
type KeyInfo struct {
	ExpiresAt time.Time
	KeyID     string
	Type      TokenType
}
