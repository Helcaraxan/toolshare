package main

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
)

func Download(cOpts *commonOpts) *cobra.Command {
	opts := &downloadOptions{
		commonOpts: cOpts,
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
	*commonOpts

	tool      string
	version   string
	platforms []string
	archs     []string
}

func (o downloadOptions) download() error {
	if o.tool == "" {
		o.log.Error("No tool was specified.")
		return errors.New("no tool set")
	}
	log := o.log.With(zap.String("tool-name", o.tool))

	var archs []config.Arch
	var platforms []config.Platform
	for _, a := range o.archs {
		archs = append(archs, config.Arch(a))
	}
	for _, p := range o.platforms {
		platforms = append(platforms, config.Platform(p))
	}

	if o.version == "" {
		o.version = o.env[o.tool].Version
		if o.version == "" {
			log.Error("Tool was not found or could not be resolved to a version to use.")
			os.Exit(invokeExitCode)
		}
	}

	local, remote, source, err := o.setupBackends()
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
			p, err := o.getToolBinary(local, remote, source, b)
			if err != nil {
				errs = append(errs, err)
			} else {
				log.Debug("Binary available.", zap.Stringer("tool", b), zap.String("binary-path", p))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to fetch some binaries: %v", errs)
	}
	return nil
}

func (o downloadOptions) setupBackends() (local backend.BinaryProvider, remote backend.Storage, source backend.Storage, err error) {
	cacheURLTemplate := []string{"v1", "{tool}", "{version}", "{platform}", "{arch}", "{tool}{exe}"}

	local = backend.NewFileSystem(o.logBuilder, &backend.FileSystemConfig{
		FilePathTemplate: filepath.Join(append([]string{config.GetStorageDir()}, cacheURLTemplate...)...),
	}, false)

	if o.config.RemoteCache != nil {
		switch {
		case o.config.RemoteCache.GCSBucket != "":
			remote = backend.NewGCS(o.logBuilder, &backend.GCSConfig{
				GCSBucket:       o.config.RemoteCache.GCSBucket,
				GCSPathTemplate: strings.Join(append([]string{o.config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		case o.config.RemoteCache.HTTPSHost != "":
			remote = backend.NewHTTPS(o.logBuilder, &backend.HTTPSConfig{
				HTTPSURLTemplate: strings.Join(append([]string{o.config.RemoteCache.HTTPSHost, o.config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		case o.config.RemoteCache.S3Bucket != "":
			remote = backend.NewS3(o.logBuilder, &backend.S3Config{
				S3Bucket:       o.config.RemoteCache.S3Bucket,
				S3PathTemplate: strings.Join(append([]string{o.config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			})
		case o.config.RemoteCache.PathPrefix != "":
			remote = backend.NewFileSystem(o.logBuilder, &backend.FileSystemConfig{
				FilePathTemplate: strings.Join(append([]string{o.config.RemoteCache.PathPrefix}, cacheURLTemplate...), "/"),
			}, false)
		default:
			return nil, nil, nil, errors.New("invalid remote cache configuration")
		}
	}

	source = o.env.Source(o.logBuilder, o.tool)

	return local, remote, source, nil
}

func (o downloadOptions) getToolBinary(local backend.BinaryProvider, remote backend.Storage, source backend.Storage, b config.Binary) (string, error) {
	path := local.Path(b)

	log := o.log.With(zap.Stringer("tool", b), zap.String("cache-path", path))
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("Could not determine presence of tool binary.", zap.Error(err))
		return "", err
	}

	var fetchErr error
	for _, s := range []backend.Storage{remote, source} {
		sLog := log.With(zap.Stringer("storage", s))
		sLog.Debug("Attempting to fetch binary from storage.")
		if s != nil {
			var raw []byte
			raw, fetchErr = s.Fetch(b)
			sLog.Debug("Fetched binary from storage.")
			if fetchErr == nil {
				if err := local.Store(b, raw); err != nil {
					log.Debug("Failed to store binary in local cache.", zap.Error(err))
					return "", err
				}
				log.Debug("Successfully stored binary in local cache.")
				break
			}
		}
	}
	if fetchErr != nil {
		return "", fetchErr
	}
	return path, nil
}
