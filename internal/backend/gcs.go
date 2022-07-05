package backend

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type GCSConfig struct {
	CommonConfig

	GCSBucket       string `yaml:"gcs_bucket"`
	GCSPathTemplate string `yaml:"gcs_path_template"`
}

type GCS struct {
	log     *logrus.Logger
	timeout time.Duration

	GCSConfig
}

func NewGCS(log *logrus.Logger, c *GCSConfig) *GCS {
	return &GCS{
		log:       log,
		timeout:   time.Minute,
		GCSConfig: *c,
	}
}

func (s *GCS) Fetch(b config.Binary) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		s.log.WithError(err).Error("Unable to set up a GCS storage client.")
		return nil, err
	}

	bucketPath := s.instantiateTemplate(b, s.GCSPathTemplate)
	obj := c.Bucket(s.GCSBucket).Object(bucketPath)
	src, err := obj.NewReader(context.Background()) // Background context as we don't want to interrupt a download.
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			s.log.Errorf("No binary found for %v.", b)
		} else {
			s.log.WithError(err).Errorf("Unable to open reader on remote GCS object for %v.", b)
		}
		return nil, err
	}
	defer src.Close()

	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	return s.extractFromArchive(raw, bucketPath, b)
}

func (s *GCS) Store(b config.Binary, content []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		s.log.WithError(err).Error("Unable to set up a GCS storage client.")
		return err
	}

	bucketPath := s.instantiateTemplate(b, s.GCSPathTemplate)
	obj := c.Bucket(s.GCSBucket).Object(bucketPath)
	if _, err = obj.Attrs(ctx); err == nil {
		s.log.Errorf("Can not store new binary for %q as one already exists.", b)
		return errFailed
	} else if !errors.Is(err, storage.ErrObjectNotExist) {
		s.log.WithError(err).Errorf("Can not check if a binary for %q already exists.", b)
		return err
	}

	dst := obj.NewWriter(context.Background()) // Background context as we don't want to interrupt an upload.
	defer func() {
		closeErr := dst.Close()
		if err == nil && closeErr != nil {
			s.log.WithError(err).Error("Failed to correctly close remote object.")
			err = closeErr
		}
	}()

	if _, err = io.Copy(dst, bytes.NewReader(content)); err != nil {
		s.log.WithError(err).Errorf("Failed to copy tool binary to gs://%s/%s.", s.GCSBucket, bucketPath)
		return err
	}
	return nil
}
