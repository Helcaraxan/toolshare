package backend

import (
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"
)

type FileSystemConfig struct {
	CommonConfig

	FilePathTemplate string `yaml:"file_path_template"`
}

type FileSystem struct {
	log     *logrus.Logger
	storage billy.Filesystem

	FileSystemConfig
}

func NewFileSystem(log *logrus.Logger, c *FileSystemConfig, inMem bool) *FileSystem {
	var fs billy.Filesystem
	if inMem {
		fs = memfs.New()
	} else {
		// TODO: Adapt disk-location based on platform.
		fs = osfs.New("/")
	}

	return &FileSystem{
		log:              log,
		storage:          fs,
		FileSystemConfig: *c,
	}
}

func (s *FileSystem) Path(b config.Binary) (string, error) {
	localPath := s.instantiateTemplate(b, s.FilePathTemplate)
	if _, err := s.storage.Stat(localPath); errors.Is(err, os.ErrNotExist) {
		return "", ErrNotFound
	} else if err != nil {
		s.log.WithError(err).Errorf("Unable to check local presence of %v.", b)
		return "", err
	}
	return localPath, nil
}

func (s *FileSystem) Fetch(b config.Binary) ([]byte, error) {
	p := s.instantiateTemplate(b, s.FilePathTemplate)
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return s.extractFromArchive(raw, p, b)
}

func (s *FileSystem) Store(b config.Binary, content []byte) error {
	localPath := s.instantiateTemplate(b, s.FilePathTemplate)
	if _, err := s.storage.Stat(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.log.WithError(err).Errorf("Unable to check for the presence of %v.", b)
		return err
	} else if err == nil {
		s.log.WithError(err).Errorf("Can not store %v as it is already present.", b)
		return errFailed
	}

	w, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o755)
	if err != nil {
		return nil
	}
	_, err = io.Copy(w, bytes.NewReader(content))
	return err
}
