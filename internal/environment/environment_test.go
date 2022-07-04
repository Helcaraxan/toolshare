package environment

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseErroneousConfigSyntax(t *testing.T) {
	t.Parallel()

	env := &Environment{Sources: map[string]Source{}}

	raw, err := os.ReadFile(filepath.Join("testdata", "erroneous_config_syntax.yaml"))
	require.NoError(t, err)

	err = mergeEnvironment(env, raw)
	require.Error(t, err)
}

func TestMergeEnvironment(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		testfile    string
		errType     error
		sourceCount int
	}{
		"InvalidEmpty": {
			testfile: "config_invalid_empty.yaml",
			errType:  ErrInvalidSource,
		},
		"InvalidGitHubMissingSlug": {
			testfile: "config_invalid_github_missing_slug.yaml",
			errType:  ErrInvalidSource,
		},
		"InvalidGitHubMissingReleaseAsset": {
			testfile: "config_invalid_github_missing_asset.yaml",
			errType:  ErrInvalidSource,
		},
		"InvalidMixedParameters": {
			testfile: "config_invalid_mixed.yaml",
			errType:  ErrInvalidSource,
		},
		"ValidFileSystemSource": {
			testfile:    "config_valid_filesystem.yaml",
			sourceCount: 1,
		},
		"ValidGCSSource": {
			testfile:    "config_valid_gcs.yaml",
			sourceCount: 1,
		},
		"ValidGitHubSource": {
			testfile:    "config_valid_github.yaml",
			sourceCount: 3,
		},
		"ValidHTTPSSource": {
			testfile:    "config_valid_https.yaml",
			sourceCount: 1,
		},
		"ValidS3Source": {
			testfile:    "config_valid_s3.yaml",
			sourceCount: 1,
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := &Environment{Sources: map[string]Source{}}

			raw, err := os.ReadFile(filepath.Join("testdata", testcase.testfile))
			require.NoError(t, err)

			err = mergeEnvironment(env, raw)
			if testcase.errType != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, testcase.errType), "error %q should be of type %q", err, testcase.errType)
			} else {
				require.NoError(t, err)
				assert.Len(t, env.Sources, testcase.sourceCount)
			}
		})
	}
}

func TestMergePins(t *testing.T) {
	t.Parallel()

	env := &Environment{
		Pins:    map[string]string{},
		Sources: map[string]Source{},
	}
	require.NoError(t, mergeEnvironment(env, []byte("pins:\n  b: child\n  c: child\n")))
	require.NoError(t, mergeEnvironment(env, []byte("pins:\n  a: parent\n  b: parent\n")))

	assert.Equal(t, "parent", env.Pins["a"])
	assert.Equal(t, "child", env.Pins["b"])
	assert.Equal(t, "child", env.Pins["c"])
}

func TestMergeSources(t *testing.T) {
	t.Parallel()

	childContent := []byte(`---
sources:
  b:
    https_url_template: child
  c:
    https_url_template: child
`)
	parentContent := []byte(`---
sources:
  a:
    https_url_template: parent
  b:
    https_url_template: parent
`)

	env := &Environment{
		Pins:    map[string]string{},
		Sources: map[string]Source{},
	}
	require.NoError(t, mergeEnvironment(env, childContent))
	require.NoError(t, mergeEnvironment(env, parentContent))

	assert.Equal(t, "parent", env.Sources["a"].HTTPSURLTemplate)
	assert.Equal(t, "child", env.Sources["b"].HTTPSURLTemplate)
	assert.Equal(t, "child", env.Sources["c"].HTTPSURLTemplate)
}
