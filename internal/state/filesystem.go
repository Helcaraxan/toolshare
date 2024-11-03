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
	"github.com/goccy/go-yaml"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
)

const cacheStatusFile = "cache.status.yaml"

type fileSystem struct {
	log             *zap.Logger
	refreshInterval time.Duration
	remote          State
	storage         billy.Filesystem
}

func (s *fileSystem) AvailableTools() ([]string, error) {
	if err := s.Refresh(false); err != nil {
		s.log.Warn("Failed to refresh state cache.", zap.Error(err))
	}

	toolsDir, err := s.storage.ReadDir(".")
	if err != nil {
		s.log.Warn("Failed to read the state cache.", zap.Error(err))
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

func (s *fileSystem) AvailableVersions(toolName string) ([]string, error) {
	if err := s.Refresh(false); err != nil {
		s.log.Warn("Failed to refresh state cache.", zap.Error(err))
	}

	state, err := s.readToolState(toolName)
	if err != nil {
		return nil, err
	}

	sort.Strings(state.Versions)
	return state.Versions, nil
}

func (s *fileSystem) RecommendedVersion(toolName string) (string, error) {
	if err := s.Refresh(false); err != nil {
		s.log.Warn("Failed to refresh state cache.", zap.Error(err))
	}

	state, err := s.readToolState(toolName)
	if err != nil {
		return "", err
	}
	return state.RecommendedVersion, nil
}

func (s *fileSystem) Refresh(force bool) error {
	log := s.log.With(zap.String("status-file", filepath.Join(s.storage.Root(), cacheStatusFile)))

	stateFile, err := s.storage.OpenFile(cacheStatusFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0o644)
	if err != nil {
		log.Error("Failed to open state cache status file.", zap.Error(err))
		return err
	}
	defer stateFile.Close()

	stateContent, err := io.ReadAll(stateFile)
	if err != nil {
		log.Error("Unable to read state cache status file.", zap.Error(err))
		return err
	}

	var state refreshState
	if len(stateContent) > 0 {
		if err = yaml.Unmarshal(stateContent, &state); err != nil {
			log.Error("Unable to unmarshal state cache status file.", zap.Error(err))
			return err
		}
	}

	if !force && time.Now().Before(state.LastRefresh.Add(s.refreshInterval)) {
		log.Debug("Not refreshing state cache.", zap.Time("last-refresh", state.LastRefresh), zap.Duration("refresh-interval", s.refreshInterval))
		return nil
	}

	if err = s.remote.Fetch(s.storage); err != nil {
		return err
	}

	state.LastRefresh = time.Now()
	stateContent, err = yaml.Marshal(&state)
	if err != nil {
		log.Error("Unable to marshal new state cache status.", zap.Error(err))
		return err
	}

	if _, err = stateFile.Write(stateContent); err != nil {
		log.Error("Unable to update state cache status file.", zap.Error(err))
		return err
	}
	return nil
}

func (s *fileSystem) Fetch(target billy.Filesystem) error {
	log := s.log.With(zap.String("state-root", s.storage.Root()))
	stateFiles, err := s.storage.ReadDir("/")
	if err != nil {
		log.Error("Unable to read the content of the state folder.", zap.Error(err))
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
		log.Error("Unable to read content of target state.", zap.Error(err))
		return err
	}

	for _, info := range targetFiles {
		if info.IsDir() || filepath.Ext(info.Name()) != ".yaml" || info.Name() == cacheStatusFile {
			continue
		} else if copiedFiles[info.Name()] {
			continue
		}

		if err = target.Remove(info.Name()); err != nil {
			log.Error("Failed to clean up stale state file.", zap.String("state-file", filepath.Join(s.storage.Root(), info.Name())), zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *fileSystem) RecommendVersion(binary config.Binary) error {
	state, err := s.readToolState(binary.Tool)
	if err != nil {
		return err
	}

	state.RecommendedVersion = binary.Version

	return s.writeToolState(binary.Tool, state)
}

func (s *fileSystem) AddVersions(binaries ...config.Binary) error {
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

func (s *fileSystem) DeleteVersions(binaries ...config.Binary) error {
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

func (s *fileSystem) readToolState(toolName string) (*toolState, error) {
	log := s.log.With(zap.String("tool-state", filepath.Join(s.storage.Root(), toolName+".yaml")))

	stateFile, err := s.storage.Open(toolName + ".yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Error("No state file for tool available in state cache.")
		} else {
			log.Error("Unable to read tool state file.", zap.Error(err))
		}
		return nil, err
	}
	defer stateFile.Close()

	stateContent, err := io.ReadAll(stateFile)
	if err != nil {
		log.Error("Unable to read tool state file.", zap.Error(err))
		return nil, err
	}

	var state toolState
	if err = yaml.Unmarshal(stateContent, &state); err != nil {
		log.Error("Unable to unmarshal tool state file.", zap.Error(err))
		return nil, err
	}
	return &state, nil
}

func (s *fileSystem) writeToolState(toolName string, state *toolState) error {
	log := s.log.With(zap.String("tool-state", filepath.Join(s.storage.Root(), toolName+".yaml")))

	stateContent, err := yaml.Marshal(state)
	if err != nil {
		log.Error("Failed to marshal new tool state file content.", zap.Error(err))
		return err
	}

	stateFile, err := s.storage.OpenFile(toolName+".yaml.new", os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Error("Unable to open tool state file.", zap.Error(err))
		return err
	}
	defer stateFile.Close()

	if _, err = stateFile.Write(stateContent); err != nil {
		log.Error("Failed to write new tool state file content.", zap.Error(err))
		return err
	}

	if err = s.storage.Rename(stateFile.Name(), toolName+".yaml"); err != nil {
		log.Error("Unable to move temporary tool state file to permanent position.", zap.Error(err))
	}
	return nil
}
