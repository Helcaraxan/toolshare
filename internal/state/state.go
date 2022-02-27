package state

import (
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Helcaraxan/toolshare/internal/types"
)

var (
	// To guarantee that implementations remain compatible with the interface.
	_ Cache = &localState{}

	_ State = &gitState{}
	_ State = &httpState{}
	_ State = &localState{}
)

type Cache interface {
	AvailableTools() ([]string, error)
	AvailableVersions(tool string) ([]string, error)
	RecommendedVersion(tool string) (string, error)
	Refresh(force bool) error
}

type State interface {
	Fetch(target billy.Filesystem) error
	RecommendVersion(binary types.Binary) error
	AddVersions(binaries ...types.Binary) error
	DeleteVersions(binaries ...types.Binary) error
}

type Settings struct {
	RefreshInterval time.Duration `yaml:"refreshInterval"`
	Local           string        `yaml:"local"`
	Type            string        `yaml:"type"`
}

func InitConfiguration(v *viper.Viper, prefix string) {
	v.SetDefault(prefix+".type", "git")
}

func NewCache(log *logrus.Logger, localRoot string, settings *Settings) Cache {
	cache := &localState{
		log:             log,
		refreshInterval: settings.RefreshInterval,
		storage:         osfs.New(localRoot),
	}

	if settings.Local != "" {
		cache.remote = &localState{
			log:     log,
			storage: osfs.New(settings.Local),
		}
		return cache
	}

	return cache
}

type refreshState struct {
	LastRefresh time.Time `yaml:"lastRefresh"`
}

type toolState struct {
	Name               string   `yaml:"name"`
	RecommendedVersion string   `yaml:"recommendedVersion"`
	Versions           []string `yaml:"versions"`
}
