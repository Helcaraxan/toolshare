package driver

import (
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/backend"
	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/state"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

func NewDownloadCommand(log *logrus.Logger, settings *config.Global) *cobra.Command {
	opts := &downloadOptions{
		log:      log,
		settings: settings,
	}

	cmd := &cobra.Command{
		Use:   "download --tool=<name> [--version=<version>] [--platforms=<darwin,...>] [--arch=<amd64,...>]",
		Short: "Download a tool to the local cache.",
		Long: `Download one or more binaries for a tool at a given version to the local cache. It is possible to
specify one or more platforms for which to fetch the binaries as well as an architecture. This can
for example be used when mounting a binary into a docker container for an OS different from the one
the host is running.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return download(opts)
		},
	}

	downloadFlags(cmd, opts)

	return cmd
}

func downloadFlags(cmd *cobra.Command, opts *downloadOptions) {
	cmd.Flags().StringVar(&opts.arch, "arch", "", "The architecture for which to download binaries.")
	cmd.Flags().StringSliceVar(&opts.platforms, "platforms", nil, "The platforms for which to download binaries.")
	cmd.Flags().StringVar(&opts.tool, "tool", "", "The tool for which to download binaries.")
	cmd.Flags().StringVar(&opts.version, "version", "", "The version of the tool for which to download binaries.")

	_ = cmd.MarkFlagRequired("tool")
}

type downloadOptions struct {
	log      *logrus.Logger
	settings *config.Global

	tool      string
	version   string
	platforms []string
	arch      string
}

func download(opts *downloadOptions) error {
	if len(opts.platforms) == 0 {
		opts.platforms = []string{runtime.GOOS}
	}
	if opts.arch == "" {
		opts.arch = runtime.GOARCH
	}

	if opts.version == "" {
		if version, ok := envPinnedTools(opts.log)[opts.tool]; ok {
			opts.version = version.version
		} else if opts.settings.DisallowUnpinned {
			opts.log.Errorf(
				"Please specify an explicit version for %q as it is not pinned and unpinned tools are actively prohibited in the current settings.",
				opts.tool,
			)
			return errFail
		} else {
			s := state.NewCache(opts.log, opts.settings.Root, opts.settings.State)
			if err := s.Refresh(true); err != nil {
				return err
			}

			version, err := s.RecommendedVersion(opts.tool)
			if err != nil {
				return err
			} else if opts.version == "" {
				opts.log.Errorf("No known default version for %q. Please ensure the tool exists and specify an explicit version.", opts.tool)
				return errFail
			}
			opts.version = version
		}
	}

	s := backend.NewCache(opts.log, opts.settings.Root, opts.settings.Storage)
	for _, platform := range opts.platforms {
		b := tool.Binary{
			Tool:     opts.tool,
			Version:  opts.version,
			Platform: tool.Platform(platform),
			Arch:     tool.Arch(opts.arch),
		}
		path, err := s.Get(b)
		if err != nil {
			return err
		}
		opts.log.Debugf("Binary for %s available at %q.", b, path)
	}

	return nil
}
