package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

var exeFile *os.File = nil

func lockSelf() error {
	if exeFile != nil {
		return unix.EWOULDBLOCK
	}
	path, absErr := filepath.Abs(os.Args[0])
	if absErr != nil {
		return absErr
	}
	selfFile, openErr := os.OpenFile(path, os.O_RDONLY, 0444)
	if openErr != nil {
		return openErr
	}
	lockErr := unix.Flock(int(selfFile.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if lockErr != nil {
		_ = selfFile.Close()
		return lockErr
	}
	exeFile = selfFile
	return nil
}

func isAlreadyLocked(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK)
}
