package backend

import (
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type HTTPSConfig struct {
	CommonConfig

	HTTPSURLTemplate string `yaml:"https_url_template"`
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
	return nil, errors.New("not yet implemented")
}

func (s *HTTPS) Store(b config.Binary, content []byte) error {
	return errors.New("not yet implemented")
}
