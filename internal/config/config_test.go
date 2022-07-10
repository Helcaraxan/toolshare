package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfigUnmarshal(t *testing.T) {
	t.Parallel()
	for name := range unmarshalTestCases {
		tc := unmarshalTestCases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dec := yaml.NewDecoder(bytes.NewBufferString(tc.content))
			dec.KnownFields(true)

			var conf Global
			err := dec.Decode(&conf)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var unmarshalTestCases = map[string]struct {
	expectedErr bool
	content     string
}{
	"ValidFilesystemCache": {
		expectedErr: false,
		content: `---
remote_cache:
  path_prefix: /mounts/nfs/tools
`,
	},
	"ValidGCSCache": {
		expectedErr: false,
		content: `---
remote_cache:
  path_prefix: /cache-root
  gcs_bucket: my-tool-cache
`,
	},
	"ValidHTTPSCache": {
		expectedErr: false,
		content: `---
remote_cache:
  path_prefix: /cache-root
  https_host: https://tools-cache.my-domain.com
`,
	},
	"ValidS3Cache": {
		expectedErr: false,
		content: `---
remote_cache:
  path_prefix: /cache-root
  s3_bucket: my-tool-cache
`,
	},
	"ValidLockedDownConfig": {
		expectedErr: false,
		content: `---
force_pinned: true
disable_sources: true
`,
	},
	"InvalidMixedCache": {
		expectedErr: true,
		content: `---
remote_cache:
  gcs_bucket: my-tool-cache
  https_host: https://tools-cache.my-domain.com
`,
	},
	"InvalidErroneousCache": {
		expectedErr: true,
		content: `---
remote_cache:
  unknown_setting: foo/bar
`,
	},
}
