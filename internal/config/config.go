package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/Helcaraxan/toolshare/internal/state"
	"github.com/Helcaraxan/toolshare/internal/storage"
)

const (
	DriverName = "imp-tool"

	configFileName = DriverName + ".yaml"
)

type Settings struct {
	DisallowUnpinned bool
	Root             string
	State            *state.Settings
	Storage          *storage.Settings
}

func Init(log *logrus.Logger, flags *pflag.FlagSet) (*Settings, error) {
	s := &Settings{}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	configPaths, err := getConfigPaths(log)
	if err != nil {
		return nil, err
	}

	if err = v.BindPFlags(flags); err != nil {
		log.WithError(err).Error("Failed to")
		return nil, err
	}
	setConfigDefaults(v)

	for _, configPath := range configPaths {
		if configPath == "" {
			continue
		}
		v.SetConfigFile(configPath)
		if err = v.MergeInConfig(); err != nil {
			log.WithError(err).Error("Failed to read in standard config.")
			return nil, err
		}
	}
	v.AutomaticEnv()

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:  nil,
		ErrorUnused: true,
		ZeroFields:  true,
	})
	if err != nil {
		log.WithError(err).Error("Failed to initialise configuration unmarshalling.")
		return nil, err
	} else if err = d.Decode(v.AllSettings()); err != nil {
		log.WithError(err).Error("Failed to unmarshal configuration.")
		return nil, err
	}
	log.Debugf("Determined the configuration to use: %+v", s)
	return s, nil
}

func setConfigDefaults(v *viper.Viper) {
	v.SetDefault("disallowUnpinned", false)
	v.SetDefault("storageRoot", getLocalStorageRoot())
}

func getConfigPaths(log *logrus.Logger) ([]string, error) {
	var configPaths []string
	if p := systemConfigPath(); p != "" {
		configPaths = append(configPaths, p)
	}
	if p := userConfigPath(); p != "" {
		configPaths = append(configPaths, p)
	}
	ctxPaths, err := contextConfigPaths(log)
	if err != nil {
		return nil, err
	}
	configPaths = append(configPaths, ctxPaths...)
	return configPaths, nil
}

func systemConfigPath() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join("/etc", DriverName, configFileName)
	case "windows":
		return filepath.Join("PROGRAMDATA", DriverName, configFileName)
	default:
		return ""
	}
}

func userConfigPath() string {
	switch runtime.GOOS {
	case "linux":
		if configPath, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
			return filepath.Join(configPath, DriverName, configFileName)
		}
		fallthrough
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), ".config", DriverName, configFileName)
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName, configFileName)
	default:
		return ""
	}
}

func contextConfigPaths(log *logrus.Logger) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.WithError(err).Error("Failed to determine current working directory.")
		return nil, errFailed
	}

	var paths []string
	for {
		path := filepath.Join(wd, configFileName)
		if _, err = os.Stat(path); err == nil {
			paths = append(paths, path)
		}

		if wd == "/" {
			break
		}
		wd = filepath.Dir(wd)
	}
	return paths, nil
}

func getLocalStorageRoot() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join(os.Getenv("HOME"), "."+DriverName)
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName)
	default:
		return ""
	}
}

var errFailed = errors.New("failed")
