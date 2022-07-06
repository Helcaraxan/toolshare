package main

import (
	"fmt"
	"sort"

	"github.com/ryanuber/columnize"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
)

func Env(log *logrus.Logger, conf *config.Global, env environment.Environment) *cobra.Command {
	opts := &envOptions{
		commonOpts: commonOpts{
			log:    log,
			config: conf,
			env:    env,
		},
	}

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Print the version of all tools available in the current environment with their version.",
		Long: fmt.Sprintf(`Displays the tool environment within as defined by the current working directory. The content of the
tool environment, meaning the available tools and their respective versions is determined as
follows:

- The list of available tools corresponds to those that have been subscribed to via a preceding call
  to the '%s subscribe' command.
- For each tool we find the first version provided by the following steps:
  - Recursively walking up the filesystem up to the root looking for '%s.yaml' files containing
    a pinned version for the config.
  - Looking at the user's configuration directory for a potential 'global.yaml' file pinning a
	version for the config. The configuration directory is '$HOME/.config/%s' on Linux and MacOS
	and '%%LOCALAPPDATA%%/%s' on Windows.
  - Looking for a system-level configuration file pinning a version for the config. This is
	'/etc/%s/toolsharerc' on Linux and MacOS and '%%PROGRAMDATA%%/%s/toolsharerc on
	Windows.
  - If, and only if, running unpinned versions is not prohibited by the local configuration we check
    the global state for the default version, if one is available.`, config.DriverName, config.DriverName, config.DriverName, config.DriverName, config.DriverName, config.DriverName),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.environment()
		},
	}

	registerEnvFlags(cmd, opts)

	return cmd
}

type envOptions struct {
	commonOpts

	full bool
}

func registerEnvFlags(cmd *cobra.Command, opts *envOptions) {
	cmd.Flags().BoolVar(&opts.full, "full", false, "Print extra information.")
}

func (o *envOptions) environment() error {
	defaultSource := "local"
	if o.config.RemoteCache != nil {
		defaultSource += " or remote"
	}
	defaultSource += " cache"

	if len(o.env) == 0 {
		fmt.Println("No tools are configured in the current environment.")
		return nil
	}

	var sortedTools []string
	for tool, reg := range o.env {
		s := defaultSource
		if reg.Source != nil {
			s = reg.Source.String()
		}
		info := fmt.Sprintf("%s | %s | %s", tool, reg.Version, s)
		if o.full {
			info += fmt.Sprintf(" | %s | %s", reg.VersionFile, reg.SourceFile)
		}
		sortedTools = append(sortedTools, info)
	}
	sort.Strings(sortedTools)

	headerRows := []string{
		"Tool | Pin | Source",
		"---- | --- | ------",
	}
	if o.full {
		headerRows[0] += " | Pin file | Source file"
		headerRows[1] += " | -------- | -----------"
	}

	fmt.Println(columnize.SimpleFormat(append(headerRows, sortedTools...)))
	return nil
}
