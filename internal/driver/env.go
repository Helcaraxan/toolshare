package driver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/improbable/toolshare/internal/config"
	"github.com/improbable/toolshare/internal/state"
)

var errFail = errors.New("failed")

func NewEnvCommand(log *logrus.Logger, settings *config.Settings) *cobra.Command {
	opts := &envOptions{
		log:      log,
		settings: settings,
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
  - Recursively walking up the filesystem up to the root looking for '.toolsharerc' files containing
    a pinned version for the tool.
  - Looking at the user's configuration directory for a potential 'toolsharerc' file pinning a
	version for the tool. The configuration directory is '$HOME/.config/%s' on Linux and MacOS
	and '%%LOCALAPPDATA%%/%s' on Windows.
  - Looking for a system-level configuration file pinning a version for the tool. This is
	'/etc/%s/toolsharerc' on Linux and MacOS and '%%PROGRAMDATA%%/%s/toolsharerc on
	Windows.
  - If, and only if, running unpinned versions is not prohibited by the local configuration we check
    the global state for the default version.`, config.DriverName, config.DriverName, config.DriverName, config.DriverName, config.DriverName),
		Args: cobra.NoArgs,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return state.NewCache(opts.log, opts.settings.Root, opts.settings.State).Refresh(false)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return env(opts)
		},
	}

	return cmd
}

type envOptions struct {
	log      *logrus.Logger
	settings *config.Settings
}

func env(opts *envOptions) error {
	infos, err := ioutil.ReadDir(filepath.Join(opts.settings.Root, subscribeFolder))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			opts.log.Error("Environment is empty. No subscriptions were found.")
		} else {
			opts.log.WithError(err).Error("Failed to read the content of the subscriptions folder.")
		}
		return err
	}

	candidates := map[string]envTool{}
	for _, info := range infos {
		if info.IsDir() || info.Mode() != 0755 {
			// We don't expect any folders or non-executable files amongst the subscriptions but we
			// skip any we encounter to be safe.
			continue
		}
		if filepath.Ext(info.Name()) == "" {
			// We don't count any executable with an extension as shims shouldn't have one.
			candidates[filepath.Base(info.Name())] = envTool{}
		}
	}

	for tool, pin := range envPinnedTools(opts.log) {
		if _, ok := candidates[tool]; ok {
			candidates[tool] = pin
		}
	}

	if !opts.settings.DisallowUnpinned {
		s := state.NewCache(opts.log, opts.settings.Root, opts.settings.State)
		for tool, env := range candidates {
			if env.version == "" {
				if env.version, err = s.RecommendedVersion(tool); err != nil {
					return err
				}
				env.source = "recommended version from global state"
			}
		}
	}

	tools := envTools([]envTool{})
	for tool, env := range candidates {
		if env.version != "" {
			tools = append(tools, envTool{
				tool:    tool,
				version: env.version,
				source:  env.source,
			})
		}
	}
	sort.Sort(tools)

	for _, tool := range tools {
		fmt.Println(tool)
	}
	return nil
}

func envPinnedTools(log *logrus.Logger) map[string]envTool {
	tools := map[string]envTool{}

	wd, err := os.Getwd()
	if err != nil {
		log.WithError(err).Warn("Failed to determine the current path.")
		return tools
	}

	for wd != filepath.Dir(wd) {
		envReadPinFile(log, filepath.Join(wd, ".toolsharerc"), tools)
		wd = filepath.Dir(wd)
	}

	switch runtime.GOOS {
	case "windows":
		envReadPinFile(log, filepath.Join(os.Getenv("LOCALAPPDATA"), config.DriverName, "toolsharerc"), tools)
		envReadPinFile(log, filepath.Join(os.Getenv("PROGRAMDATA"), config.DriverName, "toolsharerc"), tools)
	default:
		envReadPinFile(log, filepath.Join(os.Getenv("HOME"), ".config", config.DriverName, "toolsharerc"), tools)
		envReadPinFile(log, filepath.Join("/etc", config.DriverName, "toolsharerc"), tools)
	}

	return tools
}

func envReadPinFile(log *logrus.Logger, path string, tools map[string]envTool) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		log.Warnf("Failed to read potential pin file at %q.", path)
		return
	}

	pins := &config.PinFile{}
	if err = yaml.Unmarshal(b, pins); err != nil {
		log.WithError(err).Warnf("Failed to unmarshal pin file at %q.", path)
		return
	}

	for _, pin := range pins.PinnedTools {
		if _, ok := tools[pin.Tool]; !ok {
			tools[pin.Tool] = envTool{
				tool:    pin.Tool,
				version: pin.Version,
				source:  fmt.Sprintf("pinned in %q", path),
			}
		}
	}
}

type envTool struct {
	tool    string
	version string
	source  string
}

func (t *envTool) String() string {
	return fmt.Sprintf("%s @ %s - %s", t.tool, t.version, t.source)
}

type envTools []envTool

func (t envTools) Len() int               { return len(t) }
func (t envTools) Swap(i int, j int)      { t[i], t[j] = t[j], t[i] }
func (t envTools) Less(i int, j int) bool { return t[i].tool < t[j].tool }
