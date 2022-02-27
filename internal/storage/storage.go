package storage

import (
	"errors"
	"path/filepath"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Helcaraxan/toolshare/internal/types"
)

type Cache interface {
	Get(b types.Binary) (string, error)
}

type Storage interface {
	Fetch(b types.Binary, targetPath string) error
	Store(b types.Binary, sourcePath string) error
}

var (
	// To guarantee that implementations remain compatible with the interface.
	_ Cache = &localStorage{}

	_ Storage = &gcsStorage{}
	_ Storage = &githubStorage{}
	_ Storage = &localStorage{}
	_ Storage = &s3Storage{}
)

type Settings struct {
	Local string
	Auth  string
}

func InitConfiguration(v *viper.Viper, prefix string) {
	v.SetDefault(prefix+".auth", "oidc")
}

func NewCache(log *logrus.Logger, localRoot string, settings *Settings) Cache {
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
