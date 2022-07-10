package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	DriverName = "toolshare"

	configFileName = DriverName + "_conf.yaml"
)

type Global struct {
	ForcePinned    bool `yaml:"force_pinned"`
	DisableSources bool `yaml:"disable_sources"`

	RemoteCache *Cache `yaml:"remote_cache"`
	State       *State `yaml:"state"`
}

type State struct {
	Type            string        `yaml:"type"`
	Local           string        `yaml:"local"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}

func Parse(log *zap.Logger, conf *Global) error {
	if conf == nil {
		return errors.New("can not parse configuration into nil struct")
	}

	for _, p := range AllDirs() {
		raw, err := os.ReadFile(filepath.Join(p, configFileName))
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}

		dec := yaml.NewDecoder(bytes.NewBuffer(raw))
		if err = dec.Decode(conf); err != nil {
			return err
		}
	}
	log.Sugar().Debugf("Parsed configuration:\n%+v", spew.Sdump(conf))
	return nil
}

func StorageDir() string {
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

func AllDirs() []string {
	// We need the config directories in reverse-order of priority such that we can safely unmarshal
	// them in order into the same target struct and guarantee the expected semantics.
	var dirs []string
	if p := UserDir(); p != "" {
		dirs = append(dirs, p)
	}
	if p := SystemDir(); p != "" {
		dirs = append(dirs, p)
	}
	return dirs
}

func SystemDir() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join("/etc", DriverName)
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), DriverName)
	default:
		panic("unsupported platform")
	}
}

func UserDir() string {
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
