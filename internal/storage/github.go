package storage

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/types"
)

type githubStorage struct {
	log     *logrus.Logger
	timeout time.Duration

	repoSlug             string
	releaseAssetTemplate string
	archivePathTemplate  string
}

func (s *githubStorage) Fetch(b types.Binary, targetPath string) error {
	return nil
}

func (s *githubStorage) Store(b types.Binary, path string) (err error) {
	s.log.Error("Cannot perform 'store' operations on a GitHub backend.")
	return errFailed
}
