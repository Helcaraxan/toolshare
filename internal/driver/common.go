package main

import (
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
)

type commonOpts struct {
	log    *logrus.Logger
	config *config.Global
	env    environment.Environment
}
