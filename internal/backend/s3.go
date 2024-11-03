package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type S3Config struct {
	CommonConfig

	S3Bucket       string `json:"s3_bucket"`
	S3PathTemplate string `json:"s3_path_template"`
}

func (c S3Config) String() string {
	return fmt.Sprintf("s3://%s/%s", c.S3Bucket, c.S3PathTemplate)
}

type S3 struct {
	log     *zap.Logger
	timeout time.Duration
	client  *s3.Client

	S3Config
}

func NewS3(logBuilder logger.Builder, c *S3Config) *S3 {
	log := logBuilder.Domain(logger.S3Domain).With(zap.String("s3-bucket", c.S3Bucket))

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cfg, err := aws_config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("Failed to load AWS configuration from environment.", zap.Error(err))
	}
	cancel()

	return &S3{
		log:      log,
		timeout:  time.Minute,
		client:   s3.NewFromConfig(cfg),
		S3Config: *c,
	}
}

func (s *S3) Fetch(b config.Binary) ([]byte, error) {
	bucketPath := s.instantiateTemplate(b, s.S3PathTemplate)
	log := s.log.With(
		zap.Stringer("tool", b),
		zap.String("artefact-path", bucketPath),
	)

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.S3Bucket),
		Key:    aws.String(bucketPath),
	})
	if err != nil {
		var s3err *types.NoSuchKey
		if errors.As(err, &s3err) {
			log.Error("No such object available in S3.", zap.Error(err))
		} else {
			log.Error("Failed to lookup object on S3.", zap.Error(err))
		}
		return nil, err
	}
	defer out.Body.Close()

	raw, err := io.ReadAll(out.Body)
	if err != nil {
		log.Error("Failed to download object content from S3.", zap.Error(err))
		return nil, err
	}
	s.log.Debug("Finished downloading object from S3.")
	return s.extractFromArchive(log, raw, bucketPath, b)
}

func (s *S3) Store(b config.Binary, content []byte) error {
	bucketPath := s.instantiateTemplate(b, s.S3PathTemplate)
	log := s.log.With(
		zap.Stringer("tool", b),
		zap.String("artefact-path", bucketPath),
	)

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	_, err := s.client.GetObjectAttributes(ctx, &s3.GetObjectAttributesInput{
		Bucket:           aws.String(s.S3Bucket),
		Key:              aws.String(bucketPath),
		ObjectAttributes: []types.ObjectAttributes{},
	})
	if err == nil {
		log.Error("Can not store a binary as one already exists.")
		return errFailed
	}
	var s3err *types.NoSuchKey
	if !errors.As(err, &s3err) {
		log.Error("Failed to check if a binary already exists.", zap.Error(err))
		return err
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.S3Bucket),
		Key:    aws.String(bucketPath),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		log.Error("Failed to store binary as object in S3.", zap.Error(err))
		return err
	}
	log.Debug("Finished uploading the binary as object to S3.")
	return nil
}
