package config

type PinFile struct {
	PinnedTools []Pin `yaml:"pinnedTools"`
}

type Pin struct {
	Tool    string `yaml:"tool"`
	Version string `yaml:"version"`
}
