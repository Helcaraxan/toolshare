package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

type gcsStorage struct {
	log     *logrus.Logger
	timeout time.Duration

	source config.Source
}

func (s *gcsStorage) Fetch(b tool.Binary, targetPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	bucket, bucketPath, err := s.location(b)
	if err != nil {
		return err
	}

	if _, err = os.Stat(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check if %v already exists.", targetPath)
		return err
	} else if err == nil {
		s.log.Errorf("Can not fetch binary to %q as it already exixts.", targetPath)
		return errFailed
	}

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		s.log.WithError(err).Error("Unable to set up a GCS storage client.")
		return err
	}

	obj := c.Bucket(bucket).Object(bucketPath)
	src, err := obj.NewReader(context.Background()) // Background context as we don't want to interrupt a download.
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			s.log.Errorf("No binary found for %v.", b)
		} else {
			s.log.WithError(err).Errorf("Unable to open reader on remote GCS object for %v.", b)
		}
		return err
	}
	defer src.Close()

	if err = os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		s.log.WithError(err).Errorf("Unable to create directory that will contain %q.", targetPath)
		return err
	}

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		s.log.WithError(err).Errorf("Unable to open the target file %q.", targetPath)
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		s.log.WithError(err).Errorf("Failed to copy tool binary from gs://%s/%s to %q.", bucket, bucketPath, targetPath)
		return err
	}
	return nil
}

func (s *gcsStorage) Store(b tool.Binary, sourcePath string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	bucket, bucketPath, err := s.location(b)
	if err != nil {
		return err
	}

	src, err := os.Open(sourcePath)
	if !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check if %v exists.", sourcePath)
		return err
	} else if err == nil {
		s.log.Errorf("Can not store %q as binary for %v as the file does not exist.", sourcePath, b)
		return errFailed
	}
	defer src.Close()

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		s.log.WithError(err).Error("Unable to set up a GCS storage client.")
		return err
	}

	obj := c.Bucket(bucket).Object(bucketPath)
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

	if _, err = io.Copy(dst, src); err != nil {
		s.log.WithError(err).Errorf("Failed to copy tool binary from to %q gs://%s/%s.", sourcePath, bucket, bucketPath)
		return err
	}
	return nil
}

func (s *gcsStorage) location(b tool.Binary) (bucket string, path string, err error) {
	p, err := s.source.ResourcePath(b)
	if err != nil {
		return "", "", err
	}

	u, err := url.Parse(p)
	if err != nil {
		return "", "", err
	} else if u.Scheme != "gs" {
		return "", "", fmt.Errorf("unexpected schema '%s' for gcs storage", u.Scheme)
	}

	elems := strings.SplitN(u.Path, "/", 2)
	if len(elems) < 2 {
		return "", "", fmt.Errorf("gcs storage path '%s' has less than two elements", u.Path)
	}
	return elems[0], elems[1], nil
}
