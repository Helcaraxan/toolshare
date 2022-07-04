package driver

import (
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
	"github.com/Helcaraxan/toolshare/internal/tool"
)

type commonOpts struct {
	log    *logrus.Logger
	config config.Global
	env    environment.Environment
}

func (o *commonOpts) knownTools() (map[string]tool.Binary, error) {
	tools := map[string]tool.Binary{}
	for name, pin := range o.env.Pins {
		tools[name] = tool.Binary{
			Tool:    name,
			Version: pin,
		}
	}

	if !o.config.ForcePinned {
		// TODO.
	}
	return tools, nil
}

func (o *commonOpts) subscriptionFolder() string {
	return filepath.Join(o.config.Root, "subscriptions")
}
