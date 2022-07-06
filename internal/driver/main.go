package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
)

func main() {
	log := logrus.New()

	var conf config.Global
	env := map[string]environment.ToolRegistration{}

	rootCmd := &cobra.Command{
		Use: config.DriverName,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := config.Parse(log, &conf); err != nil {
				return err
			}

			if err := environment.GetEnvironment(&conf, env); err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.AddCommand(
		Download(log, &conf, env),
		Env(log, &conf, env),
		Invoke(log, &conf, env),
		Sync(log, &conf, env),
		Versions(log, &conf, env),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
