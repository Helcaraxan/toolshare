package driver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
)

const (
	syncModeFetch = "fetch"
	syncModeShim  = "shim"
)

func Sync(cOpts *CommonOpts) *cobra.Command {
	opts := &syncOptions{
		CommonOpts: cOpts,
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
	*CommonOpts

	mode  string
	tools []string
}

func (o *syncOptions) sync() error {
	log := o.Log.With(zap.String("mode", o.mode))

	switch o.mode {
	case syncModeFetch, syncModeShim:
		log.Debug("Syncing tools.")
	default:
		log.Error("Unknown sync mode.")
		return fmt.Errorf("unknown sync mode %q", o.mode)
	}

	if len(o.tools) == 0 {
		for name := range o.Env {
			o.tools = append(o.tools, name)
		}
		log.Info("No tools were specified. Syncing all tools registered in the current environment.")
	}
	if len(o.tools) == 0 {
		log.Warn("No tools were synced as none are registered in the current environment.")
		return nil
	}
	log = log.With(zap.Strings("tools", o.tools))

	if err := o.syncInitShimFolder(); err != nil {
		return err
	}

	var errs []error
	for _, name := range o.tools {
		if _, ok := o.Env[name]; !ok {
			errs = append(errs, fmt.Errorf("can not create shim for tool %q that is unknown in current environment", name))
			continue
		}
		if err := o.syncCreateShim(name); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		log.Error("Failed to create shims for some tools.", zap.Errors("subscribe-errors", errs))
		return fmt.Errorf("failed to create shims for some tools: %v", errs)
	}
	log.Debug("Successfully created shims for all tools.")

	// Exit early if we are not fetching binaries.
	if o.mode != syncModeFetch {
		return nil
	}

	log.Debug("Downloading binaries for tools to sync.")
	for _, name := range o.tools {
		dl := &downloadOptions{
			CommonOpts: o.CommonOpts,
			tool:       name,
			version:    o.Env[name].Version,
			platforms:  []string{string(config.CurrentPlatform())},
			archs:      []string{string(config.CurrentArch())},
		}
		if err := dl.download(); err != nil {
			return err
		}
	}
	log.Debug("Successfully completed tool sync.")
	return nil
}

func (o *syncOptions) syncInitShimFolder() error {
	subscriptionsPath := o.subscriptionDir()
	log := o.Log.With(zap.String("subscriptions-path", subscriptionsPath))

	if _, err := os.Stat(subscriptionsPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Error("Could not assert existence of subscriptions folder.", zap.Error(err))
			return err
		}
		if err = os.MkdirAll(subscriptionsPath, 0o755); err != nil {
			log.Error("Could not create subscriptions folder.", zap.Error(err))
			return err
		}
	}

	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		p = filepath.Clean(os.ExpandEnv(p))
		if strings.EqualFold(p, subscriptionsPath) {
			return nil
		}
	}
	o.Log.Sugar().Warnf(
		"Please add %q to your 'PATH' environment variable, preferably at the front, to start using new subscriptions.",
		subscriptionsPath,
	)
	return nil
}

func (o *syncOptions) syncCreateShim(name string) error {
	const (
		cmdShimTemplate = `@ECHO OFF
%s invoke --tool=%s -- %%*
`
		shellShimTemplate = `#!/usr/bin/env sh
%s invoke --tool=%s -- "$@"
`
	)
	log := o.Log.With(zap.String("tool-name", name))

	invoker := config.DriverName
	if name == config.DriverName {
		// We protect against infinite loops. Version-management should not be done via the same
		// system as it causes a bootstrap problem.
		log.Error("Can not create shim for tool with the same name as the driver.")
		return fmt.Errorf("can not create shim for tool with the same name as the driver %q", config.DriverName)
	}

	switch runtime.GOOS {
	case "windows":
		if err := o.syncWriteShim(name+".cmd", fmt.Sprintf(cmdShimTemplate, invoker, name)); err != nil {
			log.Error("Failed to write CMD shim.", zap.Error(err))
			return err
		}
		fallthrough
	default:
		if err := o.syncWriteShim(name, fmt.Sprintf(shellShimTemplate, invoker, name)); err != nil {
			log.Error("Failed to write shell shim", zap.Error(err))
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
		o.Log.Error("Failed to open a temporary file to write shim.", zap.Error(err))
		return err
	}

	if _, err = shim.WriteString(content); err != nil {
		o.Log.Error("Unable to write shim file.", zap.Error(err))
		return err
	} else if err = shim.Close(); err != nil {
		o.Log.Error("Unable to close temporary shim file.", zap.Error(err))
		return err
	} else if err = os.Chmod(shim.Name(), 0o755); err != nil {
		o.Log.Error("Unable to make the temporary shim file executable.", zap.Error(err))
		return err
	}

	if err = os.Rename(shim.Name(), filepath.Join(shimDir, name)); err != nil {
		o.Log.Error("Unable to move temporary shim file to final path.", zap.Error(err))
		return err
	}
	o.Log.Debug("Successfully wrote subscription shim.")
	return nil
}

func (o *syncOptions) subscriptionDir() string {
	return filepath.Join(config.UserDir(), "subscriptions")
}
