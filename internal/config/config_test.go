package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigUnmarshal(t *testing.T) {
	t.Parallel()

	unmarshalTestCases := map[string]struct {
		testFile    string
		expectedErr bool
	}{
		"ValidFilesystemCache":  {testFile: "valid_filesystem_cache.yaml", expectedErr: false},
		"ValidGCSCache":         {testFile: "valid_gcs_cache.yaml", expectedErr: false},
		"ValidHTTPSCache":       {testFile: "valid_https_cache.yaml", expectedErr: false},
		"ValidS3Cache":          {testFile: "valid_s3_cache.yaml", expectedErr: false},
		"ValidLockedDownConfig": {testFile: "valid_locked_down_config.yaml", expectedErr: false},
		"InvalidMixedCache":     {testFile: "invalid_mixed_cache.yaml", expectedErr: true},
		"InvalidErroneousCache": {testFile: "invalid_unknown_cache.yaml", expectedErr: true},
	}

	for name := range unmarshalTestCases {
		tc := unmarshalTestCases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(filepath.Join("testdata", tc.testFile))
			require.NoError(t, err)
			dec := yaml.NewDecoder(bytes.NewBuffer(raw), yaml.Strict())

			var conf Global
			err = dec.Decode(&conf)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
