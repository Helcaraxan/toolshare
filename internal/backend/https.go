package backend

import (
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type HTTPSConfig struct {
	CommonConfig

	HTTPSURLTemplate string `yaml:"https_url_template"`
}

type HTTPS struct {
	log     *logrus.Logger
	timeout time.Duration

	HTTPSConfig
}

func NewHTTPS(log *logrus.Logger, c *HTTPSConfig) *HTTPS {
	return &HTTPS{
		log:         log,
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
