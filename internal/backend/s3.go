package backend

import (
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

type s3Storage struct {
	log *logrus.Logger

	config.URLSource
}

func (s *s3Storage) Get(b tool.Binary) (string, error) {
	s.log.Error("Unimplemented.")
	return "", errFailed
}

func (s *s3Storage) Fetch(b tool.Binary, targetPath string) error {
	s.log.Error("Unimplemented.")
	return errFailed
}

func (s *s3Storage) Store(b tool.Binary, path string) error {
	s.log.Error("Unimplemented.")
	return errFailed
}
