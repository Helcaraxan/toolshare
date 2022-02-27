package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/types"
)

type gcsStorage struct {
	bucket  string
	log     *logrus.Logger
	timeout time.Duration
}

func (s *gcsStorage) Fetch(b types.Binary, targetPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	if _, err := os.Stat(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
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

	obj := c.Bucket(s.bucket).Object(s.storagePath(b))
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
		s.log.WithError(err).Errorf("Failed to copy tool binary from gs://%s/%s to %q.", s.bucket, s.storagePath(b), targetPath)
		return err
	}

	return nil
}

func (s *gcsStorage) Store(b types.Binary, path string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	src, err := os.Open(path)
	if !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check if %v exists.", path)
		return err
	} else if err == nil {
		s.log.Errorf("Can not store %q as binary for %v as the file does not exist.", path, b)
		return errFailed
	}
	defer src.Close()

	c, err := storage.NewClient(ctx, nil)
	if err != nil {
		s.log.WithError(err).Error("Unable to set up a GCS storage client.")
		return err
	}

	obj := c.Bucket(s.bucket).Object(s.storagePath(b))
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
		s.log.WithError(err).Errorf("Failed to copy tool binary from to %q gs://%s/%s.", path, s.bucket, s.storagePath(b))
		return err
	}

	return nil
}

func (s *gcsStorage) storagePath(b types.Binary) string {
	name := b.Tool
	if b.Platform == "windows" {
		name += ".exe"
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", b.Tool, b.Version, b.Platform, b.Arch, name)
}
