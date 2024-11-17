package backend

import (
	"errors"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

var (
	ErrUnsupported    = errors.New("unsupported")
	ErrHTTPStatusCode = errors.New("received a non-200 http status code")
)

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
	u := s.instantiateTemplate(b, s.HTTPSURLTemplate)
	log := s.log.With(zap.Stringer("tool", b), zap.String("url", u))

	r, err := http.Get(u)
	if err != nil {
		log.Error("Failed to download tool source URL.", zap.Error(err))
		return nil, err
	} else if r.StatusCode != http.StatusOK {
		log.Error("Download of tool source URL returned a non-200 code.", zap.Int("http-code", r.StatusCode))
		return nil, ErrHTTPStatusCode
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read full file from remote URL.", zap.Error(err))
		return nil, err
	}
	return s.extractFromArchive(log, raw, u, b)
}

func (s *HTTPS) Store(b config.Binary, content []byte) error {
	// We deliberately do not support storage through HTTP as we do not yet provide a HTTP authentication mechanism.
	return ErrUnsupported
}
