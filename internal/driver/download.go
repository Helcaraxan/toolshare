package driver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/backend"
	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/flock"
)

func Download(cOpts *CommonOpts) *cobra.Command {
	opts := &downloadOptions{
		CommonOpts: cOpts,
	}

	cmd := &cobra.Command{
		Use:   "download --tool=<name> [--version=<version>] [--platforms=<darwin,...>] [--arch=<amd64,...>]",
		Short: "Download a tool to the local cache.",
		Long: `Download one or more binaries for a tool at a given version to the local cache. It is possible to
specify one or more platforms for which to fetch the binaries as well as an architecture. This can
for example be used when mounting a binary into a docker container for an OS different from the one
the host is running.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.download()
		},
	}

	registerDownloadFlags(cmd, opts)

	return cmd
}

func registerDownloadFlags(cmd *cobra.Command, opts *downloadOptions) {
	cmd.Flags().StringSliceVar(&opts.archs, "archs", []string{runtime.GOARCH}, "The architecture(s) for which to download binaries.")
	cmd.Flags().StringSliceVar(&opts.platforms, "platforms", []string{runtime.GOOS}, "The platform(s) for which to download binaries.")
	cmd.Flags().StringVar(&opts.tool, "tool", "", "The tool for which to download binaries.")
	cmd.Flags().StringVar(&opts.version, "version", "", "The version of the tool for which to download binaries.")

	_ = cmd.MarkFlagRequired("tool")
}

type downloadOptions struct {
	*CommonOpts

	tool      string
	version   string
	platforms []string
	archs     []string
}

func (o downloadOptions) download() error {
	if o.tool == "" {
		o.Log.Error("No tool was specified.")
		return ErrNoToolSet
	}
	log := o.Log.With(zap.String("tool-name", o.tool))

	var archs []config.Arch
	var platforms []config.Platform
	for _, a := range o.archs {
		archs = append(archs, config.Arch(a))
	}
	for _, p := range o.platforms {
		platforms = append(platforms, config.Platform(p))
	}

	if o.version == "" {
		tool, ok := o.Env[o.tool]
		if !ok {
			log.Error("Tool could not be found in the current toolshare environment. Use 'toolshare env' go get an overview of currently registered tools.")
			return ErrUnknownTool
		}
		o.version = tool.Version
		if o.version == "" {
			log.Error("Tool was not found or could not be resolved to a version to use.")
			os.Exit(invokeExitCode)
		}
	}

	backends, err := o.setupBackends()
	if err != nil {
		return err
	}

	var errs []error
	for _, platform := range platforms {
		for _, arch := range archs {
			b := config.Binary{
				Tool:     o.tool,
				Version:  o.version,
				Platform: platform,
				Arch:     arch,
			}
			p, err := o.getToolBinary(backends, b)
			if err != nil {
				errs = append(errs, err)
			} else {
				log.Debug("Binary available.", zap.Stringer("tool", b), zap.String("binary-path", p))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to fetch some binaries: %w", errors.Join(errs...))
	}
	return nil
}

type storages struct {
	local  backend.BinaryProvider
	remote backend.Storage
	source backend.Storage
}

func (o downloadOptions) setupBackends() (*storages, error) {
	cacheURLTemplate := []string{"v1", "{tool}", "{version}", "{platform}", "{arch}", "{tool}{exe}"}

	backends := &storages{
		local: backend.NewFileSystem(o.LogBuilder, &backend.FileSystemConfig{
			FilePathTemplate: filepath.Join(append([]string{config.StorageDir()}, cacheURLTemplate...)...),
		}),
		source: o.Env.Source(o.LogBuilder, o.tool),
	}

	if o.Config.RemoteCache != nil {
		switch {
		case o.Config.RemoteCache.GCSBucket != "":
			backends.remote = backend.NewGCS(o.LogBuilder, &backend.GCSConfig{
				GCSBucket:       o.Config.RemoteCache.GCSBucket,
				GCSPathTemplate: strings.Join(append([]string{o.Config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		case o.Config.RemoteCache.HTTPSHost != "":
			backends.remote = backend.NewHTTPS(o.LogBuilder, &backend.HTTPSConfig{
				HTTPSURLTemplate: strings.Join(append([]string{o.Config.RemoteCache.HTTPSHost, o.Config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		case o.Config.RemoteCache.S3Bucket != "":
			backends.remote = backend.NewS3(o.LogBuilder, &backend.S3Config{
				S3Bucket:       o.Config.RemoteCache.S3Bucket,
				S3PathTemplate: strings.Join(append([]string{o.Config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		case o.Config.RemoteCache.PathPrefix != "":
			backends.remote = backend.NewFileSystem(o.LogBuilder, &backend.FileSystemConfig{
				FilePathTemplate: strings.Join(append([]string{o.Config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		default:
			return nil, ErrInvalidCacheConfig
		}
		o.Log.Debug("Configured remote cache backend.", zap.Stringer("remote-cache", backends.remote))
	}

	return backends, nil
}

func (o downloadOptions) getToolBinary(backends *storages, binary config.Binary) (string, error) {
	path := backends.local.Path(binary)
	log := o.Log.With(zap.Stringer("tool", binary), zap.String("cache-path", path), zap.Int("pid", os.Getpid()))

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Error("Failed to prepare target folder for tool download.")
		return "", err
	}

	for {
		if _, err := os.Stat(path); err == nil {
			log.Debug("Found binary in local storage.")
			return path, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			log.Error("Could not determine presence of tool binary.", zap.Error(err))
			return "", err
		}
		log.Debug("Binary not present in local storage.")

		ok, err := flock.AcquireFileLock(log, path)
		if err != nil {
			log.Error("Failed to acquire download lock.", zap.Error(err))
		} else if ok {
			break
		}
	}

	defer func() {
		if err := flock.ReleaseFileLock(log, path); err != nil {
			log.Warn("Failed to release download lock correctly.", zap.Error(err))
		}
	}()

	fetchErr := ErrNoBackends
	for _, s := range []backend.Storage{backends.remote, backends.source} {
		if s == nil {
			continue
		}

		sLog := log.With(zap.Stringer("storage", s))
		sLog.Debug("Attempting to fetch binary.")

		var raw []byte
		raw, fetchErr = s.Fetch(binary)
		sLog.Debug("Fetched binary from storage.")
		if fetchErr == nil {
			if err := backends.local.Store(binary, raw); err != nil {
				log.Debug("Failed to store binary in local cache.", zap.Error(err))
				return "", err
			}
			log.Debug("Successfully stored binary in local cache.")
			break
		}
	}
	if fetchErr != nil {
		return "", fetchErr
	}
	return path, nil
}
