package backend

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type S3Config struct {
	CommonConfig

	S3Bucket       string `yaml:"s3_bucket"`
	S3PathTemplate string `yaml:"s3_path_template"`
}

func (c S3Config) String() string {
	return fmt.Sprintf("s3://%s/%s", c.S3Bucket, c.S3PathTemplate)
}

type S3 struct {
	log     *zap.Logger
	timeout time.Duration

	S3Config
}

func NewS3(logBuilder logger.Builder, c *S3Config) *S3 {
	return &S3{
		log:      logBuilder.Domain(logger.S3Domain).With(zap.String("s3-bucket", c.S3Bucket)),
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
