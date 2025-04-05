package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

var exeFile *os.File = nil

func lockSelf() error {
	if exeFile != nil {
		return unix.EWOULDBLOCK
	}
	path, exeErr := os.Executable()
	if exeErr != nil {
		return fmt.Errorf("failed to get executable path: %w", exeErr)
	}
	selfFile, openErr := os.OpenFile(path, os.O_RDONLY, 0444)
	if openErr != nil {
		return fmt.Errorf("failed to open executable: %w", openErr)
	}
	lockErr := unix.Flock(int(selfFile.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if lockErr != nil {
		_ = selfFile.Close()
		return fmt.Errorf("failed to lock executable: %w", lockErr)
	}
	exeFile = selfFile
	return nil
}

func isAlreadyLocked(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK)
}
