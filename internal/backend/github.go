package backend

import (
	"bytes"
	"context"
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
	if c.GitHubBaseURL == "" {
		client = github.NewClient(http.DefaultClient)
	} else {
		client, err = github.NewEnterpriseClient(c.GitHubBaseURL, c.GitHubBaseURL, http.DefaultClient)
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
	rs := strings.Split(s.GitHubSlug, "/")
	if len(rs) != 2 {
		log.Error("Invalid repo slug.", zap.String("slug", s.GitHubSlug))
		return nil, fmt.Errorf("repo slug %q is invalid as it does not contain an owner and repo name", s.GitHubSlug)
	}
	page := 1
	var gr *github.RepositoryRelease
	for {
		releases, resp, listErr := s.client.Repositories.ListReleases(ctx, rs[0], rs[1], &github.ListOptions{
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
	dl, _, err := s.client.Repositories.DownloadReleaseAsset(ctx, rs[0], rs[1], a.GetID(), http.DefaultClient)
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
