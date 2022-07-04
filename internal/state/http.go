package state

import (
	"github.com/go-git/go-billy/v5"

	"github.com/Helcaraxan/toolshare/internal/tool"
)

type http struct {
	Root string
	URL  string
}

func (s *http) Fetch(target billy.Filesystem) error {
	return nil
}

func (s *http) RecommendVersion(binary tool.Binary) error {
	return nil
}

func (s *http) AddVersions(binaries ...tool.Binary) error {
	return nil
}

func (s *http) DeleteVersions(binaries ...tool.Binary) error {
	return nil
}
