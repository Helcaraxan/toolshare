package state

import (
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
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
	RecommendVersion(binary config.Binary) error
	AddVersions(binaries ...config.Binary) error
	DeleteVersions(binaries ...config.Binary) error
}

func NewCache(log *zap.Logger, localRoot string, settings *config.State) Cache {
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
	LastRefresh time.Time `json:"last_refresh"`
}

type toolState struct {
	Name               string   `json:"name"`
	RecommendedVersion string   `json:"recommended_version"`
	Versions           []string `json:"versions"`
}
