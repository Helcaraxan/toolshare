package backend

import (
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

var ErrUnimplemented = errors.New("unimplemented")

type HTTPSConfig struct {
	CommonConfig

	HTTPSURLTemplate string `json:"https_url_template"`
}

func (c HTTPSConfig) String() string {
	return c.HTTPSURLTemplate
}

type HTTPS struct {
	log     *zap.Logger
	timeout time.Duration

	HTTPSConfig
}

func NewHTTPS(logBuilder logger.Builder, c *HTTPSConfig) *HTTPS {
	return &HTTPS{
		log:         logBuilder.Domain(logger.HTTPSDomain),
		timeout:     time.Minute,
		HTTPSConfig: *c,
	}
}

func (s *HTTPS) Fetch(b config.Binary) ([]byte, error) {
	return nil, ErrUnimplemented
}

func (s *HTTPS) Store(b config.Binary, content []byte) error {
	return ErrUnimplemented
}
