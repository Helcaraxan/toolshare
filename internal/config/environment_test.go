package config

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
	err := mergeEnvironment(env, filepath.Join("testdata", "erroneous_config_syntax.yaml"))
	require.Error(t, err)
}

func TestMergeEnvironment(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		testfile    string
		errType     error
		sourceCount int
	}{
		"ValidGitHubSource": {
			testfile:    "config_valid_github.yaml",
			sourceCount: 3,
		},
		"ValidURLSource": {
			testfile:    "config_valid_url.yaml",
			sourceCount: 4,
		},
		"InvalidNonExistentFile": {
			testfile: "non-existent",
			errType:  os.ErrNotExist,
		},
		"InvalidEmpty": {
			testfile: "config_invalid_empty.yaml",
			errType:  ErrInvalidSource,
		},
		"InvalidMixedParameters": {
			testfile: "config_invalid_mixed.yaml",
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
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := &Environment{Sources: map[string]Source{}}
			err := mergeEnvironment(env, filepath.Join("testdata", testcase.testfile))
			if err == nil {
				for _, source := range env.Sources {
					if err == nil {
						err = source.Validate()
					}
				}
			}
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

func TestMergeForcePin(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "child.yaml"), []byte("---\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "parent.yaml"), []byte("---\nenforce_pins: true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "ancestor.yaml"), []byte("---\nenforce_pins: false\n"), 0o644))

	env := &Environment{
		Pins:    map[string]string{},
		Sources: map[string]Source{},
	}
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "child.yaml")))
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "parent.yaml")))
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "ancestor.yaml")))

	assert.True(t, env.EnforcePins)
}

func TestMergePins(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "child.yaml"), []byte("pins:\n  b: child\n  c: child\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "parent.yaml"), []byte("pins:\n  a: parent\n  b: parent\n"), 0o644))

	env := &Environment{
		Pins:    map[string]string{},
		Sources: map[string]Source{},
	}
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "child.yaml")))
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "parent.yaml")))

	assert.Equal(t, "parent", env.Pins["a"])
	assert.Equal(t, "child", env.Pins["b"])
	assert.Equal(t, "child", env.Pins["c"])
}

func TestMergeSources(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "child.yaml"), []byte(`---
sources:
  b:
    type: local
    url:
      url_template: child
  c:
    type: local
    url:
      url_template: child
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "parent.yaml"), []byte(`---
sources:
  a:
    type: local
    url:
      url_template: parent
  b:
    type: local
    url:
      url_template: parent
`), 0o644))

	env := &Environment{
		Pins:    map[string]string{},
		Sources: map[string]Source{},
	}
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "child.yaml")))
	require.NoError(t, mergeEnvironment(env, filepath.Join(testDir, "parent.yaml")))

	assert.Equal(t, "parent", env.Sources["a"].URL.URLTemplate)
	assert.Equal(t, "child", env.Sources["b"].URL.URLTemplate)
	assert.Equal(t, "child", env.Sources["c"].URL.URLTemplate)
}
