package backend

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type S3Config struct {
	CommonConfig

	S3Bucket       string `yaml:"s3_bucket"`
	S3PathTemplate string `yaml:"s3_path_template"`
}

type S3 struct {
	log     *logrus.Logger
	timeout time.Duration

	S3Config
}

func NewS3(log *logrus.Logger, c *S3Config) *S3 {
	return &S3{
		log:      log,
		timeout:  time.Minute,
		S3Config: *c,
	}
}

func (s *S3) Fetch(_ config.Binary) ([]byte, error) {
	s.log.Error("Unimplemented.")
	return nil, errFailed
}

func (s *S3) Store(_ config.Binary, _ []byte) error {
	s.log.Error("Unimplemented.")
	return errFailed
}
