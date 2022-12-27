package state

import (
	"github.com/go-git/go-billy/v5"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type http struct {
	Root string
	URL  string
}

func (s *http) Fetch(target billy.Filesystem) error {
	return nil
}

func (s *http) RecommendVersion(binary config.Binary) error {
	return nil
}

func (s *http) AddVersions(binaries ...config.Binary) error {
	return nil
}

func (s *http) DeleteVersions(binaries ...config.Binary) error {
	return nil
}
