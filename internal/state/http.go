package state

import (
	"github.com/go-git/go-billy/v5"

	"github.com/improbable/toolshare/internal/types"
)

type httpState struct {
	Root string
	URL  string
}

func (s *httpState) Fetch(target billy.Filesystem) error {
	return nil
}

func (s *httpState) RecommendVersion(binary types.Binary) error {
	return nil
}

func (s *httpState) AddVersions(binaries ...types.Binary) error {
	return nil
}

func (s *httpState) DeleteVersions(binaries ...types.Binary) error {
	return nil
}
