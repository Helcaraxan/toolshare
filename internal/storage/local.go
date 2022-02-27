package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/sirupsen/logrus"

	"github.com/improbable/toolshare/internal/types"
)

type localStorage struct {
	log     *logrus.Logger
	remote  Storage
	storage billy.Filesystem
}

func (s *localStorage) Get(b types.Binary) (string, error) {
	localPath := s.storagePath(b)
	if _, err := s.storage.Stat(localPath); err == nil {
		return localPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check local presence of %v.", b)
		return "", err
	}

	if s.remote == nil {
		s.log.Debugf("Not fetch binary for %v as there is no remote configured.", b)
		return "", errFailed
	} else if err := s.remote.Fetch(b, localPath); err != nil {
		return "", err
	}

	return localPath, nil
}

func (s *localStorage) Fetch(b types.Binary, targetPath string) error {
	if _, err := s.storage.Stat(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check if %v already exists.", targetPath)
		return err
	} else if err == nil {
		s.log.Errorf("Can not fetch binary to %q as it already exixts.", targetPath)
		return errFailed
	}

	localPath := s.storagePath(b)
	if _, err := s.storage.Stat(localPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.log.WithError(err).Errorf("Unable to check availability of %v.", b)
		} else {
			s.log.Errorf("No binary for %v available.", b)
		}
		return err
	}

	return s.localCopyBinary(localPath, targetPath)
}

func (s *localStorage) Store(b types.Binary, path string) error {
	localPath := s.storagePath(b)
	if _, err := s.storage.Stat(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check for the presence of %v.", b)
		return err
	} else if err == nil {
		s.log.WithError(err).Errorf("Can not store %v as it is already present.", b)
		return errFailed
	}

	return s.localCopyBinary(path, localPath)
}

func (s *localStorage) storagePath(b types.Binary) string {
	name := b.Tool
	if b.Platform == "windows" {
		name += ".exe"
	}
	return filepath.Join(b.Tool, b.Version, b.Platform, b.Arch, name)
}

func (s *localStorage) localCopyBinary(srcPath string, dstPath string) error {
	src, err := s.storage.Open(srcPath)
	if err != nil {
		s.log.WithError(err).Errorf("Unable to open the source file %q.", srcPath)
		return err
	}
	defer src.Close()

	if err = s.storage.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		s.log.WithError(err).Errorf("Unable to create directory that will contain %q.", dstPath)
		return err
	}

	dst, err := s.storage.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		s.log.WithError(err).Errorf("Unable to open the target file %q.", dstPath)
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		s.log.WithError(err).Errorf("Failed to copy tool binary from %q to %q.", srcPath, dstPath)
		return err
	}
	return nil
}
