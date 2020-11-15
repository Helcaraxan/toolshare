package driver

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/improbable/toolshare/internal/config"
	"github.com/improbable/toolshare/internal/state"
)

func NewVersionsCommand(log *logrus.Logger, settings *config.Settings) *cobra.Command {
	opts := &versionOpts{
		log:      log,
		settings: settings,
	}

	cmd := &cobra.Command{
		Use:     "versions <tool> [--count=<n>]",
		Aliases: []string{"list-versions"},
		Short:   "List the versions available for a tool.",
		Args:    cobra.ExactArgs(1),
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return state.NewCache(opts.log, opts.settings.Root, opts.settings.State).Refresh(true)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.tool = args[0]
			return versions(opts)
		},
	}

	versionFlags(cmd, opts)

	return cmd
}

func versionFlags(cmd *cobra.Command, opts *versionOpts) {
	cmd.Flags().IntVar(&opts.count, "count", 10, "Number of versions to list. The default version will always be printed.")
}

type versionOpts struct {
	log      *logrus.Logger
	settings *config.Settings

	tool  string
	count int
}

func versions(opts *versionOpts) error {
	s := state.NewCache(opts.log, opts.settings.Root, opts.settings.State)

	recommended, err := s.RecommendedVersion(opts.tool)
	if err != nil {
		return err
	} else if recommended == "" {
		opts.log.Errorf("No versions available for %q. Did you spell the toolname correctly?", opts.tool)
		return errFail
	}
	fmt.Println(recommended, "(recommended)")

	versions, err := s.AvailableVersions(opts.tool)
	if err != nil {
		return err
	}

	var printCount int
	for _, version := range versions {
		if printCount >= opts.count {
			break
		}

		if version != recommended {
			fmt.Println(version)
			printCount++
		}
	}
	return nil
}
