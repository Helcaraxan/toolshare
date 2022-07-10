package backend

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
)

const stdTemplate = "{tool}_{version}_{platform}_{arch}{exe}"

func TestInstantiateTemplate(t *testing.T) {
	t.Parallel()

	testBin := func(p config.Platform, a config.Arch) config.Binary {
		return config.Binary{
			Tool:     "test-tool",
			Version:  "v1.2.3",
			Platform: p,
			Arch:     a,
		}
	}
	strPtr := func(s string) *string {
		c := s
		return &c
	}

	testcases := map[string]struct {
		in       string
		bin      config.Binary
		mappings TemplateMappings
		out      string
	}{
		"NoPlaceholders": {
			in:  "my-template/without_any.placeholders{in}it",
			bin: testBin(config.PlatformDarwin, config.ArchARM64),
			out: "my-template/without_any.placeholders{in}it",
		},
		"OnlyToolPlaceholder": {
			in:  "my-template/with_a.{tool}@placeholder",
			bin: testBin(config.PlatformDarwin, config.ArchARM64),
			out: "my-template/with_a.test-tool@placeholder",
		},
		"DarwinARM64": {
			in:  stdTemplate,
			bin: testBin(config.PlatformDarwin, config.ArchARM64),
			out: "test-tool_v1.2.3_darwin_arm64",
		},
		"LinuxARM32": {
			in:  stdTemplate,
			bin: testBin(config.PlatformLinux, config.ArchARM32),
			out: "test-tool_v1.2.3_linux_arm32",
		},
		"WindowsX86": {
			in:  stdTemplate,
			bin: testBin(config.PlatformWindows, config.ArchX86),
			out: "test-tool_v1.2.3_windows_x86.exe",
		},
		"DarwinX64": {
			in:  stdTemplate,
			bin: testBin(config.PlatformDarwin, config.ArchX64),
			out: "test-tool_v1.2.3_darwin_x86_64",
		},
		"DarwinARM64Mappings": {
			in:       stdTemplate,
			bin:      testBin(config.PlatformDarwin, config.ArchARM64),
			mappings: TemplateMappings{Darwin: strPtr("macos"), ARM64: strPtr("arm-64")},
			out:      "test-tool_v1.2.3_macos_arm-64",
		},
		"LinuxARM32Mappings": {
			in:       stdTemplate,
			bin:      testBin(config.PlatformLinux, config.ArchARM32),
			mappings: TemplateMappings{Linux: strPtr("unix"), ARM32: strPtr("armv1")},
			out:      "test-tool_v1.2.3_unix_armv1",
		},
		"WindowsX86Mappings": {
			in:       stdTemplate,
			bin:      testBin(config.PlatformWindows, config.ArchX86),
			mappings: TemplateMappings{Windows: strPtr("win11"), X86: strPtr("x86_32")},
			out:      "test-tool_v1.2.3_win11_x86_32.exe",
		},
		"DarwinX64Mappings": {
			in:       stdTemplate,
			bin:      testBin(config.PlatformDarwin, config.ArchX64),
			mappings: TemplateMappings{Darwin: strPtr("osx"), X8664: strPtr("amd64")},
			out:      "test-tool_v1.2.3_osx_amd64",
		},
		"NonStandardPlatformArch": {
			in:  stdTemplate,
			bin: testBin(config.Platform("solaris"), config.Arch("rv64i")),
			out: "test-tool_v1.2.3_solaris_rv64i",
		},
	}

	for name := range testcases {
		tc := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := CommonConfig{Mappings: tc.mappings}
			out := c.instantiateTemplate(tc.bin, tc.in)
			assert.Equal(t, tc.out, out)
		})
	}
}

func TestArchiveExtractionNoTemplate(t *testing.T) {
	var (
		conf    = &CommonConfig{}
		content = []byte("tool-binary-content")
	)

	b, err := conf.extractFromArchive(zap.NewNop(), content, "my-tool", config.Binary{})
	require.NoError(t, err)
	assert.Equal(t, content, b)
}

func TestArchiveExtractionUnknownFormat(t *testing.T) {
	conf := &CommonConfig{ArchivePathTemplate: "foo/bar"}

	b, err := conf.extractFromArchive(zap.NewNop(), nil, "archive.unknown", config.Binary{})
	assert.Error(t, err)
	assert.Nil(t, b)
}

func TestArchiveExtractionZIP(t *testing.T) {
	t.Parallel()

	var (
		testArchive bytes.Buffer
		testContent = []byte("tool-binary-bytes")
		conf        = &CommonConfig{ArchivePathTemplate: "{platform}/{arch}/{tool}"}
		bin         = config.Binary{
			Tool:     "my-tool",
			Platform: config.PlatformLinux,
			Arch:     config.ArchX64,
		}
	)

	b, err := conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.zip", bin)
	require.Error(t, err)
	assert.Nil(t, b)

	archiveWriter := zip.NewWriter(&testArchive)
	require.NoError(t, archiveWriter.Close())
	b, err = conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.zip", bin)
	require.Error(t, err)
	assert.Nil(t, b)

	testArchive.Reset()
	archiveWriter = zip.NewWriter(&testArchive)
	contentWriter, err := archiveWriter.Create("linux/x86_64/my-tool")
	require.NoError(t, err)
	_, err = contentWriter.Write(testContent)
	require.NoError(t, err)
	require.NoError(t, archiveWriter.Close())

	b, err = conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.zip", bin)
	require.NoError(t, err)
	assert.Equal(t, testContent, b)
}

func TestArchiveExtractionGzipTAR(t *testing.T) {
	t.Parallel()

	var (
		testArchive bytes.Buffer
		testContent = []byte("tool-binary-bytes")
		conf        = &CommonConfig{ArchivePathTemplate: "{platform}/{arch}/{tool}"}
		bin         = config.Binary{
			Tool:     "my-tool",
			Platform: config.PlatformLinux,
			Arch:     config.ArchX64,
		}
	)

	b, err := conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.tar.gz", bin)
	require.Error(t, err)
	assert.Nil(t, b)

	archiveWriter := tar.NewWriter(&testArchive)
	require.NoError(t, archiveWriter.Close())
	b, err = conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.tar.gz", bin)
	require.Error(t, err)
	assert.Nil(t, b)

	testArchive.Reset()
	archiveWriter = tar.NewWriter(gzip.NewWriter(&testArchive))
	require.NoError(t, archiveWriter.Close())
	b, err = conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.tar.gz", bin)
	require.Error(t, err)
	assert.Nil(t, b)

	testArchive.Reset()
	compressor := gzip.NewWriter(&testArchive)
	archiveWriter = tar.NewWriter(compressor)
	require.NoError(t, archiveWriter.WriteHeader(&tar.Header{
		Name: "linux/x86_64/my-tool",
		Size: int64(len(testContent)),
		Mode: 0o755,
	}))
	_, err = archiveWriter.Write(testContent)
	require.NoError(t, err)
	require.NoError(t, archiveWriter.Close())
	require.NoError(t, compressor.Close())

	b, err = conf.extractFromArchive(zap.NewNop(), testArchive.Bytes(), "archive.tar.gz", bin)
	require.NoError(t, err)
	assert.Equal(t, testContent, b)
}
