package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	DriverName = "tbd"

	configFileName = DriverName + ".yaml"
)

type Global struct {
	DisallowUnpinned bool
	Root             string
	State            *State
	Storage          *Storage
}

type State struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	Local           string        `yaml:"local"`
	Type            string        `yaml:"type"`
}

type Storage struct {
	Local string
}

func Init(log *logrus.Logger, flags *pflag.FlagSet) (*Global, error) {
	s := &Global{}

	v := viper.New()
	v.SetConfigName("tbd-conf")
	v.SetConfigType("yaml")

	configPaths := getConfigPaths()
	if err := v.BindPFlags(flags); err != nil {
		log.WithError(err).Error("Failed to")
		return nil, err
	}
	setConfigDefaults(v)

	for _, configPath := range configPaths {
		if configPath == "" {
			continue
		}
		v.SetConfigFile(configPath)
		if err := v.MergeInConfig(); err != nil {
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

func getConfigPaths() []string {
	var configPaths []string
	if p := systemConfigPath(); p != "" {
		configPaths = append(configPaths, p)
	}
	if p := userConfigPath(); p != "" {
		configPaths = append(configPaths, p)
	}
	return configPaths
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
