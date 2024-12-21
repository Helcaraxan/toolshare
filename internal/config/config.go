package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/goccy/go-yaml"
	"go.uber.org/zap"
)

const (
	DriverName = "toolshare"

	configFileName = DriverName + "_conf.yaml"
)

var (
	ErrAmbiguousBackend = errors.New("ambiguous backend configuration")
	ErrUnknownFields    = errors.New("unknown fields present in cache configuration")
)

type Global struct {
	ForcePinned    bool `json:"force_pinned"`
	DisableSources bool `json:"disable_sources"`

	RemoteCache *Cache `json:"remote_cache"`
	State       *State `json:"state"`
}

type State struct {
	Type            string        `json:"type"`
	Local           string        `json:"local"`
	RefreshInterval time.Duration `json:"refresh_interval"`
}

type Cache struct {
	cacheContent
}

type cacheContent struct {
	PathPrefix string `json:"path_prefix"`

	GCSBucket string `json:"gcs_bucket"`
	HTTPSHost string `json:"https_host"`
	S3Bucket  string `json:"s3_bucket"`
}

func (c *Cache) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&c.cacheContent); err != nil {
		return err
	}
	all := map[string]interface{}{}
	if err := unmarshal(&all); err != nil {
		return err
	}
	for _, k := range []string{"path_prefix", "gcs_bucket", "https_host", "s3_bucket"} {
		delete(all, k)
	}
	if len(all) > 0 {
		return ErrUnknownFields
	}
	var hostCount int
	for _, h := range []*string{&c.GCSBucket, &c.HTTPSHost, &c.S3Bucket} {
		if h != nil && *h != "" {
			hostCount++
		}
	}
	if hostCount > 1 {
		return ErrAmbiguousBackend
	}
	return nil
}

func Parse(log *zap.Logger, conf *Global) error {
	if conf == nil {
		return fmt.Errorf("can not parse configuration into nil struct %w", errors.ErrUnsupported)
	}

	for _, p := range AllDirs() {
		raw, err := os.ReadFile(filepath.Join(p, configFileName))
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}

		dec := yaml.NewDecoder(bytes.NewBuffer(raw))
		if err = dec.Decode(conf); err != nil {
			return err
		}
	}
	log.Sugar().Debugf("Parsed configuration:\n%+v", spew.Sdump(conf))
	return nil
}

func AllDirs() []string {
	// We need the config directories in reverse-order of priority such that we can safely unmarshal
	// them in order into the same target struct and guarantee the expected semantics.
	var dirs []string
	if p := UserDir(); p != "" {
		dirs = append(dirs, p)
	}
	if p := SystemDir(); p != "" {
		dirs = append(dirs, p)
	}
	return dirs
}

func SystemDir() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join("/etc", DriverName)
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), DriverName)
	default:
		panic("unsupported platform")
	}
}

func UserDir() string {
	switch runtime.GOOS {
	case "linux":
		if configPath, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
			return filepath.Join(configPath, DriverName)
		}
		fallthrough
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), ".config", DriverName)
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DriverName)
	default:
		panic("unsupported platform")
	}
}

func StorageDir() string {
	return filepath.Join(UserDir(), "cache")
}

func SubscriptionDir() string {
	return filepath.Join(UserDir(), "subscriptions")
}
