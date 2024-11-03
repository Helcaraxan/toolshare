package backend

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type FileSystemConfig struct {
	CommonConfig

	FilePathTemplate string `json:"file_path_template"`
}

func (c FileSystemConfig) String() string {
	return c.FilePathTemplate
}

type FileSystem struct {
	log     *zap.Logger
	storage billy.Filesystem

	FileSystemConfig
}

func NewFileSystem(logBuilder logger.Builder, c *FileSystemConfig, inMem bool) *FileSystem {
	var fs billy.Filesystem
	if inMem {
		fs = memfs.New()
	} else {
		fs = osfs.New("/")
	}

	return &FileSystem{
		log:              logBuilder.Domain(logger.FileSystemDomain),
		storage:          fs,
		FileSystemConfig: *c,
	}
}

func (s *FileSystem) Path(b config.Binary) string {
	return s.instantiateTemplate(b, s.FilePathTemplate)
}

func (s *FileSystem) Fetch(b config.Binary) ([]byte, error) {
	p := s.instantiateTemplate(b, s.FilePathTemplate)
	log := s.log.With(zap.Stringer("tool", b), zap.String("local-path", p))
	fd, err := s.storage.Open(p)
	if err != nil {
		log.Error("Failed to open tool binary file.", zap.Error(err))
		return nil, err
	}
	raw, err := io.ReadAll(fd)
	if err != nil {
		log.Error("Failed to read content of tool binary file.", zap.Error(err))
		return nil, err
	}
	return s.extractFromArchive(log, raw, p, b)
}

func (s *FileSystem) Store(b config.Binary, content []byte) error {
	localPath := s.instantiateTemplate(b, s.FilePathTemplate)
	log := s.log.With(zap.Stringer("tool", b), zap.String("local-path", localPath))

	if _, err := s.storage.Stat(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("Unable to check for a pre-existing tool binary.", zap.Error(err))
		return err
	} else if err == nil {
		log.Error("Can not store binary as it is already present.", zap.Error(err))
		return errFailed
	}

	if err := s.storage.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	w, err := s.storage.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()

	_, err = io.Copy(w, bytes.NewReader(content))
	return err
}
