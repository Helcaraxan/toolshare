package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/toolshare/internal/logger"
)

func TestFileSystem(t *testing.T) {
	t.Parallel()

	fs := NewFileSystem(logger.NewTestBuilder(), &FileSystemConfig{FilePathTemplate: stdTestTemplate}, true)

	assert.Equal(t, "test-tool_v1.2.3_linux_x86_64", fs.Path(stdTestBinary))

	b, err := fs.Fetch(stdTestBinary)
	require.Error(t, err)
	assert.Nil(t, b)

	err = fs.Store(stdTestBinary, stdTestBinaryContent)
	require.NoError(t, err)

	err = fs.Store(stdTestBinary, stdTestBinaryContent)
	require.Error(t, err)

	b, err = fs.Fetch(stdTestBinary)
	require.NoError(t, err)
	assert.Equal(t, stdTestBinaryContent, b)
}
