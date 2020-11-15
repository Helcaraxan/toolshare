package types

import "fmt"

type Binary struct {
	Tool     string
	Version  string
	Platform string
	Arch     string
}

func (b *Binary) String() string {
	return fmt.Sprintf("%s-%s-%s@%s", b.Tool, b.Platform, b.Arch, b.Version)
}
