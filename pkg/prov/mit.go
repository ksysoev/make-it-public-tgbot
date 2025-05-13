package prov

import (
	"net/http"
	"time"
)

type Config struct {
	Url string `mapstructure:"url"`
}

type MIT struct {
	baseUrl string
	cl      *http.Client
}

// New creates and returns a new instance of the MIT struct initialized with the provided configuration.
func New(cfg Config) *MIT {
	return &MIT{
		baseUrl: cfg.Url,
		cl: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}
