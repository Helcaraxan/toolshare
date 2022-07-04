package main

import (
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
)

type commonOpts struct {
	log    *logrus.Logger
	config *config.Global
	env    *environment.Environment
}

func (o *commonOpts) knownTools() (map[string]config.Binary, error) {
	tools := map[string]config.Binary{}
	for name, pin := range o.env.Pins {
		tools[name] = config.Binary{
			Tool:    name,
			Version: pin,
		}
	}

	if !o.config.ForcePinned {
		// TODO.
	}
	return tools, nil
}
