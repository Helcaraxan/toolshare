package environment

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/Helcaraxan/toolshare/internal/backend"
	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type environmentSpec struct {
	Pins    map[string]string  `json:"pins"`
	Sources map[string]*Source `json:"sources"`
}

type Environment map[string]ToolRegistration

type ToolRegistration struct {
	Source      *Source
	SourceFile  string
	Version     string
	VersionFile string
}

func GetEnvironment(conf *config.Global, env Environment) error {
	envFileName := fmt.Sprintf("%s.yaml", config.DriverName)

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
	for _, p := range config.AllDirs() {
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

		if err = mergeEnvironment(conf, env, p, raw); err != nil {
			return err
		}
	}

	if !conf.ForcePinned && conf.State != nil { //nolint: staticcheck // Requires further implementation
		// TODO.
	}
	return nil
}

func mergeEnvironment(conf *config.Global, env Environment, path string, content []byte) error {
	// We should preferably set the yaml.Strict() option on the decoder. This is currently not possible due to the
	// goccy/go-yaml library not supporting partial unmarshalling in combination with yaml.Strict(). Setting the option
	// would currently result in not being able to decode anything as we use embedded structs to account for the
	// different backends and their specific configuration options.
	dec := yaml.NewDecoder(bytes.NewReader(content))

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
	if conf.DisableSources {
		return nil
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

func (e Environment) Source(logBuilder logger.Builder, tool string) backend.Storage {
	sc := e[tool].Source
	if sc == nil {
		return nil
	}

	switch {
	case sc.FileSystemConfig != nil:
		return backend.NewFileSystem(logBuilder, sc.FileSystemConfig)
	case sc.GCSConfig != nil:
		return backend.NewGCS(logBuilder, sc.GCSConfig)
	case sc.GitHubConfig != nil:
		return backend.NewGitHub(logBuilder, sc.GitHubConfig)
	case sc.HTTPSConfig != nil:
		return backend.NewHTTPS(logBuilder, sc.HTTPSConfig)
	case sc.S3Config != nil:
		return backend.NewS3(logBuilder, sc.S3Config)
	default:
		return nil
	}
}
