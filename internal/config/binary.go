package config

import (
	"fmt"
	"runtime"
)

type Binary struct {
	Tool     string
	Version  string
	Platform Platform
	Arch     Arch
}

func (b *Binary) String() string {
	return fmt.Sprintf("%s-%s-%s@%s", b.Tool, b.Platform, b.Arch, b.Version)
}

type Platform string

const (
	PlatformDarwin  Platform = "darwin"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
)

type Arch string

const (
	ArchARM32 Arch = "arm32"
	ArchARM64 Arch = "arm64"
	ArchX86   Arch = "x86"
	ArchX64   Arch = "x86_64"
)

func CurrentPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformDarwin
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	default:
		panic("unsupported GOOS " + runtime.GOOS)
	}
}

func CurrentArch() Arch {
	switch runtime.GOARCH {
	case "386":
		return ArchX86
	case "amd64":
		return ArchX64
	case "arm":
		return ArchARM32
	case "arm64":
		return ArchARM64
	default:
		panic("unsupported GOARCH " + runtime.GOARCH)
	}
}
