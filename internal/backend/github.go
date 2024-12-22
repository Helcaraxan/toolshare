package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

var (
	ErrGitHubAPIError            = errors.New("github api error")
	ErrInvalidGitHubSlug         = errors.New("repo slug is invalid")
	ErrUnknownGitHubRelease      = errors.New("github release does not exist")
	ErrUnknownGitHubReleaseAsset = errors.New("github release does not contain asset")
)

type GitHubConfig struct {
	CommonConfig

	GitHubSlug                 string `json:"github_slug"`
	GitHubReleaseAssetTemplate string `json:"github_release_asset_template"`
	GitHubBaseURL              string `json:"github_base_url"`
}

func (c GitHubConfig) String() string {
	githubBase := c.GitHubBaseURL
	if githubBase == "" {
		githubBase = "github.com"
	}
	return fmt.Sprintf("%s/%s", githubBase, c.GitHubSlug)
}

type GitHub struct {
	log     *zap.Logger
	timeout time.Duration
	client  *github.Client

	GitHubConfig
}

func NewGitHub(logBuilder logger.Builder, c *GitHubConfig) *GitHub {
	var (
		err    error
		client *github.Client
	)

	log := logBuilder.Domain(logger.GitHubDomain).With(zap.Stringer("github-repo", c))
	client = github.NewClient(http.DefaultClient)
	if c.GitHubBaseURL != "" {
		client, err = client.WithEnterpriseURLs(c.GitHubBaseURL, c.GitHubBaseURL)
		if err != nil {
			log.Error("Failed to initialise new GitHub Enterprise client.", zap.Error(err))
			panic(err)
		}
	}

	return &GitHub{
		log:          log,
		timeout:      time.Minute,
		client:       client,
		GitHubConfig: *c,
	}
}

func (s *GitHub) Fetch(b config.Binary) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	log := s.log.With(zap.Stringer("tool", b))
	repoSlug := strings.Split(s.GitHubSlug, "/")
	if len(repoSlug) != 2 {
		log.Error("Invalid repo slug.", zap.String("slug", s.GitHubSlug))
		return nil, ErrInvalidGitHubSlug
	}

	gr, err := s.getRelease(ctx, log, repoSlug, b.Version)
	if err != nil {
		return nil, err
	}

	assetName := s.instantiateTemplate(b, s.GitHubReleaseAssetTemplate)
	log = log.With(zap.String("release-asset", assetName))

	var a *github.ReleaseAsset
	for _, c := range gr.Assets {
		if c.GetName() == assetName {
			log.Debug("Found targeted release asset.")
			a = c
			break
		}
	}
	if a == nil {
		log.Error("The targeted release asset was not found within the release.")
		return nil, ErrUnknownGitHubReleaseAsset
	}

	log.Debug("Downloading the release asset.")
	dl, _, err := s.client.Repositories.DownloadReleaseAsset(ctx, repoSlug[0], repoSlug[1], a.GetID(), http.DefaultClient)
	if err != nil {
		log.Error("Could not get download handle for the release asset.", zap.Error(err))
		return nil, fmt.Errorf("failed to get link to asset %q from release %q in repository %q: %w", assetName, b.Version, s.GitHubSlug, err)
	}
	buf := &bytes.Buffer{}
	if _, err = io.Copy(buf, dl); err != nil {
		log.Error("Download failed.", zap.Error(err))
		return nil, fmt.Errorf("failed to download asset %q from release %q in repository %q: %w", assetName, b.Version, s.GitHubSlug, err)
	}
	log.Debug("Finished downloading the release asset.")

	return s.extractFromArchive(log, buf.Bytes(), assetName, b)
}

func (s *GitHub) Store(_ config.Binary, _ []byte) error {
	s.log.Error("Cannot perform 'store' operations on a GitHub backend.")
	return errFailed
}

func (s *GitHub) getRelease(ctx context.Context, log *zap.Logger, repoSlug []string, version string) (*github.RepositoryRelease, error) {
	page := 1
	var gr *github.RepositoryRelease
	for {
		releases, resp, listErr := s.client.Repositories.ListReleases(ctx, repoSlug[0], repoSlug[1], &github.ListOptions{
			Page:    page,
			PerPage: 50,
		})
		if listErr != nil {
			return nil, fmt.Errorf("unable to request releases page %d for %q: %w", page, s.GitHubSlug, listErr)
		} else if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to list releases page %d for %q: %s: %w", page, s.GitHubSlug, resp.Status, ErrGitHubAPIError)
		}
		log.Debug("Retrieved GitHub releases.", zap.Int("release-count", len(releases)))
		for _, r := range releases {
			if strings.TrimLeft(r.GetTagName(), "v") == strings.TrimLeft(version, "v") {
				gr = r
				log.Debug("Found targeted release.")
				break
			}
		}
		if gr != nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	if gr == nil {
		log.Error("The targeted release was not found within the GitHub repository.")
		return nil, ErrUnknownGitHubRelease
	}
	return gr, nil
}
