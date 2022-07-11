package backend

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGitHub(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string {
		c := s
		return &c
	}
	strInt64 := func(i int64) *int64 {
		c := i
		return &c
	}

	fakeGH := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposReleasesByOwnerByRepo,
			[]github.RepositoryRelease{
				{
					Name: strPtr("v1.2.3"),
					Assets: []*github.ReleaseAsset{
						{
							ID:   strInt64(123456),
							Name: strPtr("test-tool_v1.2.3_linux_x86_64"),
						},
					},
				},
			},
		),
		mock.WithRequestMatchHandler(
			mock.GetReposReleasesAssetsByOwnerByRepoByAssetId,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(stdTestBinaryContent)
			}),
		),
	)

	gh := &GitHub{
		log:     zap.NewNop(),
		timeout: 10 * time.Second,
		client:  github.NewClient(fakeGH),
		GitHubConfig: GitHubConfig{
			GitHubSlug:                 "foo/bar",
			GitHubReleaseAssetTemplate: stdTestTemplate,
		},
	}

	b, err := gh.Fetch(stdTestBinary)
	require.NoError(t, err)
	assert.Equal(t, stdTestBinaryContent, b)

	err = gh.Store(stdTestBinary, stdTestBinaryContent)
	require.Error(t, err)
}
