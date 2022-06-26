package backend

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

type githubStorage struct {
	log     *logrus.Logger
	timeout time.Duration

	source config.Source
}

func (s *githubStorage) Fetch(b tool.Binary, targetPath string) error {
	if s.source.GitHub == nil {
		return errors.New("no github source configuration available")
	}
	gh := s.source.GitHub

	assetName, buf, err := s.getReleaseAsset(b)
	if err != nil {
		return err
	}

	rd := io.Reader(buf)
	sz := int64(buf.Len())
	if gh.ArchivePathTemplate != "" {
		var archivePath string
		archivePath, err = s.source.ArchivePath(b)
		if err != nil {
			return err
		}

		switch {
		case strings.HasSuffix(assetName, ".zip"):
			var zr *zip.Reader
			var fl fs.File
			var fi fs.FileInfo
			zr, err = zip.NewReader(bytes.NewReader(buf.Bytes()), sz)
			if err != nil {
				return fmt.Errorf("failed to open asset %q from release %q in repository %q as zip archive: %w", assetName, b.Version, gh.Slug, err)
			}
			fl, err = zr.Open(archivePath)
			if err != nil {
				return fmt.Errorf("failed to find path %q inside asset %q from release %q in repository %q: %w", archivePath, assetName, b.Version, gh.Slug, err)
			}
			fi, err = fl.Stat()
			if err != nil {
				return fmt.Errorf("failed to read file information for path %q inside asset %q from release %q in repository %q: %w", archivePath, assetName, b.Version, gh.Slug, err)
			}
			rd = fl
			sz = fi.Size()

		case strings.HasSuffix(assetName, ".tar.gz"):
			rd, err = gzip.NewReader(rd)
			if err != nil {
				return fmt.Errorf("failed to open gzip reader for asset %q from release %q in repository %q: %w", archivePath, assetName, gh.Slug, err)
			}
			fallthrough

		case strings.HasSuffix(assetName, ".tar"):
			var hdr *tar.Header
			tr := tar.NewReader(rd)
			for err == nil {
				hdr, err = tr.Next()
				if hdr.Name == archivePath {
					break
				}
			}
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("failed to search for path %q in asset %q from release %q in repository %q: %w", archivePath, assetName, b.Version, gh.Slug, err)
			} else if hdr == nil {
				return fmt.Errorf("no path %q found in archive asset %q from release %q in repository %q: %w", archivePath, assetName, b.Version, gh.Slug, err)
			}
			rd = tr
			sz = hdr.Size

		default:
			return fmt.Errorf("unrecognised archive format in asset %q from release %q in repository %q", assetName, b.Version, gh.Slug)
		}
	}

	w, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE, 0o755)
	if err != nil {
		return fmt.Errorf("failed to open target file %q to write tool binary: %w", targetPath, err)
	}
	_, err = io.CopyN(w, rd, sz)
	if err != nil {
		return fmt.Errorf("failed to copy content for tool binary to target file %q: %w", targetPath, err)
	}
	return nil
}

func (s *githubStorage) getReleaseAsset(b tool.Binary) (name string, content *bytes.Buffer, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	gh := s.source.GitHub
	var c *github.Client
	if gh.BaseURL == "" {
		c = github.NewClient(http.DefaultClient)
	} else {
		c, err = github.NewEnterpriseClient(gh.BaseURL, gh.BaseURL, http.DefaultClient)
		if err != nil {
			return "", nil, err
		}
	}

	rs := strings.Split(gh.Slug, "/")
	if len(rs) != 2 {
		return "", nil, fmt.Errorf("repo slug %q is invalid as it does not contain an owner and repo name", gh.Slug)
	}
	page := 1
	var gr *github.RepositoryRelease
	for {
		releases, resp, listErr := c.Repositories.ListReleases(ctx, rs[0], rs[1], &github.ListOptions{
			Page:    page,
			PerPage: 50,
		})
		if listErr != nil {
			return "", nil, fmt.Errorf("unable to request releases page %d for %q: %w", page, gh.Slug, listErr)
		} else if resp.StatusCode != http.StatusOK {
			return "", nil, fmt.Errorf("failed to list releases page %d for %q: %s", page, gh.Slug, resp.Status)
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
		return "", nil, fmt.Errorf("repository %q does not have a release named %q", gh.Slug, b.Version)
	}

	name, err = s.source.ResourcePath(b)
	if err != nil {
		return "", nil, err
	}

	var a *github.ReleaseAsset
	for _, c := range gr.Assets {
		if c.GetName() == name {
			a = c
			break
		}
	}
	if a == nil {
		return "", nil, fmt.Errorf("release %q of repository %q does not have an asset named %q", b.Version, gh.Slug, name)
	}

	dl, _, err := c.Repositories.DownloadReleaseAsset(ctx, rs[0], rs[1], a.GetID(), http.DefaultClient)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get link to asset %q from release %q in repository %q: %w", name, b.Version, gh.Slug, err)
	}
	buf := &bytes.Buffer{}
	if _, err = io.Copy(buf, dl); err != nil {
		return "", nil, fmt.Errorf("failed to download asset %q from release %q in repository %q: %w", name, b.Version, gh.Slug, err)
	}
	return name, buf, nil
}

func (s *githubStorage) Store(b tool.Binary, path string) (err error) {
	s.log.Error("Cannot perform 'store' operations on a GitHub backend.")
	return errFailed
}
