package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/driver"
)

func main() {
	opts := driver.NewCommonOpts()

	rootCmd := &cobra.Command{
		Use: config.DriverName,
		Long: `Provide and manage tool versions for reproducible outcomes.

Create well-defined environments for developer and automation workflows by
ensuring that tools are always invoked at their expected version. Eliminate
hard-to-understand bugs and automation failures caused by tool-drift that waste
development time.

For a quickstart guide see the documentation available at:
https://github.com/Helcaraxan/toolshare/docs/setup.md
`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return opts.Parse()
		},
	}

	registerRootFlags(rootCmd, opts)

	rootCmd.AddCommand(
		driver.Download(opts),
		driver.Env(opts),
		driver.Invoke(opts),
		driver.Sync(opts),
		driver.Versions(opts),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func registerRootFlags(cmd *cobra.Command, opts *driver.CommonOpts) {
	cmd.PersistentFlags().StringSliceVarP(
		&opts.Verbose,
		"verbose",
		"v",
		nil,
		"Verbose output. See 'toolshare --help' for more information.",
	)
	cmd.Flag("verbose").NoOptDefVal = "all"
}
