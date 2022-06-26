package backend

import (
	"errors"
	"path/filepath"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

type Cache interface {
	Get(b tool.Binary) (string, error)
}

type Backend interface {
	Fetch(b tool.Binary, targetPath string) error
	Store(b tool.Binary, sourcePath string) error
}

var (
	// To guarantee that implementations remain compatible with the interface.
	_ Cache = &localStorage{}

	_ Backend = &gcsStorage{}
	_ Backend = &githubStorage{}
	_ Backend = &localStorage{}
	_ Backend = &s3Storage{}
)

func NewCache(log *logrus.Logger, localRoot string, settings *config.Storage) Cache {
	cache := &localStorage{
		log:     log,
		storage: osfs.New(filepath.Join(localRoot, "binaries")),
	}

	if settings.Local != "" {
		cache.remote = &localStorage{
			log:     log,
			storage: osfs.New(settings.Local),
		}
		return cache
	}
	return nil
}

var errFailed = errors.New("failed")
