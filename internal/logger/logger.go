package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Domain uint8

const (
	UnknownDomain Domain = iota
	AllDomain
	InitDomain
	CLIDomain
	FileSystemDomain
	GCSDomain
	GitHubDomain
	HTTPSDomain
	S3Domain
)

var (
	domainFromString = map[string]Domain{
		"all":    AllDomain,
		"init":   InitDomain,
		"cli":    CLIDomain,
		"fs":     FileSystemDomain,
		"gcs":    GCSDomain,
		"github": GitHubDomain,
		"https":  HTTPSDomain,
		"s3":     S3Domain,
	}

	stringFromDomain = map[Domain]string{
		AllDomain:        "all",
		InitDomain:       "init",
		CLIDomain:        "cli",
		FileSystemDomain: "fs",
		GCSDomain:        "gcs",
		GitHubDomain:     "github",
		HTTPSDomain:      "https",
		S3Domain:         "s3",
	}
)

type Builder struct {
	log          *zap.Logger
	defaultLevel zapcore.Level
	domainLevels map[Domain]zapcore.Level
	cache        map[Domain]*zap.Logger
}

func NewBuilder(out zapcore.WriteSyncer) *Builder {
	enc := newEncoder()
	return &Builder{
		log:          zap.New(zapcore.NewCore(enc, out, zapcore.DebugLevel)),
		defaultLevel: zap.InfoLevel,
		domainLevels: map[Domain]zapcore.Level{},
		cache:        map[Domain]*zap.Logger{},
	}
}

func (b *Builder) SetDomainLevel(domain string, level zapcore.Level) {
	d := domainFromString[domain]
	switch d {
	case UnknownDomain:
		b.log.Warn("Unrecognised logger domain.")
	case AllDomain:
		b.defaultLevel = level
	case InitDomain, CLIDomain, FileSystemDomain, GCSDomain, GitHubDomain, HTTPSDomain, S3Domain:
		b.domainLevels[d] = level
	default:
		panic(fmt.Sprintf("unexpected domain %q", d))
	}
}

func (b *Builder) Domain(domain Domain) *zap.Logger {
	return b.logger(domain)
}

func (b *Builder) logger(domain Domain) *zap.Logger {
	if _, ok := b.cache[domain]; !ok {
		targetLevel := b.defaultLevel
		if lvl, ok := b.domainLevels[domain]; ok {
			targetLevel = lvl
		}
		b.cache[domain] = b.log.Named(stringFromDomain[domain]).WithOptions(zap.IncreaseLevel(targetLevel))
	}
	return b.cache[domain]
}
