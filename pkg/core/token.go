package core

import "time"

// APIToken holds the details of a newly generated API token.
type APIToken struct {
	KeyID     string
	Token     string
	ExpiresIn time.Duration
}

// KeyInfo holds the display information for an existing API key.
type KeyInfo struct {
	ExpiresAt time.Time
	KeyID     string
}
