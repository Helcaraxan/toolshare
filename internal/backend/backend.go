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
	"regexp"
	"strings"

	"github.com/ulikunitz/xz"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
)

type Storage interface {
	fmt.Stringer
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
	_ Storage = &GCS{}
	_ Storage = &GitHub{}
	_ Storage = &HTTPS{}
	_ Storage = &S3{}

	errFailed = errors.New("failed")
)

type CommonConfig struct {
	ArchivePathTemplate string           `json:"archive_path_template"`
	Mappings            TemplateMappings `json:"template_mappings"`
}

type TemplateMappings struct {
	// OS name mappings.
	Darwin  *string `json:"darwin"`
	Linux   *string `json:"linux"`
	Windows *string `json:"windows"`

	// Arch name mappings.
	ARM32 *string `json:"arm32"`
	ARM64 *string `json:"arm64"`
	X86   *string `json:"x86_32"`
	X8664 *string `json:"x86_64"`
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

func (c *CommonConfig) extractFromArchive(log *zap.Logger, srcRaw []byte, srcPath string, b config.Binary) ([]byte, error) {
	if c.ArchivePathTemplate == "" {
		log.Debug("No archive path set. Using the fetched content as the tool binary itself.")
		return srcRaw, nil
	}

	var (
		err error
		raw []byte
		rd  io.Reader
	)

	archivePath := c.instantiateTemplate(b, c.ArchivePathTemplate)
	log = log.With(zap.String("archive-path", archivePath))

	switch {
	case strings.HasSuffix(srcPath, ".zip"):
		rd, err = c.extractFromArchiveZIP(log, srcRaw, archivePath)

	case regexp.MustCompile(".tar(.(g|x)z)?$").MatchString(srcPath):
		switch {
		case strings.HasSuffix(srcPath, ".gz"):
			log.Debug("Applying a GZIP decoder on the fetched content.")
			rd, err = gzip.NewReader(bytes.NewBuffer(srcRaw))
			if err != nil {
				log.Error("Failed to open fetched content with a GZIP reader.", zap.Error(err))
				return nil, fmt.Errorf("failed to open gzip reader for fetched content: %w", err)
			}
		case strings.HasSuffix(srcPath, "xz"):
			log.Debug("Applying an XZ decoder on the fetched content.")
			rd, err = xz.NewReader(bytes.NewBuffer(srcRaw))
			if err != nil {
				log.Error("Failed to open fetched content with an XZ reader.", zap.Error(err))
				return nil, fmt.Errorf("failed to open xz reader for fetched content: %w", err)
			}
		}
		rd, err = c.extractFromArchiveTAR(log, srcRaw, archivePath, rd)

	default:
		err = fmt.Errorf("unrecognised archive format: %w", errors.ErrUnsupported)
	}
	if err != nil {
		return nil, err
	}

	raw, err = io.ReadAll(rd)
	if err != nil {
		log.Error("Failed to read binary from archive.", zap.Error(err))
		return nil, err
	}
	log.Debug("Successfully read binary from archive.")
	return raw, nil
}

func (c *CommonConfig) extractFromArchiveZIP(log *zap.Logger, srcRaw []byte, archivePath string) (io.Reader, error) {
	log.Debug("Reading the fetched content as a ZIP archive.")
	var (
		zr *zip.Reader
		fl fs.File
	)
	zr, err := zip.NewReader(bytes.NewReader(srcRaw), int64(len(srcRaw)))
	if err != nil {
		log.Error("Failed to open content with a ZIP reader.", zap.Error(err))
		return nil, fmt.Errorf("failed to open fetched content as zip archive: %w", err)
	}
	fl, err = zr.Open(archivePath)
	if err != nil {
		log.Error("Path not found in archive.", zap.Error(err))
		return nil, fmt.Errorf("failed to find path inside fetched content: %w", err)
	}
	_, err = fl.Stat()
	if err != nil {
		log.Error("Failed to open archive path for reading.", zap.Error(err))
		return nil, fmt.Errorf("failed to read file information for path inside fetched content: %w", err)
	}
	return fl, nil
}

func (c *CommonConfig) extractFromArchiveTAR(log *zap.Logger, srcRaw []byte, archivePath string, rd io.Reader) (io.Reader, error) {
	log.Debug("Reading the fetched content as a TAR archive.")
	if rd == nil {
		rd = bytes.NewBuffer(srcRaw)
	}
	tr := tar.NewReader(rd)

	hdr, err := tr.Next()
	for err == nil {
		if hdr.Name == archivePath {
			break
		}
		hdr, err = tr.Next()
	}
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		log.Error("Failed to search archive for path.", zap.Error(err))
		return nil, fmt.Errorf("failed to search for path in fetched content: %w", err)
	} else if hdr == nil {
		log.Error("Path not found in archive.")
		return nil, fmt.Errorf("failed to find path in fetched content: %w", err)
	}
	return tr, nil
}
