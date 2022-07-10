package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type GitHubConfig struct {
	CommonConfig

	GitHubSlug                 string `yaml:"github_slug"`
	GitHubReleaseAssetTemplate string `yaml:"github_release_asset_template"`
	GitHubBaseURL              string `yaml:"github_base_url"`
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

	GitHubConfig
}

func NewGitHub(logBuilder logger.Builder, c *GitHubConfig) *GitHub {
	return &GitHub{
		log:          logBuilder.Domain(logger.GitHubDomain).With(zap.Stringer("github-repo", c)),
		timeout:      time.Minute,
		GitHubConfig: *c,
	}
}

func (s *GitHub) Fetch(b config.Binary) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	log := s.log.With(zap.Stringer("tool", b))

	var err error
	var c *github.Client
	if s.GitHubBaseURL == "" {
		c = github.NewClient(http.DefaultClient)
	} else {
		c, err = github.NewEnterpriseClient(s.GitHubBaseURL, s.GitHubBaseURL, http.DefaultClient)
		if err != nil {
			log.Error("Failed to initialise new GitHub Enterprise client.", zap.Error(err))
			return nil, err
		}
	}

	rs := strings.Split(s.GitHubSlug, "/")
	if len(rs) != 2 {
		return nil, fmt.Errorf("repo slug %q is invalid as it does not contain an owner and repo name", s.GitHubSlug)
	}
	page := 1
	var gr *github.RepositoryRelease
	for {
		releases, resp, listErr := c.Repositories.ListReleases(ctx, rs[0], rs[1], &github.ListOptions{
			Page:    page,
			PerPage: 50,
		})
		if listErr != nil {
			return nil, fmt.Errorf("unable to request releases page %d for %q: %w", page, s.GitHubSlug, listErr)
		} else if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to list releases page %d for %q: %s", page, s.GitHubSlug, resp.Status)
		}
		log.Debug("Listing GitHub releases", zap.Int("release-count", len(releases)))

		for _, r := range releases {
			if r.GetName() == b.Version {
				gr = r
				log.Debug("Found targeted release.")
				break
			}
		}
		if gr != nil || page == resp.LastPage {
			break
		}
		page = resp.NextPage
	}

	if gr == nil {
		log.Error("The targeted release was not found within the GitHub repository.")
		return nil, fmt.Errorf("repository %q does not have a release named %q", s.GitHubSlug, b.Version)
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
		return nil, fmt.Errorf("release %q of repository %q does not have an asset named %q", b.Version, s.GitHubSlug, assetName)
	}

	log.Debug("Downloading the release asset.")
	dl, _, err := c.Repositories.DownloadReleaseAsset(ctx, rs[0], rs[1], a.GetID(), http.DefaultClient)
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
