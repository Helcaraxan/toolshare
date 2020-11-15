package state

import (
	"fmt"
	"sort"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sirupsen/logrus"

	"github.com/improbable/toolshare/internal/types"
)

type gitState struct {
	log *logrus.Logger
	url string
}

func (s *gitState) Fetch(target billy.Filesystem) error {
	_, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &localState{
		log:     s.log,
		storage: state,
	}
	return tempState.Fetch(target)
}

func (s *gitState) RecommendVersion(binary types.Binary) error {
	repo, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &localState{
		log:     s.log,
		storage: state,
	}
	if err = tempState.RecommendVersion(binary); err != nil {
		return err
	}

	return s.commitAndPush(repo, fmt.Sprintf("Recommend version %q for %q.", binary.Version, binary.Tool))
}

func (s *gitState) AddVersions(binaries ...types.Binary) error {
	if len(binaries) == 0 {
		return nil
	}

	repo, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &localState{
		log:     s.log,
		storage: state,
	}
	if err = tempState.AddVersions(binaries...); err != nil {
		return err
	}

	var msgElts []string
	for _, binary := range binaries {
		msgElts = append(msgElts, fmt.Sprintf("%s@%s", binary.Tool, binary.Version))
	}
	sort.Strings(msgElts)

	return s.commitAndPush(repo, fmt.Sprintf("Added tool versions.\n%v", msgElts))
}

func (s *gitState) DeleteVersions(binaries ...types.Binary) error {
	if len(binaries) == 0 {
		return nil
	}

	repo, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &localState{
		log:     s.log,
		storage: state,
	}
	if err = tempState.DeleteVersions(binaries...); err != nil {
		return err
	}

	var msgElts []string
	for _, binary := range binaries {
		msgElts = append(msgElts, fmt.Sprintf("%s@%s", binary.Tool, binary.Version))
	}
	sort.Strings(msgElts)

	return s.commitAndPush(repo, fmt.Sprintf("Deleted tool versions.\n%v", msgElts))
}

func (s *gitState) createLocalCheckout() (repo *git.Repository, state billy.Filesystem, err error) {
	storage := memory.NewStorage()
	state = memfs.New()

	repo, err = git.Clone(storage, state, &git.CloneOptions{
		URL:           s.url,
		ReferenceName: plumbing.Master,
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		s.log.WithError(err).Errorf("Failed to clone %q.", s.url)
		return nil, nil, err
	}
	return repo, state, nil
}

func (s *gitState) commitAndPush(repo *git.Repository, message string) error {
	wt, err := repo.Worktree()
	if err != nil {
		s.log.WithError(err).Error("Failed to determine the git worktree.")
		return err
	}

	if err = wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		s.log.WithError(err).Error("Failed to stage all modified state files.")
		return err
	}

	if _, err = wt.Commit(message, &git.CommitOptions{}); err != nil {
		s.log.WithError(err).Error("Failed to commit modified state files.")
		return err
	}

	if err = repo.Push(&git.PushOptions{}); err != nil {
		s.log.WithError(err).Error("Failed to push state file changes to the remote state.")
		return err
	}
	return nil
}
