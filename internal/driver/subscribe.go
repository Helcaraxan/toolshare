package driver

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
	"github.com/Helcaraxan/toolshare/internal/state"
)

const (
	subscribeModeFetch = "fetch"
	subscribeModeShim  = "shim"

	subscribeFolder = "subscriptions"
)

func NewSubscribeCommand(log *logrus.Logger, settings *config.Settings) *cobra.Command {
	opts := &subscribeOptions{
		log:      log,
		settings: settings,
	}

	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to one or more tools making them available in your $PATH.",
		Long: `Creates shim scripts for tools in the subscription folder. With the subscription folder
appropriately placed at the start of your $PATH environment variable this ensures that tools that
have been subscribed to can be directly invoked as if they were installed directly in your $PATH.`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if settings.DisallowUnpinned {
				// We do not use state if we are not allowing unpinned versions.
				return nil
			}

			return state.NewCache(opts.log, opts.settings.Root, opts.settings.State).Refresh(true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return subscribe(opts)
		},
	}

	subscribeFlags(cmd, opts)

	return cmd
}

func subscribeFlags(cmd *cobra.Command, opts *subscribeOptions) {
	cmd.Flags().StringSliceVar(
		&opts.tools,
		"tools",
		nil,
		"List of tools to subscribe to. If left empty all pinned tools in the current environment will be subscribed to.",
	)
	cmd.Flags().StringVar(
		&opts.mode,
		"mode",
		subscribeModeFetch,
		"Actions to take: 'shim' to only create shim scripts, 'fetch' to download subscribed-to binaries as well",
	)
}

type subscribeOptions struct {
	log      *logrus.Logger
	settings *config.Settings

	mode  string
	tools []string
}

func subscribe(opts *subscribeOptions) error {
	switch opts.mode {
	case subscribeModeFetch, subscribeModeShim:
		opts.log.Debugf("Subscribing in %q mode.", opts.mode)
	default:
		opts.log.Errorf("Unknown subscribe mode %q.", opts.mode)
		return fmt.Errorf("unknown subscribe mode %q", opts.mode)
	}

	if err := subscribeInitShimFolder(opts.log, opts.settings.Root); err != nil {
		return err
	}

	if len(opts.tools) == 0 {
		opts.log.Warn("Not subscribing to any tools as none were specified.")
		return nil
	}

	for _, tool := range opts.tools {
		if err := subscribeToTool(opts, tool); err != nil {
			return err
		}
	}

	// Exit early if we are not fetching binaries.
	if opts.mode != subscribeModeFetch {
		return nil
	}

	for _, tool := range opts.tools {
		if err := download(&downloadOptions{
			log:  opts.log,
			tool: tool,
		}); err != nil {
			return err
		}
	}

	return nil
}

func subscribeInitShimFolder(log *logrus.Logger, toolshareRoot string) error {
	subscriptionsPath := filepath.Join(toolshareRoot, subscribeFolder)
	if _, err := os.Stat(subscriptionsPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.WithError(err).Error("Could not assert existence of subscriptions folder.")
			return err
		}
		if err = os.MkdirAll(subscriptionsPath, 0o755); err != nil {
			log.WithError(err).Error("Could not create subscriptions folder.")
			return err
		}
	}

	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		p = filepath.Clean(os.ExpandEnv(p))
		if strings.EqualFold(p, subscriptionsPath) {
			return nil
		}
	}

	log.Warnf(
		"Please add %q to your 'PATH' environment variable, preferably at the front, to start using new subscriptions.",
		subscriptionsPath,
	)
	return nil
}

func subscribeToTool(opts *subscribeOptions, tool string) error {
	opts.log.Debugf("Subscribing to tool %q.", tool)

	var toolExists bool
	for pinnedTool := range envPinnedTools(opts.log) {
		if pinnedTool == tool {
			toolExists = true
			break
		}
	}

	if !toolExists {
		if opts.settings.DisallowUnpinned {
			opts.log.Errorf(
				"Can not subscribe to tool %q as it is not pinned and unpinned tools are actively prohibited in the current settings.",
				tool,
			)
			return errFail
		}

		statePath := filepath.Join(opts.settings.Root, "state", tool)
		if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
			opts.log.Errorf("Can not subscribe to tool %q as there is no known state for it", tool)
			return errFail
		}
	}

	return subscribeCreateShim(opts, tool)
}

func subscribeCreateShim(opts *subscribeOptions, tool string) error {
	const (
		cmdShimTemplate = `@ECHO OFF
%s invoke -- %s %%*
`
		shellShimTemplate = `#!/usr/bin/env sh
%s invoke -- %s "$@"
`
	)

	shimPath := filepath.Join(opts.settings.Root, subscribeFolder)

	invoker := config.DriverName
	if tool == config.DriverName {
		// A shim script for the driver tool itself should not invoke itself as that would lead to
		// an infinite shim loop. Instead we make sure to invoke the system-wide binary directly.
		switch runtime.GOOS {
		case "windows":
			invoker = filepath.Join(os.Getenv("PROGRAMFILES"), config.DriverName, config.DriverName+".exe")
		default:
			invoker = filepath.Join("/usr/local/bin", config.DriverName)
		}
	}

	switch runtime.GOOS {
	case "windows":
		if err := subscribeWriteShim(opts, tool, tool+".cmd", fmt.Sprintf(cmdShimTemplate, invoker, tool)); err != nil {
			return err
		}
		fallthrough
	default:
		if err := subscribeWriteShim(opts, tool, tool, fmt.Sprintf(shellShimTemplate, invoker, tool)); err != nil {
			return err
		}
	}

	opts.log.Infof("Wrote the subscription shim for %q to %q.", tool, shimPath)
	return nil
}

func subscribeWriteShim(opts *subscribeOptions, tool string, name string, content string) error {
	shimPath := filepath.Join(opts.settings.Root, subscribeFolder)

	shim, err := ioutil.TempFile(shimPath, tool)
	if err != nil {
		opts.log.WithError(err).Errorf("Failed to open a temporary file to write shim for tool %q.", tool)
		return err
	}

	if _, err = shim.WriteString(content); err != nil {
		opts.log.WithError(err).Errorf("Unable to write shim file for %q.", tool)
		return err
	} else if err = shim.Close(); err != nil {
		opts.log.WithError(err).Errorf("Unable to close temporary shim file for %q.", tool)
		return err
	} else if err = os.Chmod(shim.Name(), 0o755); err != nil {
		opts.log.WithError(err).Error("Unable to make the temporary shim file executable.")
		return err
	}

	if err = os.Rename(shim.Name(), filepath.Join(shimPath, name)); err != nil {
		opts.log.WithError(err).Error("Unable to move temporary shim file to final path.")
		return err
	}
	return nil
}
