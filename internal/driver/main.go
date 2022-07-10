package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type commonOpts struct {
	logBuilder logger.Builder
	log        *zap.Logger
	config     *config.Global
	env        environment.Environment
}

func main() {
	cOpts := &commonOpts{
		logBuilder: logger.NewBuilder(os.Stderr),
		config:     &config.Global{},
		env:        environment.Environment{},
	}

	var verbose []string
	rootCmd := &cobra.Command{
		Use: config.DriverName,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			for _, domain := range verbose {
				cOpts.logBuilder.SetDomainLevel(domain, zapcore.DebugLevel)
			}
			cOpts.log = cOpts.logBuilder.Domain(logger.CLIDomain)

			if err := config.Parse(cOpts.logBuilder.Domain(logger.InitDomain), cOpts.config); err != nil {
				return err
			}

			if err := environment.GetEnvironment(cOpts.config, cOpts.env); err != nil {
				return err
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringSliceVarP(
		&verbose,
		"verbose",
		"v",
		nil,
		"Verbose output. See 'gomod --help' for more information.",
	)
	rootCmd.Flag("verbose").NoOptDefVal = "all"

	rootCmd.AddCommand(
		Download(cOpts),
		Env(cOpts),
		Invoke(cOpts),
		Sync(cOpts),
		Versions(cOpts),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
