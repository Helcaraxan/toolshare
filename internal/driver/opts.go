package driver

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

type CommonOpts struct {
	LogBuilder logger.Builder
	Log        *zap.Logger
	Config     *config.Global
	Env        environment.Environment
	Verbose    []string
}

func NewCommonOpts() *CommonOpts {
	return &CommonOpts{
		LogBuilder: logger.NewBuilder(os.Stderr),
		Config:     &config.Global{},
		Env:        environment.Environment{},
	}
}

func (c *CommonOpts) Parse() error {
	for _, domain := range c.Verbose {
		c.LogBuilder.SetDomainLevel(domain, zapcore.DebugLevel)
	}
	c.Log = c.LogBuilder.Domain(logger.CLIDomain)

	if err := config.Parse(c.LogBuilder.Domain(logger.InitDomain), c.Config); err != nil {
		return err
	}

	if err := environment.GetEnvironment(c.Config, c.Env); err != nil {
		return err
	}
	return nil
}
