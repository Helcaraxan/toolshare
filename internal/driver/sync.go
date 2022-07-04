package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
)

const (
	syncModeFetch = "fetch"
	syncModeShim  = "shim"
)

func Sync(log *logrus.Logger, conf *config.Global, env *environment.Environment) *cobra.Command {
	opts := &syncOptions{
		commonOpts: commonOpts{
			log:    log,
			config: conf,
			env:    env,
		},
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync one or more tools and make them available in your $PATH.",
		Long: `Creates shim scripts for tools in the subscription folder and downloads the binaries for the
versions according to the current environment. If the subscription folder is appropriately placed at
the start of your $PATH environment variable this ensures that tools can be directly invoked at the
configured version as if they were installed directly in your $PATH.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.sync()
		},
	}

	registerSyncFlags(cmd, opts)

	return cmd
}

func registerSyncFlags(cmd *cobra.Command, opts *syncOptions) {
	cmd.Flags().StringSliceVar(
		&opts.tools,
		"tools",
		nil,
		"List of tools to sync to. If left empty all pinned tools in the current environment will be syncd to.",
	)
	cmd.Flags().StringVar(
		&opts.mode,
		"mode",
		syncModeFetch,
		"Actions to take: 'shim' to only create shim scripts, 'fetch' to download syncd-to binaries as well",
	)
}

type syncOptions struct {
	commonOpts

	mode  string
	tools []string
}

func (o *syncOptions) sync() error {
	switch o.mode {
	case syncModeFetch, syncModeShim:
		o.log.Debugf("Subscribing in %q mode.", o.mode)
	default:
		o.log.Errorf("Unknown sync mode %q.", o.mode)
		return fmt.Errorf("unknown sync mode %q", o.mode)
	}

	knownTools, err := o.commonOpts.knownTools()
	if err != nil {
		return err
	}

	if len(o.tools) == 0 {
		for name := range knownTools {
			o.tools = append(o.tools, name)
		}
		o.log.Debug("No tools were specified. Subscribing to all tools registered in the current environment.")
	}
	if len(o.tools) == 0 {
		o.log.Warn("No shims were written as no tools are registered in the current environment.")
		return nil
	}

	if err = o.syncInitShimFolder(); err != nil {
		return err
	}

	var errs []error
	for _, name := range o.tools {
		if b, ok := knownTools[name]; !ok {
			errs = append(errs, fmt.Errorf("failed to subscribe to %q, tool not known in current environment", name))
			continue
		} else if err = o.syncCreateShim(b); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to subscribe to tools: %v", errs)
	}

	// Exit early if we are not fetching binaries.
	if o.mode != syncModeFetch {
		return nil
	}

	for _, name := range o.tools {
		b := knownTools[name]
		dl := &downloadOptions{
			commonOpts: o.commonOpts,
			tool:       name,
			version:    b.Version,
			platforms:  []string{string(config.CurrentPlatform())},
			archs:      []string{string(config.CurrentArch())},
		}
		if err := dl.download(); err != nil {
			return err
		}
	}
	return nil
}

func (o *syncOptions) syncInitShimFolder() error {
	subscriptionsPath := o.subscriptionDir()
	if _, err := os.Stat(subscriptionsPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			o.log.WithError(err).Error("Could not assert existence of subscriptions folder.")
			return err
		}
		if err = os.MkdirAll(subscriptionsPath, 0o755); err != nil {
			o.log.WithError(err).Error("Could not create subscriptions folder.")
			return err
		}
	}

	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		p = filepath.Clean(os.ExpandEnv(p))
		if strings.EqualFold(p, subscriptionsPath) {
			return nil
		}
	}
	o.log.Warnf(
		"Please add %q to your 'PATH' environment variable, preferably at the front, to start using new subscriptions.",
		subscriptionsPath,
	)
	return nil
}

func (o *syncOptions) syncCreateShim(b config.Binary) error {
	const (
		cmdShimTemplate = `@ECHO OFF
%s invoke --tool=%s -- %%*
`
		shellShimTemplate = `#!/usr/bin/env sh
%s invoke --tool=%s -- "$@"
`
	)

	invoker := config.DriverName
	if b.Tool == config.DriverName {
		// We protect against infinite loops. Version-management should not be done via the same
		// system as it causes a bootstrap problem.
		return fmt.Errorf("can not create shim for tool with the same name as the driver %q", config.DriverName)
	}

	switch runtime.GOOS {
	case "windows":
		if err := o.syncWriteShim(b.Tool+".cmd", fmt.Sprintf(cmdShimTemplate, invoker, b.Tool)); err != nil {
			return err
		}
		fallthrough
	default:
		if err := o.syncWriteShim(b.Tool, fmt.Sprintf(shellShimTemplate, invoker, b.Tool)); err != nil {
			return err
		}
	}
	return nil
}

// Shim-files should normally not change over time. However we want to ensure that regenerating
// these files is safe, even when the shim-files are also being used at the same time by a different
// process.
//
// It is in the the nature of shell interpreters that they may lazily read the script file. If a
// different processes overwrites the script file at the same time this may cause an unintentional
// failure. The work-around is to create a new temporary file that contains the new shim's content
// after which we use a file rename to replace the existing shim file. This has the effect that any
// existing file-descriptors for the old file used by shell interpreters rename valid and continue
// to point to the old content while any new file descriptors will appropriately read the new
// content.
func (o *syncOptions) syncWriteShim(name string, content string) error {
	shimDir := o.subscriptionDir()

	shim, err := ioutil.TempFile(shimDir, name)
	if err != nil {
		o.log.WithError(err).Errorf("Failed to open a temporary file to write shim for tool %q.", name)
		return err
	}

	if _, err = shim.WriteString(content); err != nil {
		o.log.WithError(err).Errorf("Unable to write shim file for %q.", name)
		return err
	} else if err = shim.Close(); err != nil {
		o.log.WithError(err).Errorf("Unable to close temporary shim file for %q.", name)
		return err
	} else if err = os.Chmod(shim.Name(), 0o755); err != nil {
		o.log.WithError(err).Error("Unable to make the temporary shim file executable.")
		return err
	}

	if err = os.Rename(shim.Name(), filepath.Join(shimDir, name)); err != nil {
		o.log.WithError(err).Error("Unable to move temporary shim file to final path.")
		return err
	}
	o.log.Debugf("Wrote the subscription shim for %q to %q", name, filepath.Join(shimDir, name))
	return nil
}

func (o *syncOptions) subscriptionDir() string {
	return filepath.Join(config.GetUserConfigDir(), "subscriptions")
}
