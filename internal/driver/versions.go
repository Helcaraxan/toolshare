package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func Versions(cOpts *commonOpts) *cobra.Command {
	opts := &versionOpts{
		commonOpts: cOpts,
	}

	cmd := &cobra.Command{
		Use:     "versions <tool> [--count=<n>]",
		Aliases: []string{"list-versions"},
		Short:   "List the versions available for a config.",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.tool = args[0]
			return opts.versions()
		},
	}

	registerVersionFlags(cmd, opts)

	return cmd
}

func registerVersionFlags(cmd *cobra.Command, opts *versionOpts) {
	cmd.Flags().IntVar(&opts.count, "count", 10, "Number of versions to list. The default version will always be printed.")
}

type versionOpts struct {
	*commonOpts

	tool  string
	count int
}

func (o *versionOpts) versions() error {
	// TODO - requires the use of state.
	return errors.New("not yet implemented")
}
