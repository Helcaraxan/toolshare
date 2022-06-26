package config

import (
	"bytes"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Environment struct {
	EnforcePins bool              `yaml:"enforce_pins"`
	Pins        map[string]string `yaml:"pins"`
	Sources     map[string]Source `yaml:"sources"`
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
		path := filepath.Join(cwd, ".tbd")
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

	path := filepath.Join(getLocalStorageRoot(), "global")
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

	newEnv := Environment{}
	if err = dec.Decode(&newEnv); err != nil {
		return err
	}

	env.EnforcePins = env.EnforcePins || newEnv.EnforcePins

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
