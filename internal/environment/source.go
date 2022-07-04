package environment

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Helcaraxan/toolshare/internal/backend"
)

var ErrInvalidSource = errors.New("invalid source")

type Source struct {
	*backend.FileSystemConfig
	*backend.HTTPSConfig
	*backend.GCSConfig
	*backend.GitHubConfig
	*backend.S3Config

	common *backend.CommonConfig
}

func (s *Source) String() string {
	switch {
	case s.FileSystemConfig != nil:
		return s.FilePathTemplate
	case s.GCSConfig != nil:
		return fmt.Sprintf("gs://%s/%s", s.GCSBucket, s.GCSPathTemplate)
	case s.GitHubConfig != nil:
		b := s.GitHubBaseURL
		if b == "" {
			b = "github.com"
		}
		return fmt.Sprintf("%s/%s:%s", b, s.GitHubSlug, s.GitHubReleaseAssetTemplate)
	case s.HTTPSConfig != nil:
		return s.HTTPSURLTemplate
	case s.S3Config != nil:
		return fmt.Sprintf("s3://%s/%s", s.S3Bucket, s.S3PathTemplate)
	default:
		return ""
	}
}

func (s *Source) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("can not unmarshal non-mapping yaml as a source definition")
	}

	var isFile, isGCS, isGitHub, isHTTPS, isS3 bool
	for idx := 0; idx < len(value.Content); idx += 2 {
		if value.Content[idx].Kind != yaml.ScalarNode {
			return fmt.Errorf("expected mapping data inside yaml object")
		}

		fe := strings.Split(value.Content[idx].Value, "_")
		switch fe[0] {
		case "file":
			isFile = true
		case "gcs":
			isGCS = true
		case "github":
			isGitHub = true
		case "https":
			isHTTPS = true
		case "s3":
			isS3 = true
		}
	}

	if isFile {
		s.FileSystemConfig = &backend.FileSystemConfig{}
		if err := value.Decode(s.FileSystemConfig); err != nil {
			return err
		}
		s.common = &s.FileSystemConfig.CommonConfig
	}
	if isGCS {
		s.GCSConfig = &backend.GCSConfig{}
		if err := value.Decode(s.GCSConfig); err != nil {
			return err
		}
		s.common = &s.GCSConfig.CommonConfig
	}
	if isGitHub {
		s.GitHubConfig = &backend.GitHubConfig{}
		if err := value.Decode(s.GitHubConfig); err != nil {
			return err
		}
		s.common = &s.GitHubConfig.CommonConfig
	}
	if isHTTPS {
		s.HTTPSConfig = &backend.HTTPSConfig{}
		if err := value.Decode(s.HTTPSConfig); err != nil {
			return err
		}
		s.common = &s.HTTPSConfig.CommonConfig
	}
	if isS3 {
		s.S3Config = &backend.S3Config{}
		if err := value.Decode(s.S3Config); err != nil {
			return err
		}
		s.common = &s.S3Config.CommonConfig
	}
	return s.validate()
}

func (s *Source) validate() error {
	var sourceConfigCount int
	for _, si := range []interface{}{s.FileSystemConfig, s.GCSConfig, s.GitHubConfig, s.HTTPSConfig, s.S3Config} {
		if !reflect.ValueOf(si).IsNil() {
			sourceConfigCount++
		}
	}

	if sourceConfigCount == 0 {
		return fmt.Errorf("backend has no configuration attached: %w", ErrInvalidSource)
	} else if sourceConfigCount > 1 {
		return fmt.Errorf("backend has multiple configuration attached: %w", ErrInvalidSource)
	}

	switch {
	case s.FileSystemConfig != nil:
		if s.FilePathTemplate == "" {
			return fmt.Errorf("filesystem backend has no path template set: %w", ErrInvalidSource)
		}

	case s.GCSConfig != nil:
		if s.GCSBucket == "" || s.GCSPathTemplate == "" {
			return fmt.Errorf("gcs backend has no bucket and / or path template set: %w", ErrInvalidSource)
		}

	case s.GitHubConfig != nil:
		if s.GitHubSlug == "" || s.GitHubReleaseAssetTemplate == "" {
			return fmt.Errorf("github backend has no slug and / or release asset template set: %w", ErrInvalidSource)
		}

	case s.HTTPSConfig != nil:
		if s.HTTPSURLTemplate == "" {
			return fmt.Errorf("https backend has no url template set: %w", ErrInvalidSource)
		}

	case s.S3Config != nil:
		if s.S3Bucket == "" || s.S3PathTemplate == "" {
			return fmt.Errorf("s3 backend has no bucket and / or path template set: %w", ErrInvalidSource)
		}

	default:
		return fmt.Errorf("%w: no parameters were specified", ErrInvalidSource)
	}
	return nil
}
