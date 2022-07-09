package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	DriverName = "toolshare"

	configFileName = DriverName + "_conf.yaml"
)

type Global struct {
	RemoteCache *RemoteCache `yaml:"remote_cache"`
	State       *State       `yaml:"state"`

	ForcePinned    bool `yaml:"force_pinned"`
	DisableSources bool `yaml:"disable_sources"`
}

type RemoteCache struct {
	cache
}

type State struct {
	Type            string        `yaml:"type"`
	Local           string        `yaml:"local"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}

func Parse(log *logrus.Logger, conf *Global) error {
	if conf == nil {
		return errors.New("can not parse configuration into nil struct")
	}

	for _, p := range GetConfigDirs() {
		raw, err := os.ReadFile(filepath.Join(p, configFileName))
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}

		if err = yaml.Unmarshal(raw, conf); err != nil {
			return err
		}
	}
	log.Debugf("Parsed configuration:\n%+v", spew.Sdump(conf))
	return nil
}

func GetStorageDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join("/var/tmp", DriverName)
	case "linux":
		return filepath.Join("/var/cache", DriverName)
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), DriverName)
	default:
		panic("unsupported platform")
	}
}

// We need the config directories in reverse-order of priority such that we can safely unmarshal
// them in order into the same target struct and guarantee the expected semantics.
func GetConfigDirs() []string {
	var dirs []string
	if p := GetUserConfigDir(); p != "" {
		dirs = append(dirs, p)
	}
	if p := GetSystemConfigDir(); p != "" {
		dirs = append(dirs, p)
	}
	return dirs
}

func GetSystemConfigDir() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join("/etc", DriverName)
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), DriverName)
	default:
		panic("unsupported platform")
	}
}

func GetUserConfigDir() string {
	switch runtime.GOOS {
	case "linux":
		if configPath, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
			return filepath.Join(configPath, DriverName)
		}
		fallthrough
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), ".config", DriverName)
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName)
	default:
		panic("unsupported platform")
	}
}
