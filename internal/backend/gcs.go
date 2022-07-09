package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type GCSConfig struct {
	CommonConfig

	GCSBucket       string `yaml:"gcs_bucket"`
	GCSPathTemplate string `yaml:"gcs_path_template"`
}

func (c GCSConfig) String() string {
	return fmt.Sprintf("gs://%s/%s", c.GCSBucket, c.GCSPathTemplate)
}

type GCS struct {
	log     *zap.Logger
	timeout time.Duration

	GCSConfig
}

func NewGCS(logBuilder *logger.Builder, c *GCSConfig) *GCS {
	return &GCS{
		log:       logBuilder.Domain(logger.GCSDomain).With(zap.String("gcs-bucket", c.GCSBucket)),
		timeout:   time.Minute,
		GCSConfig: *c,
	}
}

func (s *GCS) Fetch(b config.Binary) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	log := s.log.With(zap.Stringer("tool", b))

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		log.Error("Unable to set up a GCS storage client.", zap.Error(err))
		return nil, err
	}

	bucketPath := s.instantiateTemplate(b, s.GCSPathTemplate)
	log = log.With(zap.String("artefact-path", bucketPath))

	obj := c.Bucket(s.GCSBucket).Object(bucketPath)
	src, err := obj.NewReader(context.Background()) // Background context as we don't want to interrupt a download.
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			log.Error("No binary found.")
		} else {
			log.Error("Unable to open reader on remote GCS object.", zap.Error(err))
		}
		return nil, err
	}
	defer src.Close()

	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	s.log.Debug("Finished downloading remote artefact.")
	return s.extractFromArchive(log, raw, bucketPath, b)
}

func (s *GCS) Store(b config.Binary, content []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	log := s.log.With(zap.Stringer("tool", b))

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		log.Error("Unable to set up a GCS storage client.", zap.Error(err))
		return err
	}

	bucketPath := s.instantiateTemplate(b, s.GCSPathTemplate)
	log = log.With(zap.String("artefact-path", bucketPath))

	obj := c.Bucket(s.GCSBucket).Object(bucketPath)
	if _, err = obj.Attrs(ctx); err == nil {
		log.Error("Can not store new binary as one already exists.")
		return errFailed
	} else if !errors.Is(err, storage.ErrObjectNotExist) {
		log.Error("Can not check if a binary already exists.", zap.Error(err))
		return err
	}

	dst := obj.NewWriter(context.Background()) // Background context as we don't want to interrupt an upload.
	defer func() {
		closeErr := dst.Close()
		if err == nil && closeErr != nil {
			log.Error("Failed to correctly close remote object.", zap.Error(err))
			err = closeErr
		}
	}()

	if _, err = io.Copy(dst, bytes.NewReader(content)); err != nil {
		log.Error("Failed to upload tool binary.", zap.Error(err))
		return err
	}
	return nil
}
