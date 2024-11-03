package driver

import (
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/toolshare/internal/config"
	"github.com/Helcaraxan/toolshare/internal/environment"
	"github.com/Helcaraxan/toolshare/internal/logger"
)

var (
	ErrFailedShimCreation   = errors.New("failed to create tool shim")
	ErrInvalidCacheConfig   = errors.New("invalid cache configuration")
	ErrInvalidToolshareShim = fmt.Errorf("can not create shim for tool with the same name as the driver %q", config.DriverName)
	ErrNoToolSet            = errors.New("no tool set")
	ErrUnknownSyncMode      = errors.New("unknown sync mode")
	ErrUnknownTool          = errors.New("tool unknown in current environment")

	ErrUnimplemented = errors.New("unimplemented")
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
