package backend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

func TestHTTPS(t *testing.T) {
	t.Skip() // Skipping until we've implemented the actual HTTP backend.

	t.Parallel()

	serverStorage := map[string][]byte{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			b, ok := serverStorage[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
			} else {
				_, _ = w.Write(b)
			}

		case http.MethodPost, http.MethodPut:
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				serverStorage[r.URL.Path] = b
			}
		}
	}))
	t.Cleanup(testServer.Close)

	https := NewHTTPS(logger.NewTestBuilder(), &HTTPSConfig{HTTPSURLTemplate: testServer.URL + "/" + stdTemplate})
	bin := config.Binary{
		Tool:     "test-tool",
		Version:  "v1.2.3",
		Platform: config.PlatformLinux,
		Arch:     config.ArchX64,
	}
	binContent := []byte("tool-binary-content")

	_, err := https.Fetch(bin)
	assert.Error(t, err)

	err = https.Store(bin, binContent)
	require.NoError(t, err)

	b, err := https.Fetch(bin)
	require.NoError(t, err)
	assert.Equal(t, binContent, b)
}
