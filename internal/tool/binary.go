package tool

import (
	"fmt"
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
