package state

import (
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

var (
	// To guarantee that implementations remain compatible with the interface.
	_ Cache = &fileSystem{}

	_ State = &git{}
	_ State = &http{}
	_ State = &fileSystem{}
)

type Cache interface {
	AvailableTools() ([]string, error)
	AvailableVersions(tool string) ([]string, error)
	RecommendedVersion(tool string) (string, error)
	Refresh(force bool) error
}

type State interface {
	Fetch(target billy.Filesystem) error
	RecommendVersion(binary tool.Binary) error
	AddVersions(binaries ...tool.Binary) error
	DeleteVersions(binaries ...tool.Binary) error
}

func NewCache(log *logrus.Logger, localRoot string, settings *config.State) Cache {
	cache := &fileSystem{
		log:             log,
		refreshInterval: settings.RefreshInterval,
		storage:         osfs.New(localRoot),
	}

	if settings.Local != "" {
		cache.remote = &fileSystem{
			log:     log,
			storage: osfs.New(settings.Local),
		}
		return cache
	}

	return cache
}

type refreshState struct {
	LastRefresh time.Time `yaml:"last_refresh"`
}

type toolState struct {
	Name               string   `yaml:"name"`
	RecommendedVersion string   `yaml:"recommended_version"`
	Versions           []string `yaml:"versions"`
}
