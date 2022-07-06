package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
)

func Invoke(log *logrus.Logger, conf *config.Global, env environment.Environment) *cobra.Command {
	opts := &invokeOptions{
		commonOpts: commonOpts{
			log:    log,
			config: conf,
			env:    env,
		},
	}

	cmd := &cobra.Command{
		Use:   "invoke [--] <tool-name> [<tool-args>]",
		Args:  cobra.MaximumNArgs(1),
		Short: "Run a tool with the given arguments.",
		Long: fmt.Sprintf(`Run a tool at a version determined by the current environment with the given arguments. For details
about how the current environment is determined please see '%s env --help'.`, config.DriverName),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.args = args
			return opts.invoke()
		},
	}

	registerInvokeFlags(cmd, opts)

	return cmd
}

type invokeOptions struct {
	commonOpts

	tool    string
	version string
	args    []string
}

func registerInvokeFlags(cmd *cobra.Command, opts *invokeOptions) {
	cmd.Flags().StringVar(&opts.tool, "tool", "", "Name of the tool to be invoked.")
	cmd.Flags().StringVar(&opts.version, "version", "", "Override the version of the tool that should be used. Is normally determined from the environment.")

	_ = cmd.MarkFlagRequired("tool")
}

const invokeExitCode = 128 // Used to differentiate from exit codes from an invoked process.

func (o *invokeOptions) invoke() error {
	version := o.env[o.tool].Version
	if version == "" {
		o.log.Errorf("%q was not found or could not be resolved to a version to use", o.tool)
		os.Exit(invokeExitCode)
	}

	// Ensure we have the tool available to run.
	dl := &downloadOptions{
		commonOpts: o.commonOpts,
		tool:       o.tool,
		version:    version,
		platforms:  []string{string(config.CurrentPlatform())},
		archs:      []string{string(config.CurrentArch())},
	}
	local, remote, source, err := dl.setupBackends()
	if err != nil {
		o.log.Errorf("Unable to fetch tool: %v", err)
		os.Exit(invokeExitCode)
	}
	path, err := dl.getToolBinary(local, remote, source, config.Binary{
		Tool:     o.tool,
		Version:  version,
		Platform: config.CurrentPlatform(),
		Arch:     config.CurrentArch(),
	})
	if err != nil {
		return err
	}

	// Ideally we would be using a syscall.Exec() here to simply replace the driver process with the
	// target binary one. However... this is not cross-platform compatible as this pattern is not
	// supported on Windows. Hence we need to a more complex jiggle to ensure that signals are
	// forwarded, etc.
	cmd := exec.Command(path, o.args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	done := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	go invokeSignalForwarder(o.log, cmd, sigs, done)

	signal.Notify(sigs)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			o.log.WithError(err).Errorf("Failed to invoke the target binary %q.", path)
			os.Exit(invokeExitCode)
			return err
		}
	}

	close(done) // This should result in an os.Exit() call from the signal forwarder.

	time.Sleep(1 * time.Second)
	o.log.Errorf("Unexpected failure to exit %q.", config.DriverName)
	os.Exit(invokeExitCode)
	return nil
}

func invokeSignalForwarder(log *logrus.Logger, cmd *exec.Cmd, sigs chan os.Signal, done chan struct{}) {
	for {
		select {
		case sig, ok := <-sigs:
			if !ok {
				log.Error("Unexpected closure of signal forwarding.")
				os.Exit(invokeExitCode)
			}

			if sig == os.Interrupt {
				go invokeTimeBomb(log, cmd, done)

				if runtime.GOOS == "windows" {
					// Interrupt forwarding does not exist as a concept on Windows and hence we
					// replace it with a 'kill' for lack of a better alternative.
					sig = os.Kill
				}
			}

			invokeForwardSignal(log, cmd, sig)
		case <-done:
			signal.Stop(sigs)
			close(sigs)
			<-sigs
			if cmd.ProcessState != nil {
				os.Exit(cmd.ProcessState.ExitCode())
			} else {
				log.Errorf("Unable to determine the exit status of the invocation of %v.", cmd.Args)
				os.Exit(invokeExitCode)
			}
		}
	}
}

func invokeTimeBomb(log *logrus.Logger, cmd *exec.Cmd, done chan struct{}) {
	const gracePeriod = 30 * time.Second

	select {
	case <-done:
		return
	case <-time.After(gracePeriod):
		log.Warnf("Invocation of %q failed to exit after %v. Forcefully exiting the %q driver.", cmd.Args[0], gracePeriod, config.DriverName)
		os.Exit(invokeExitCode)
	}
}

func invokeForwardSignal(log *logrus.Logger, cmd *exec.Cmd, sig os.Signal) {
	defer func() {
		if p := recover(); p != nil {
			log.Debugf("Recovered from panic while forwarding %v to the invoked binary.", sig)
		}
	}()
	if err := cmd.Process.Signal(sig); err != nil {
		log.WithError(err).Debugf("Could not forward %v to the invoked binary.", sig)
	}
}
