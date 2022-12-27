package backend

import (
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGCS(t *testing.T) {
	t.Parallel()

	const bucketName = "test-bucket"

	fakeGCS := fakestorage.NewServer([]fakestorage.Object{})
	fakeGCS.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: bucketName})

	gcs := &GCS{
		log:     zap.NewNop(),
		timeout: 10 * time.Second,
		client:  fakeGCS.Client(),
		GCSConfig: GCSConfig{
			GCSBucket:       bucketName,
			GCSPathTemplate: stdTestTemplate,
		},
	}

	b, err := gcs.Fetch(stdTestBinary)
	require.Error(t, err)
	assert.Nil(t, b)

	err = gcs.Store(stdTestBinary, stdTestBinaryContent)
	require.NoError(t, err)

	err = gcs.Store(stdTestBinary, stdTestBinaryContent)
	require.Error(t, err)

	b, err = gcs.Fetch(stdTestBinary)
	require.NoError(t, err)
	assert.Equal(t, stdTestBinaryContent, b)
}
