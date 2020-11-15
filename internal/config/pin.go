package config

type PinFile struct {
	PinnedTools []Pin `yaml:"pinned_tools,pinnedTools"`
}

type Pin struct {
	Tool    string `yaml:"tool"`
	Version string `yaml:"version"`
}
