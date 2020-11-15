package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/improbable/toolshare/internal/state"
	"github.com/improbable/toolshare/internal/storage"
)

const DriverName = "imp-tool"

type Settings struct {
	DisallowUnpinned bool
	Root             string
	State            *state.Settings
	Storage          *storage.Settings
}

func Init(log *logrus.Logger) (*Settings, error) {
	s := &Settings{}

	v := viper.New()
	v.SetDefault("disallow_unpinned", false)
	v.SetDefault("root", getRoot())
	state.InitConfiguration(v, "state")
	storage.InitConfiguration(v, "storage")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if runtime.GOOS == "windows" {
		v.AddConfigPath(filepath.Join(os.Getenv("PROGRAMDATA"), DriverName))
		v.AddConfigPath(filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName))
	} else {
		v.AddConfigPath(filepath.Join("/etc", DriverName))
		v.AddConfigPath(filepath.Join(os.Getenv("HOME"), "."+DriverName))
	}
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debug("No configuration file was found.")
		} else if err, ok = err.(viper.ConfigParseError); ok {
			log.WithError(err).Error("Failed to parse the configuration file.")
			return nil, err
		} else {
			log.WithError(err).Error("Could not read the configuration file.")
			return nil, err
		}
	} else {
		if err = v.Unmarshal(s); err != nil {
			log.WithError(err).Error("Failed to unmarshal the configuration data.")
			return nil, err
		}
	}
	log.Debugf("Determined the configuration to use: %+v", s)

	return s, nil
}

func getRoot() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName)
	default:
		return filepath.Join(os.Getenv("HOME"), "."+DriverName)
	}
}
