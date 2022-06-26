package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Helcaraxan/toolshare/internal/tool"
)

var ErrInvalidSource = errors.New("invalid source")

type SourceType = string

const (
	SourceTypeGCS    SourceType = "gcs"
	SourceTypeGitHub SourceType = "github"
	SourceTypeHTTPS  SourceType = "https"
	SourceTypeLocal  SourceType = "local"
	SourceTypeS3     SourceType = "s3"
)

type Source struct {
	Type     SourceType       `yaml:"type"`
	Mappings templateMappings `json:"template_mappings"`
	// For GitHub sources.
	GitHub *GithubSource `yaml:"github"`
	// For local and cloud bucket sources.
	URL *URLSource `yaml:"url"`
}

type GithubSource struct {
	BaseURL              string `yaml:"base_url"`
	Slug                 string `yaml:"slug"`
	ReleaseAssetTemplate string `yaml:"release_asset_template"`
	ArchivePathTemplate  string `yaml:"archive_path_template"`
}

type URLSource struct {
	URLTemplate         string `yaml:"url_template"`
	ArchivePathTemplate string `yaml:"archive_path_template"`
}

func (s *Source) Validate() error {
	switch s.Type {
	case SourceTypeGitHub:
		if s.GitHub == nil {
			return fmt.Errorf("%w: missing configuration for github source", ErrInvalidSource)
		} else if s.URL != nil {
			return fmt.Errorf("%w: github source has extraneous url configuration", ErrInvalidSource)
		}

		if s.GitHub.Slug == "" || s.GitHub.ReleaseAssetTemplate == "" {
			return fmt.Errorf("%w: github source configuration needs at least a slug and a releaseAssetTemplate", ErrInvalidSource)
		}

	case SourceTypeGCS, SourceTypeHTTPS, SourceTypeLocal, SourceTypeS3:
		if s.URL == nil {
			return fmt.Errorf("%w: missing configuration for %s source", ErrInvalidSource, s.Type)
		} else if s.GitHub != nil {
			return fmt.Errorf("%w: %s source has extraneous github configuration", ErrInvalidSource, s.Type)
		}

		_, err := url.Parse(s.URL.URLTemplate)
		if err != nil {
			return fmt.Errorf("%w: invalid url template %q - %v", ErrInvalidSource, s.URL.URLTemplate, err)
		}

	default:
		return fmt.Errorf("%w: no parameters were specified", ErrInvalidSource)
	}
	return nil
}

func (s *Source) ResourcePath(b tool.Binary) (string, error) {
	switch {
	case s.GitHub != nil:
		return s.instantiateTemplate(s.GitHub.ReleaseAssetTemplate, b), nil
	case s.URL != nil:
		return s.instantiateTemplate(s.URL.URLTemplate, b), nil
	}
	return "", errors.New("invalid source specification")
}

func (s *Source) ArchivePath(b tool.Binary) (string, error) {
	var tmpl string
	switch {
	case s.GitHub != nil:
		tmpl = s.GitHub.ArchivePathTemplate
	case s.URL != nil:
		tmpl = s.URL.ArchivePathTemplate
	default:
		return "", errors.New("invalid source specification")
	}
	if tmpl == "" {
		tmpl = b.Tool
		if b.Platform == tool.PlatformWindows {
			tmpl += ".exe"
		}
	}
	return s.instantiateTemplate(tmpl, b), nil
}

func (s *Source) instantiateTemplate(tmpl string, b tool.Binary) string {
	return strings.NewReplacer(
		"arch", s.Mappings.arch(b),
		"os", s.Mappings.platform(b),
		"tool", b.Tool,
		"version", b.Version,
	).Replace(tmpl)
}

type templateMappings struct {
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

func (m *templateMappings) platform(b tool.Binary) string {
	switch b.Platform {
	case tool.PlatformDarwin:
		if m.Darwin != nil {
			return *m.Darwin
		}
		return string(tool.PlatformDarwin)
	case tool.PlatformLinux:
		if m.Linux != nil {
			return *m.Linux
		}
		return string(tool.PlatformLinux)
	case tool.PlatformWindows:
		if m.Windows != nil {
			return *m.Windows
		}
		return string(tool.PlatformWindows)
	default:
		return string(b.Platform)
	}
}

func (m *templateMappings) arch(b tool.Binary) string {
	switch b.Arch {
	case tool.ArchARM32:
		if m.ARM32 != nil {
			return *m.ARM32
		}
		return string(tool.ArchARM32)
	case tool.ArchARM64:
		if m.ARM64 != nil {
			return *m.ARM64
		}
		return string(tool.ArchARM64)
	case tool.ArchX64:
		if m.X8664 != nil {
			return *m.X8664
		}
		return string(tool.ArchX64)
	case tool.ArchX86:
		if m.X86 != nil {
			return *m.X86
		}
		return string(tool.ArchX86)
	default:
		return string(b.Arch)
	}
}
