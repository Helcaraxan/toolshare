package flock

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

var (
	ErrNoPID         = errors.New("failed to determine PID of process holding the download lock")
	ErrNoLockRelease = errors.New("unable to release file lock")
)

const (
	pollInterval        = 100 * time.Millisecond
	pidWriteGracePeriod = 1 * time.Second
)

func AcquireFileLock(log *zap.Logger, path string) (bool, error) {
	sem, err := os.OpenFile(path+".pid", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return false, err
	} else if errors.Is(err, os.ErrExist) {
		log.Debug("Lockfile already exists. Waiting for it to be released.")
		return false, waitOnPID(log, path)
	}

	log.Debug("Acquired lock. Writing PID to file.")
	if _, err = fmt.Fprint(sem, os.Getpid()); err != nil {
		return false, err
	} else if err = sem.Close(); err != nil {
		return false, err
	}
	return true, nil
}

func ReleaseFileLock(log *zap.Logger, path string) error {
	log.Debug("Deleting lock file.")
	if err := os.Remove(path + ".pid"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("Could not delete lock file.", zap.Error(err))
		return fmt.Errorf("%w(%s): %w", ErrNoLockRelease, path+".pid", err)
	}
	return nil
}

func waitOnPID(log *zap.Logger, path string) error {
	iterations := 1
	for {
		if iterations%100 == 0 {
			log.Info("Waiting for tool download lock to be released.")
		}

		c, err := os.ReadFile(path + ".pid")
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug("Lock has been released. PID file was deleted.")
				return nil
			}
			return err
		}

		var pid int
		if _, err := fmt.Sscan(string(c), &pid); err != nil {
			log.Debug("Error reading PID.", zap.Error(err))
			// It may be that the process acquiring the lock did not yet have time to write the PID. However if we have
			// passed the defined grace period we will presume the process died before it could write and we can
			// force-release the lock to proceed.
			fi, err := os.Stat(path + ".pid")
			if err != nil {
				return err
			}
			if time.Since(fi.ModTime()) < pidWriteGracePeriod {
				continue
			}
			log.Debug("Forcing lock release after PID-write grace period expired.")
			return ReleaseFileLock(log, path)
		}

		if !processIsRunning(pid) {
			log.Debug("Forcing lock release after owning PID exited.")
			return ReleaseFileLock(log, path)
		}
		time.Sleep(pollInterval)
		iterations++
	}
}
