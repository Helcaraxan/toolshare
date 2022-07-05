package backend

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type Storage interface {
	Fetch(binary config.Binary) ([]byte, error)
	Store(binary config.Binary, content []byte) error
}

type BinaryProvider interface {
	Storage
	Path(binary config.Binary) string
}

var (
	// To guarantee that implementations remain compatible with the interface.
	_ BinaryProvider = &FileSystem{}

	_ Storage = &FileSystem{}
	_ Storage = &HTTPS{}
	_ Storage = &GCS{}
	_ Storage = &GitHub{}
	_ Storage = &S3{}

	errFailed = errors.New("failed")
)

type CommonConfig struct {
	ArchivePathTemplate string           `yaml:"archive_path_template"`
	Mappings            TemplateMappings `yaml:"template_mappings"`
}

type TemplateMappings struct {
	// OS name mappings.
	Darwin  *string `yaml:"darwin"`
	Linux   *string `yaml:"linux"`
	Windows *string `yaml:"windows"`

	// Arch name mappings.
	ARM32 *string `yaml:"arm32"`
	ARM64 *string `yaml:"arm64"`
	X86   *string `yaml:"x86_32"`
	X8664 *string `yaml:"x86_64"`
}

func (c *CommonConfig) instantiateTemplate(b config.Binary, tmpl string) string {
	return strings.NewReplacer(
		"{arch}", c.arch(b),
		"{exe}", c.exe(b),
		"{platform}", c.platform(b),
		"{tool}", b.Tool,
		"{version}", b.Version,
	).Replace(tmpl)
}

func (c *CommonConfig) platform(b config.Binary) string {
	switch b.Platform {
	case config.PlatformDarwin:
		if c.Mappings.Darwin != nil {
			return *c.Mappings.Darwin
		}
		return string(config.PlatformDarwin)
	case config.PlatformLinux:
		if c.Mappings.Linux != nil {
			return *c.Mappings.Linux
		}
		return string(config.PlatformLinux)
	case config.PlatformWindows:
		if c.Mappings.Windows != nil {
			return *c.Mappings.Windows
		}
		return string(config.PlatformWindows)
	default:
		return string(b.Platform)
	}
}

func (c *CommonConfig) arch(b config.Binary) string {
	switch b.Arch {
	case config.ArchARM32:
		if c.Mappings.ARM32 != nil {
			return *c.Mappings.ARM32
		}
		return string(config.ArchARM32)
	case config.ArchARM64:
		if c.Mappings.ARM64 != nil {
			return *c.Mappings.ARM64
		}
		return string(config.ArchARM64)
	case config.ArchX64:
		if c.Mappings.X8664 != nil {
			return *c.Mappings.X8664
		}
		return string(config.ArchX64)
	case config.ArchX86:
		if c.Mappings.X86 != nil {
			return *c.Mappings.X86
		}
		return string(config.ArchX86)
	default:
		return string(b.Arch)
	}
}

func (c *CommonConfig) exe(b config.Binary) string {
	if b.Platform == config.PlatformWindows {
		return ".exe"
	}
	return ""
}

func (c *CommonConfig) extractFromArchive(srcRaw []byte, srcPath string, b config.Binary) ([]byte, error) {
	if c.ArchivePathTemplate == "" {
		return srcRaw, nil
	}

	var (
		err error
		rd  io.Reader
	)

	archivePath := c.instantiateTemplate(b, c.ArchivePathTemplate)
	switch {
	case strings.HasSuffix(srcPath, ".zip"):
		var (
			zr *zip.Reader
			fl fs.File
		)
		zr, err = zip.NewReader(bytes.NewReader(srcRaw), int64(len(srcRaw)))
		if err != nil {
			return nil, fmt.Errorf("failed to open fetched content as zip archive: %w", err)
		}
		fl, err = zr.Open(archivePath)
		if err != nil {
			return nil, fmt.Errorf("failed to find path %q inside fetched content: %w", archivePath, err)
		}
		_, err = fl.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to read file information for path %q inside fetched content: %w", archivePath, err)
		}
		rd = fl

	case strings.HasSuffix(srcPath, ".tar.gz"):
		rd, err = gzip.NewReader(rd)
		if err != nil {
			return nil, fmt.Errorf("failed to open gzip reader for fetched content: %w", err)
		}
		fallthrough

	case strings.HasSuffix(srcPath, ".tar"):
		var hdr *tar.Header
		tr := tar.NewReader(rd)
		for err == nil {
			hdr, err = tr.Next()
			if hdr.Name == archivePath {
				break
			}
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("failed to search for path %q fetched content: %w", archivePath, err)
		} else if hdr == nil {
			return nil, fmt.Errorf("failed to find path %q in fetched content: %w", archivePath, err)
		}
		rd = tr

	default:
		return nil, fmt.Errorf("unrecognised archive format for downloaded content at %q", srcPath)
	}

	return io.ReadAll(rd)
}
