package config

import "time"

type Global struct {
	Root string `yaml:"root"`

	RemoteCache *RemoteCache `yaml:"remote_cache"`
	State       *State       `yaml:"state"`

	ForcePinned    bool `yaml:"force_pinned"`
	DisableSources bool `yaml:"disable_sources"`
}

type RemoteCache struct {
	cache
}

type State struct {
	Type            string        `yaml:"type"`
	Local           string        `yaml:"local"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}
