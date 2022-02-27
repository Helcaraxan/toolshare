package storage

import (
	"github.com/sirupsen/logrus"

	"github.com/improbable/toolshare/internal/types"
)

type s3Storage struct {
	log *logrus.Logger
}

func (s *s3Storage) Get(b types.Binary) (string, error) {
	s.log.Error("Unimplemented.")
	return "", errFailed
}

func (s *s3Storage) Fetch(b types.Binary, targetPath string) error {
	s.log.Error("Unimplemented.")
	return errFailed
}

func (s *s3Storage) Store(b types.Binary, path string) error {
	s.log.Error("Unimplemented.")
	return errFailed
}
