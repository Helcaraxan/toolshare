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

type environmentSpec struct {
	Pins    map[string]string  `yaml:"pins"`
	Sources map[string]*Source `yaml:"sources"`
}

type Environment map[string]ToolRegistration

type ToolRegistration struct {
	Source      *Source
	SourceFile  string
	Version     string
	VersionFile string
}

func GetEnvironment(conf *config.Global, env Environment) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var candidatePaths []string
	for {
		candidatePaths = append(candidatePaths, filepath.Join(cwd, "."+envFileName))
		if cwd == filepath.Dir(cwd) {
			break
		}
		cwd = filepath.Dir(cwd)
	}
	for _, p := range config.GetConfigDirs() {
		candidatePaths = append(candidatePaths, filepath.Join(p, envFileName))
	}

	for _, p := range candidatePaths {
		var raw []byte
		raw, err = os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}

		if err = mergeEnvironment(env, p, raw); err != nil {
			return err
		}
	}

	if !conf.ForcePinned && conf.State != nil {
		// TODO.
	}
	return nil
}

func mergeEnvironment(env Environment, path string, content []byte) error {
	dec := yaml.NewDecoder(bytes.NewReader(content))
	dec.KnownFields(true)

	var newEnv environmentSpec
	if err := dec.Decode(&newEnv); err != nil {
		return err
	}

	// For both pins and sources we only add tool settings if there are none available yet.
	for tool, version := range newEnv.Pins {
		r := env[tool]
		if r.Version == "" {
			r.Version = version
			r.VersionFile = path
			env[tool] = r
		}
	}
	for tool, source := range newEnv.Sources {
		r := env[tool]
		if r.Source == nil {
			r.Source = source
			r.SourceFile = path
			env[tool] = r
		}
	}
	return nil
}

func (e Environment) Source(log *logrus.Logger, tool string) backend.Storage {
	r, ok := e[tool]
	if !ok {
		return nil
	}
	sc := r.Source

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
