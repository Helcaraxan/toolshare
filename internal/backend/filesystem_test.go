package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

func TestFileSystem(t *testing.T) {
	t.Parallel()

	var (
		fs  = NewFileSystem(logger.NewTestBuilder(), &FileSystemConfig{FilePathTemplate: stdTemplate}, true)
		bin = config.Binary{
			Tool:     "test-tool",
			Version:  "v1.2.3",
			Platform: config.PlatformLinux,
			Arch:     config.ArchX64,
		}
		binContent = []byte("tool-binary-content")
	)

	assert.Equal(t, "test-tool_v1.2.3_linux_x86_64", fs.Path(bin))

	b, err := fs.Fetch(bin)
	require.Error(t, err)
	assert.Nil(t, b)

	err = fs.Store(bin, binContent)
	require.NoError(t, err)

	err = fs.Store(bin, binContent)
	require.Error(t, err)

	b, err = fs.Fetch(bin)
	require.NoError(t, err)
	assert.Equal(t, binContent, b)
}
