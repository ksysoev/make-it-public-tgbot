package core

import "time"

type APIToken struct {
	KeyID     string
	Token     string
	ExpiresIn time.Duration
}
