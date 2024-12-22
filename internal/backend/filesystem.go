package backend

import (
	"errors"
	"io"
	"os"
	"path/filepath"

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

	FileSystemConfig
}

func NewFileSystem(logBuilder logger.Builder, c *FileSystemConfig) *FileSystem {
	return &FileSystem{
		log:              logBuilder.Domain(logger.FileSystemDomain),
		FileSystemConfig: *c,
	}
}

func (s *FileSystem) Path(b config.Binary) string {
	return s.instantiateTemplate(b, s.FilePathTemplate)
}

func (s *FileSystem) Fetch(b config.Binary) ([]byte, error) {
	p := s.instantiateTemplate(b, s.FilePathTemplate)
	log := s.log.With(zap.Stringer("tool", b), zap.String("local-path", p))
	fd, err := os.Open(p)
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

	if _, err := os.Stat(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("Unable to check for a pre-existing tool binary.", zap.Error(err))
		return err
	}

	toolDir := filepath.Dir(localPath)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		log.Error("Failed to create directory to store tool binary.", zap.Error(err))
		return err
	}

	toolBin, err := os.CreateTemp(toolDir, b.Tool+"-*")
	if err != nil {
		log.Error("Failed to open temporary file to store tool binary.", zap.Error(err))
		return err
	} else if _, err = toolBin.Write(content); err != nil {
		log.Error("Failed to write tool binary to temp file.", zap.Error(err))
		return err
	} else if err = toolBin.Close(); err != nil {
		log.Error("Failed to close temporary tool binary file.", zap.Error(err))
		return err
	} else if err = os.Chmod(toolBin.Name(), 0o755); err != nil {
			log.Error("Failed to make temporary tool binary file executable.", zap.Error(err))
			return err
		}

	if err = os.Rename(toolBin.Name(), localPath); err != nil {
		log.Error("Failed to move temporary tool binary to final path.", zap.Error(err))
		return err
	}
	log.Debug("Successfully stored tool binary.")
	return nil
}
