package backend

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	s3_lib "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestS3(t *testing.T) {
	t.Parallel()

	const bucketName = "test-bucket"

	backend := s3mem.New()
	err := backend.CreateBucket(bucketName)
	require.NoError(t, err)

	fakeS3 := gofakes3.New(backend)
	serv := httptest.NewServer(fakeS3.Server())
	defer serv.Close()

	s3Config, err := aws_config.LoadDefaultConfig(
		context.TODO(),
		aws_config.WithSharedConfigProfile("test"),
		aws_config.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}),
		aws_config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(_ string, _ string, _ ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: serv.URL}, nil
			}),
		),
	)
	require.NoError(t, err)

	s3 := &S3{
		log:     zap.NewNop(),
		timeout: 10 * time.Second,
		client:  s3_lib.NewFromConfig(s3Config, func(o *s3_lib.Options) { o.UsePathStyle = true }),
		S3Config: S3Config{
			S3Bucket:       bucketName,
			S3PathTemplate: stdTestTemplate,
		},
	}

	b, err := s3.Fetch(stdTestBinary)
	require.Error(t, err)
	assert.Nil(t, b)

	err = s3.Store(stdTestBinary, stdTestBinaryContent)
	require.NoError(t, err)

	err = s3.Store(stdTestBinary, stdTestBinaryContent)
	require.Error(t, err)

	b, err = s3.Fetch(stdTestBinary)
	require.NoError(t, err)
	assert.Equal(t, stdTestBinaryContent, b)
}
