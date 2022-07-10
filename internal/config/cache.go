package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

type Cache struct {
	cacheContent
}

type cacheContent struct {
	PathPrefix string `yaml:"path_prefix"`

	GCSBucket string `yaml:"gcs_bucket"`
	HTTPSHost string `yaml:"https_host"`
	S3Bucket  string `yaml:"s3_bucket"`
}

func (c *Cache) UnmarshalYAML(value *yaml.Node) error {
	if err := value.Decode(&c.cacheContent); err != nil {
		return err
	}
	all := map[string]interface{}{}
	if err := value.Decode(&all); err != nil {
		return err
	}
	for _, k := range []string{"path_prefix", "gcs_bucket", "https_host", "s3_bucket"} {
		delete(all, k)
	}
	if len(all) > 0 {
		return errors.New("unknown fields present in cache configuration")
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
