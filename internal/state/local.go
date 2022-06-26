package state

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/Helcaraxan/toolshare/internal/tool"
)

const cacheStatusFile = "cache.status.yaml"

type localState struct {
	log             *logrus.Logger
	refreshInterval time.Duration
	remote          State
	storage         billy.Filesystem
}

func (s *localState) AvailableTools() ([]string, error) {
	if err := s.Refresh(false); err != nil {
		s.log.WithError(err).Warn("Failed to refresh state cache.")
	}

	toolsDir, err := s.storage.ReadDir(".")
	if err != nil {
		s.log.WithError(err).Warn("Failed to read the state cache.")
		return nil, err
	}

	var tools []string
	for _, info := range toolsDir {
		if !info.IsDir() && filepath.Ext(info.Name()) == ".yaml" && info.Name() != cacheStatusFile {
			// We don't expect any folders or non-YAML files amongst cached state but we skip any we
			// encounter to be safe just like the cache status file.
			tools = append(tools, strings.TrimSuffix(info.Name(), ".yaml"))
		}
	}

	return tools, nil
}

func (s *localState) AvailableVersions(toolName string) ([]string, error) {
	if err := s.Refresh(false); err != nil {
		s.log.WithError(err).Warn("Failed to refresh state cache.")
	}

	state, err := s.readToolState(toolName)
	if err != nil {
		return nil, err
	}

	sort.Strings(state.Versions)
	return state.Versions, nil
}

func (s *localState) RecommendedVersion(toolName string) (string, error) {
	if err := s.Refresh(false); err != nil {
		s.log.WithError(err).Warn("Failed to refresh state cache.")
	}

	state, err := s.readToolState(toolName)
	if err != nil {
		return "", err
	}
	return state.RecommendedVersion, nil
}

func (s *localState) Refresh(force bool) error {
	var state refreshState

	stateFile, err := s.storage.OpenFile(cacheStatusFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0o644)
	if err != nil {
		s.log.WithError(err).Error("Failed to open state cache status file.")
		return err
	}
	defer stateFile.Close()

	stateContent, err := io.ReadAll(stateFile)
	if err != nil {
		s.log.WithError(err).Error("Unable to read state cache status file.")
		return err
	}

	if len(stateContent) > 0 {
		if err = yaml.Unmarshal(stateContent, &state); err != nil {
			s.log.WithError(err).Error("Unable to unmarshal state cache status file.")
			return err
		}
	}

	if !force && time.Now().Before(state.LastRefresh.Add(s.refreshInterval)) {
		s.log.Debugf("Not refreshing state cache. Last refresh was at %v and refresh interval is %v.", state.LastRefresh, s.refreshInterval)
		return nil
	}

	if err = s.remote.Fetch(s.storage); err != nil {
		return err
	}

	state.LastRefresh = time.Now()
	stateContent, err = yaml.Marshal(&state)
	if err != nil {
		s.log.WithError(err).Error("Unable to marshal new state cache status.")
		return err
	}

	if _, err = stateFile.Write(stateContent); err != nil {
		s.log.WithError(err).Errorf("Unable to update state cache status file %q.", stateFile)
		return err
	}
	return nil
}

func (s *localState) Fetch(target billy.Filesystem) error {
	stateFiles, err := s.storage.ReadDir("/")
	if err != nil {
		s.log.WithError(err).Error("Unable to read the state folder.")
		return err
	}

	var state *toolState
	copiedFiles := map[string]bool{}
	for _, info := range stateFiles {
		if info.IsDir() || filepath.Ext(info.Name()) != ".yaml" || info.Name() == cacheStatusFile {
			continue
		}

		toolName := strings.TrimSuffix(info.Name(), ".yaml")
		state, err = s.readToolState(toolName)
		if err != nil {
			return err
		}
		if err = s.writeToolState(toolName, state); err != nil {
			return err
		}

		copiedFiles[info.Name()] = true
	}

	targetFiles, err := target.ReadDir("/")
	if err != nil {
		s.log.WithError(err).Error("Unable to read content of target state.")
		return err
	}

	for _, info := range targetFiles {
		if info.IsDir() || filepath.Ext(info.Name()) != ".yaml" || info.Name() == cacheStatusFile {
			continue
		} else if copiedFiles[info.Name()] {
			continue
		}

		if err = target.Remove(info.Name()); err != nil {
			s.log.WithError(err).Errorf("Failed to clean up stale state file %q.", info.Name())
			return err
		}
	}
	return nil
}

func (s *localState) RecommendVersion(binary tool.Binary) error {
	state, err := s.readToolState(binary.Tool)
	if err != nil {
		return err
	}

	state.RecommendedVersion = binary.Version

	return s.writeToolState(binary.Tool, state)
}

func (s *localState) AddVersions(binaries ...tool.Binary) error {
	for _, binary := range binaries {
		state, err := s.readToolState(binary.Tool)
		if err != nil {
			return err
		}

		var exists bool
		for _, version := range state.Versions {
			if version == binary.Version {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		state.Versions = append(state.Versions, binary.Version)
		sort.Strings(state.Versions)

		if err = s.writeToolState(binary.Tool, state); err != nil {
			return err
		}
	}
	return nil
}

func (s *localState) DeleteVersions(binaries ...tool.Binary) error {
	for _, binary := range binaries {
		state, err := s.readToolState(binary.Tool)
		if err != nil {
			return err
		}

		for idx, version := range state.Versions {
			if version == binary.Version {
				state.Versions = append(state.Versions[:idx], state.Versions[idx+1:]...)
				break
			}
		}

		if err = s.writeToolState(binary.Tool, state); err != nil {
			return err
		}
	}
	return nil
}

func (s *localState) readToolState(toolName string) (*toolState, error) {
	stateFile, err := s.storage.Open(toolName + ".yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.log.Errorf("No state file for tool %q available in state cache.", toolName)
		} else {
			s.log.WithError(err).Errorf("Unable to read state file for tool %q.", toolName)
		}
		return nil, err
	}
	defer stateFile.Close()

	stateContent, err := io.ReadAll(stateFile)
	if err != nil {
		s.log.WithError(err).Errorf("Unable to read state file for tool %q.", toolName)
		return nil, err
	}

	var state toolState
	if err = yaml.Unmarshal(stateContent, &state); err != nil {
		s.log.WithError(err).Errorf("Unable to unmarshal state file for tool %q.", toolName)
		return nil, err
	}
	return &state, nil
}

func (s *localState) writeToolState(toolName string, state *toolState) error {
	stateContent, err := yaml.Marshal(state)
	if err != nil {
		s.log.WithError(err).Error("Failed to marshal new state file content.")
		return err
	}

	stateFile, err := s.storage.OpenFile(toolName+".yaml.new", os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		s.log.WithError(err).Errorf("Unable to open state file for tool %q.", toolName)
		return err
	}
	defer stateFile.Close()

	if _, err = stateFile.Write(stateContent); err != nil {
		s.log.WithError(err).Error("Failed to write new state file content.")
		return err
	}

	if err = s.storage.Rename(stateFile.Name(), toolName+".yaml"); err != nil {
		s.log.WithError(err).Error("Unable to move temporary state file to permanent position.")
	}
	return nil
}
