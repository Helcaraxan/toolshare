package environment

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/Helcaraxan/toolshare/internal/backend"
	"github.com/Helcaraxan/toolshare/internal/config"
)

var envFileName = fmt.Sprintf("%s.yaml", config.DriverName)

type Environment struct {
	Pins    map[string]string `yaml:"pins"`
	Sources map[string]Source `yaml:"sources"`
}

func GetEnvironment(env *Environment) error {
	if env == nil {
		return errors.New("can not parse environment into nil struct")
	}
	env.Pins = map[string]string{}
	env.Sources = map[string]Source{}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var candidatePaths []string
	for {
		candidatePaths = append(candidatePaths, filepath.Join(cwd, envFileName))
		if cwd == filepath.Dir(cwd) {
			break
		}
		cwd = filepath.Dir(cwd)
	}
	candidatePaths = append(candidatePaths, config.GetConfigDirs()...)

	for _, p := range candidatePaths {
		var raw []byte
		raw, err = os.ReadFile(filepath.Join(p, envFileName))
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := mergeEnvironment(env, raw); err != nil {
			return err
		}
	}
	return nil
}

func mergeEnvironment(env *Environment, content []byte) error {
	dec := yaml.NewDecoder(bytes.NewReader(content))
	dec.KnownFields(true)

	newEnv := Environment{Sources: map[string]Source{}}
	if err := dec.Decode(&newEnv); err != nil {
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
