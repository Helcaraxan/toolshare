package driver

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

	"github.com/improbable/toolshare/internal/config"
	"github.com/improbable/toolshare/internal/state"
	"github.com/improbable/toolshare/internal/storage"
	"github.com/improbable/toolshare/internal/types"
)

func NewInvokeCommand(log *logrus.Logger, settings *config.Settings) *cobra.Command {
	opts := &invokeOptions{
		log:      log,
		settings: settings,
	}

	cmd := &cobra.Command{
		Use:   "invoke [--] <tool-name> [<tool-args>]",
		Args:  cobra.MaximumNArgs(1),
		Short: "Run a tool with the given arguments.",
		Long: fmt.Sprintf(`Run a tool at a version determined by the current environment with the given arguments. For details
about how the current environment is determined please see '%s env --help'.`, config.DriverName),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.tool = args[0]
			opts.args = args[1:]
			return invoke(opts)
		},
	}

	return cmd
}

type invokeOptions struct {
	log      *logrus.Logger
	settings *config.Settings

	tool string
	args []string
}

const invokeExitCode = 128 // Used to differentiate from exit codes from an invoked process.

func invoke(opts *invokeOptions) error {
	version, err := invokeToolVersion(opts)
	if err != nil {
		return err
	}

	path, err := storage.NewCache(opts.log, opts.settings.Root, opts.settings.Storage).Get(types.Binary{
		Tool:     opts.tool,
		Version:  version,
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
	})
	if err != nil {
		return err
	}

	// Ideally we would be using a syscall.Exec() here to simply replace the driver process with the
	// target binary one. However... this is not cross-platform compatible as this pattern is not
	// supported on Windows. Hence we need to a more complex jiggle to ensure that signals are
	// forwarded, etc.
	cmd := exec.Command(path, opts.args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	done := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	go invokeSignalForwarder(opts.log, cmd, sigs, done)

	signal.Notify(sigs)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			opts.log.WithError(err).Errorf("Failed to invoke the target binary %q.", path)
			os.Exit(invokeExitCode)
			return err
		}
	}

	close(done) // This should result in an os.Exit() call from the signal forwarder.

	time.Sleep(1 * time.Second)
	opts.log.Errorf("Unexpected failure to exit %q.", config.DriverName)
	os.Exit(invokeExitCode)
	return nil
}

func invokeToolVersion(opts *invokeOptions) (string, error) {
	if pin, ok := envPinnedTools(opts.log)[opts.tool]; ok {
		return pin.version, nil
	}

	if opts.settings.DisallowUnpinned {
		opts.log.Errorf(
			"Can not invoke %q as there is no pinned version and unpinned tools are actively prohibited in the current settings.",
			opts.tool,
		)
		return "", errFail
	}

	s := state.NewCache(opts.log, opts.settings.Root, opts.settings.State)
	if err := s.Refresh(false); err != nil {
		return "", err
	}

	version, err := s.RecommendedVersion(opts.tool)
	if err != nil {
		return "", err
	} else if version == "" {
		opts.log.Errorf("Can not invoke %q as there is no pinned version and no default version registered in the global state.", opts.tool)
		return "", errFail
	}
	return version, nil
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
