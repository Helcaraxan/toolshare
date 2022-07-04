package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

type cache struct {
	cacheContent
}

type cacheContent struct {
	PathPrefix string `yaml:"file_root"`

	GCSBucket string `yaml:"gcs_bucket"`
	HTTPSHost string `yaml:"https_host"`
	S3Bucket  string `yaml:"s3_bucket"`
}

func (c *cache) UnmarshalYAML(value *yaml.Node) error {
	if err := value.Decode(&c.cacheContent); err != nil {
		return nil
	}

	var hostCount int
	for _, h := range []*string{&c.GCSBucket, &c.HTTPSHost, &c.S3Bucket} {
		if h != nil && *h != "" {
			hostCount++
		}
	}
	if hostCount > 1 {
		return errors.New("Invalid cache configuration, multiple remote hosts / buckets found")
	}
	return nil
}
