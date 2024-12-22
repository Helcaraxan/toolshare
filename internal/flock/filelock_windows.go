//go:build windows

package flock

import (
	"os"
)

func processIsRunning(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}
