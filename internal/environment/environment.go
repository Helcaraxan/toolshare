package environment

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/Helcaraxan/toolshare/internal/backend"
	"github.com/Helcaraxan/toolshare/internal/config"
)

type Environment struct {
	Pins    map[string]string `yaml:"pins"`
	Sources map[string]Source `yaml:"sources"`
}

func GetEnvironment() (*Environment, error) {
	env := &Environment{
		Pins:    map[string]string{},
		Sources: map[string]Source{},
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		path := filepath.Join(cwd, "toolshare.yaml")
		if _, err = os.Stat(path); err == nil {
			if err = mergeEnvironment(env, path); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if cwd == filepath.Dir(cwd) {
			break
		}
		cwd = filepath.Dir(cwd)
	}

	path := filepath.Join(config.GetLocalStorageRoot(), "global.yaml")
	if _, err = os.Stat(path); err == nil {
		if err = mergeEnvironment(env, path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return env, nil
}

func mergeEnvironment(env *Environment, path string) error {
	envRaw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(envRaw))
	dec.KnownFields(true)

	newEnv := Environment{Sources: map[string]Source{}}
	if err = dec.Decode(&newEnv); err != nil {
		return err
	}

	// For both pins and sources we only add tool settings if there are none available yet.
	for tool, version := range newEnv.Pins {
		if _, ok := env.Pins[tool]; !ok {
			env.Pins[tool] = version
		}
	}
	for tool, source := range newEnv.Sources {
		if _, ok := env.Sources[tool]; !ok {
			env.Sources[tool] = source
		}
	}
	return nil
}

func (e *Environment) Source(log *logrus.Logger, tool string) backend.Storage {
	sc, ok := e.Sources[tool]
	if !ok {
		return nil
	}

	switch {
	case sc.FileSystemConfig != nil:
		return backend.NewFileSystem(log, sc.FileSystemConfig, false)
	case sc.GCSConfig != nil:
		return backend.NewGCS(log, sc.GCSConfig)
	case sc.GitHubConfig != nil:
		return backend.NewGitHub(log, sc.GitHubConfig)
	case sc.HTTPSConfig != nil:
		return backend.NewHTTPS(log, sc.HTTPSConfig)
	case sc.S3Config != nil:
		return backend.NewS3(log, sc.S3Config)
	default:
		return nil
	}
}
