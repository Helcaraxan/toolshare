package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/tool"
)

type GitHubConfig struct {
	CommonConfig

	GitHubSlug                 string `yaml:"github_slug"`
	GitHubReleaseAssetTemplate string `yaml:"github_release_asset_template"`
	GitHubBaseURL              string `yaml:"github_base_url"`
}

type GitHub struct {
	log     *logrus.Logger
	timeout time.Duration

	GitHubConfig
}

func NewGitHub(log *logrus.Logger, c *GitHubConfig) *GitHub {
	return &GitHub{
		log:          log,
		timeout:      time.Minute,
		GitHubConfig: *c,
	}
}

func (s *GitHub) Fetch(b tool.Binary) ([]byte, error) {
	assetName, buf, err := s.getReleaseAsset(b)
	if err != nil {
		return nil, err
	}
	return s.extractFromArchive(buf.Bytes(), assetName, b)
}

func (s *GitHub) Store(_ tool.Binary, _ []byte) error {
	s.log.Error("Cannot perform 'store' operations on a GitHub backend.")
	return errFailed
}

func (s *GitHub) getReleaseAsset(b tool.Binary) (name string, content *bytes.Buffer, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	var c *github.Client
	if s.GitHubBaseURL == "" {
		c = github.NewClient(http.DefaultClient)
	} else {
		c, err = github.NewEnterpriseClient(s.GitHubBaseURL, s.GitHubBaseURL, http.DefaultClient)
		if err != nil {
			return "", nil, err
		}
	}

	rs := strings.Split(s.GitHubSlug, "/")
	if len(rs) != 2 {
		return "", nil, fmt.Errorf("repo slug %q is invalid as it does not contain an owner and repo name", s.GitHubSlug)
	}
	page := 1
	var gr *github.RepositoryRelease
	for {
		releases, resp, listErr := c.Repositories.ListReleases(ctx, rs[0], rs[1], &github.ListOptions{
			Page:    page,
			PerPage: 50,
		})
		if listErr != nil {
			return "", nil, fmt.Errorf("unable to request releases page %d for %q: %w", page, s.GitHubSlug, listErr)
		} else if resp.StatusCode != http.StatusOK {
			return "", nil, fmt.Errorf("failed to list releases page %d for %q: %s", page, s.GitHubSlug, resp.Status)
		}
		if page == resp.LastPage {
			break
		}

		for _, r := range releases {
			if r.GetName() == b.Version {
				gr = r
				break
			}
		}
		if gr != nil {
			goto end_search
		}
		page = resp.NextPage
	}
end_search:

	if gr == nil {
		return "", nil, fmt.Errorf("repository %q does not have a release named %q", s.GitHubSlug, b.Version)
	}

	name = s.instantiateTemplate(b, s.GitHubReleaseAssetTemplate)
	var a *github.ReleaseAsset
	for _, c := range gr.Assets {
		if c.GetName() == name {
			a = c
			break
		}
	}
	if a == nil {
		return "", nil, fmt.Errorf("release %q of repository %q does not have an asset named %q", b.Version, s.GitHubSlug, name)
	}

	dl, _, err := c.Repositories.DownloadReleaseAsset(ctx, rs[0], rs[1], a.GetID(), http.DefaultClient)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get link to asset %q from release %q in repository %q: %w", name, b.Version, s.GitHubSlug, err)
	}
	buf := &bytes.Buffer{}
	if _, err = io.Copy(buf, dl); err != nil {
		return "", nil, fmt.Errorf("failed to download asset %q from release %q in repository %q: %w", name, b.Version, s.GitHubSlug, err)
	}
	return name, buf, nil
}
