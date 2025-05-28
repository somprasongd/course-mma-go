package config

import (
	"errors"
	"go-mma/util/env"
)

var (
	ErrInvalidHTTPPort = errors.New("HTTP_PORT must be a positive integer")
)

type Config struct {
	HTTPPort int
}

func Load() (*Config, error) {
	config := &Config{
		HTTPPort: env.GetIntDefault("HTTP_PORT", 8090),
	}
	err := config.Validate()
	if err != nil {
		return nil, err
	}
	return config, err
}

func (c *Config) Validate() error {
	if c.HTTPPort <= 0 {
		return ErrInvalidHTTPPort
	}
	return nil
}
