package driver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Helcaraxan/toolshare/internal/config"
)

func Invoke(cOpts *CommonOpts) *cobra.Command {
	opts := &invokeOptions{
		CommonOpts: cOpts,
	}

	cmd := &cobra.Command{
		Use:   "invoke --tool=<tool-name> [--] [<tool-args>]",
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
	*CommonOpts

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
	if o.tool == "" {
		o.Log.Error("No tool was specified.")
		return ErrNoToolSet
	}
	log := o.Log.With(zap.String("tool-name", o.tool))

	version := o.version
	if version == "" {
		version = o.Env[o.tool].Version
	}
	if version == "" {
		log.Error("Tool was not found or could not be resolved to a version to use")
		os.Exit(invokeExitCode)
	}
	log = log.With(zap.String("tool-version", version))

	path, err := o.ensureTool(log, version)
	if err != nil {
		return err
	}
	log = log.With(zap.String("binary-path", path))

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
	go invokeSignalForwarder(log, cmd, sigs, done)
	signal.Notify(sigs)

	log.Debug("Invoking tool binary.")
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			log.Error("Failed to invoke the tool binary.", zap.Error(err))
			os.Exit(invokeExitCode)
			return err
		}
	}

	close(done) // This should result in an os.Exit() call from the signal forwarder.

	time.Sleep(1 * time.Second)
	log.Error("Unexpected failure to shut down binary via signal forwarder.")
	os.Exit(invokeExitCode)
	return nil
}

func (o *invokeOptions) ensureTool(log *zap.Logger, version string) (string, error) {
	dl := &downloadOptions{
		CommonOpts: o.CommonOpts,
		tool:       o.tool,
		version:    version,
		platforms:  []string{string(config.CurrentPlatform())},
		archs:      []string{string(config.CurrentArch())},
	}
	backends, err := dl.setupBackends()
	if err != nil {
		log.Error("Failed to prepare storage backends.", zap.Error(err))
		os.Exit(invokeExitCode)
	}
	path, err := dl.getToolBinary(backends, config.Binary{
		Tool:     o.tool,
		Version:  version,
		Platform: config.CurrentPlatform(),
		Arch:     config.CurrentArch(),
	})
	if err != nil {
		log.Error("Failed to fetch tool.", zap.Error(err))
		return "", err
	}
	return path, nil
}

func invokeSignalForwarder(log *zap.Logger, cmd *exec.Cmd, sigs chan os.Signal, done chan struct{}) {
	log.Debug("Starting signal forwarder.")
	for {
		select {
		case sig, ok := <-sigs:
			if !ok {
				log.Error("Unexpected closure of signal forwarding.")
				os.Exit(invokeExitCode)
			}
			log.Debug("Received signal. Forwarding it to the binary.", zap.Stringer("signal", sig))

			if sig == os.Interrupt {
				go invokeTimeBomb(log, done)

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
				log.Debug("Tool process exited.", zap.Int("exit-code", cmd.ProcessState.ExitCode()))
				os.Exit(cmd.ProcessState.ExitCode())
			} else {
				log.Error("Unable to determine the exit status of the invocation.")
				os.Exit(invokeExitCode)
			}
		}
	}
}

func invokeTimeBomb(log *zap.Logger, done chan struct{}) {
	const gracePeriod = 30 * time.Second

	log = log.With(zap.Duration("grace-period", gracePeriod))
	log.Debug("Staring interrupt time-bomb.")
	select {
	case <-done:
		return
	case <-time.After(gracePeriod):
		log.Warn("Tool failed to exit after receiving interrupt. Forcefully exiting the driver.")
		os.Exit(invokeExitCode)
	}
}

func invokeForwardSignal(log *zap.Logger, cmd *exec.Cmd, sig os.Signal) {
	defer func() {
		if p := recover(); p != nil {
			log.Debug("Recovered from panic while forwarding signal to the invoked binary.", zap.Stringer("signal", sig))
		}
	}()
	if err := cmd.Process.Signal(sig); err != nil {
		log.Debug("Could not forward signal to the invoked binary.", zap.Stringer("signal", sig), zap.Error(err))
	}
}
