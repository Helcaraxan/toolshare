package backend

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/toolshare/internal/logger"
)

func TestFileSystem(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	fs := NewFileSystem(logger.NewTestBuilder(), &FileSystemConfig{FilePathTemplate: filepath.Join(td, stdTestTemplate)})

	assert.Equal(t, filepath.Join(td, "test-tool_v1.2.3_linux_x86_64"), fs.Path(stdTestBinary))

	b, err := fs.Fetch(stdTestBinary)
	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, b)

	err = fs.Store(stdTestBinary, stdTestBinaryContent)
	require.NoError(t, err)

	b, err = fs.Fetch(stdTestBinary)
	require.NoError(t, err)
	assert.Equal(t, stdTestBinaryContent, b)

	err = fs.Store(stdTestBinary, stdTestBinaryContent)
	require.NoError(t, err)
}
