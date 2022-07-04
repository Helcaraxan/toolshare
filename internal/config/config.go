package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	DriverName = "toolshare"

	configFileName = DriverName + ".yaml"
)

func Parse(log *logrus.Logger, flags *pflag.FlagSet) (*Global, error) {
	viper.SetConfigType("yaml")

	setConfigDefaults()
	if err := viper.BindPFlags(flags); err != nil {
		log.WithError(err).Error("Failed to")
		return nil, err
	}

	for _, configPath := range getConfigPaths() {
		if configPath == "" {
			continue
		}
		viper.SetConfigFile(configPath)
		if err := viper.MergeInConfig(); err != nil {
			log.WithError(err).Error("Failed to read in standard config.")
			return nil, err
		}
	}
	viper.AutomaticEnv()

	base := &Global{}
	if err := viper.UnmarshalExact(base); err != nil {
		return nil, err
	}
	return base, nil
}

func SystemConfigPath() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join("/etc", DriverName)
	case "windows":
		return filepath.Join("PROGRAMDATA", DriverName)
	default:
		return ""
	}
}

func UserConfigPath() string {
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
		return ""
	}
}

func GetLocalStorageRoot() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join(os.Getenv("HOME"), "."+DriverName)
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName)
	default:
		return ""
	}
}

func setConfigDefaults() {
	viper.SetDefault("force_pinned", false)
	viper.SetDefault("root", GetLocalStorageRoot())
}

func getConfigPaths() []string {
	var configPaths []string
	if p := SystemConfigPath(); p != "" {
		configPaths = append(configPaths, filepath.Join(p, configFileName))
	}
	if p := UserConfigPath(); p != "" {
		configPaths = append(configPaths, filepath.Join(p, configFileName))
	}
	return configPaths
}
