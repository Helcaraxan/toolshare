package state

import (
	"github.com/go-git/go-billy/v5"

	"github.com/Helcaraxan/toolshare/internal/tool"
)

type httpState struct {
	Root string
	URL  string
}

func (s *httpState) Fetch(target billy.Filesystem) error {
	return nil
}

func (s *httpState) RecommendVersion(binary tool.Binary) error {
	return nil
}

func (s *httpState) AddVersions(binaries ...tool.Binary) error {
	return nil
}

func (s *httpState) DeleteVersions(binaries ...tool.Binary) error {
	return nil
}
