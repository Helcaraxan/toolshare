//go:build unix

package flock

import (
	"os"
	"syscall"
)

func processIsRunning(pid int) bool {
	proc, _ := os.FindProcess(pid)
	return proc.Signal(syscall.Signal(0)) == nil
}
