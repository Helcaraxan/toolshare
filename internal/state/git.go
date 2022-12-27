package state

import (
	"fmt"
	"sort"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type git struct {
	log *zap.Logger
	url string
}

func (s *git) Fetch(target billy.Filesystem) error {
	_, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &fileSystem{
		log:     s.log,
		storage: state,
	}
	return tempState.Fetch(target)
}

func (s *git) RecommendVersion(binary config.Binary) error {
	repo, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &fileSystem{
		log:     s.log,
		storage: state,
	}
	if err = tempState.RecommendVersion(binary); err != nil {
		return err
	}

	return s.commitAndPush(repo, fmt.Sprintf("Recommend version %q for %q.", binary.Version, binary.Tool))
}

func (s *git) AddVersions(binaries ...config.Binary) error {
	if len(binaries) == 0 {
		return nil
	}

	repo, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &fileSystem{
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

func (s *git) DeleteVersions(binaries ...config.Binary) error {
	if len(binaries) == 0 {
		return nil
	}

	repo, state, err := s.createLocalCheckout()
	if err != nil {
		return err
	}

	tempState := &fileSystem{
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

func (s *git) createLocalCheckout() (repo *gogit.Repository, state billy.Filesystem, err error) {
	storage := memory.NewStorage()
	state = memfs.New()

	repo, err = gogit.Clone(storage, state, &gogit.CloneOptions{
		URL:           s.url,
		ReferenceName: plumbing.Master,
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		s.log.Error("Failed to clone git state.", zap.String("remote-url", s.url), zap.Error(err))
		return nil, nil, err
	}
	return repo, state, nil
}

func (s *git) commitAndPush(repo *gogit.Repository, message string) error {
	wt, err := repo.Worktree()
	if err != nil {
		s.log.Error("Failed to determine the git worktree.", zap.Error(err))
		return err
	}

	if err = wt.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
		s.log.Error("Failed to stage all modified state files.", zap.Error(err))
		return err
	}

	if _, err = wt.Commit(message, &gogit.CommitOptions{}); err != nil {
		s.log.Error("Failed to commit modified state files.", zap.Error(err))
		return err
	}

	if err = repo.Push(&gogit.PushOptions{}); err != nil {
		s.log.Error("Failed to push state file changes to the remote state.", zap.Error(err))
		return err
	}
	return nil
}
