package driver

import (
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/state"
)

func NewListCommand(log *logrus.Logger, settings *config.Settings) *cobra.Command {
	opts := &listOptions{
		log:      log,
		settings: settings,
	}

	cmd := &cobra.Command{
		Use:   "list [pattern]",
		Args:  cobra.MaximumNArgs(1),
		Short: "List all tools available for subscription and use.",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return state.NewCache(opts.log, opts.settings.Root, opts.settings.State).Refresh(true)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.pattern = args[0]
			}
			return list(opts)
		},
	}

	return cmd
}

type listOptions struct {
	log      *logrus.Logger
	settings *config.Settings

	pattern string
}

func list(opts *listOptions) error {
	m := regexp.MustCompile(`^.*$`)
	if opts.pattern != "" {
		var err error
		m, err = regexp.Compile(opts.pattern)
		if err != nil {
			opts.log.WithError(err).Errorf("Invalid pattern %s.", opts.pattern)
			return err
		}
	}

	tools, err := state.NewCache(opts.log, opts.settings.Root, opts.settings.State).AvailableTools()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		if m.MatchString(tool) {
			fmt.Println(tool)
		}
	}
	return nil
}
