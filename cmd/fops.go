package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func truncFile(fp *os.File, size int64) error {
	size -= 1
	offset, seekErr := fp.Seek(size, io.SeekStart)
	if seekErr != nil {
		return seekErr
	}
	if offset != size {
		return fmt.Errorf("unable to seek to end of file")
	}
	writeSize, writeErr := fp.Write([]byte{0})
	if writeErr != nil {
		return writeErr
	}
	if writeSize != 1 {
		return fmt.Errorf("unable write one byte")
	}
	return nil
}

func rmFile(path string) error {
	if rmErr := os.Remove(path); rmErr != nil {
		if os.IsNotExist(rmErr) {
			return nil
		}
		return rmErr
	}
	return nil
}

func isFileExist(path string) bool {
	info, infoErr := os.Stat(path)
	if infoErr == nil {
		if info.Mode().IsRegular() {
			return true
		}
		panic(fmt.Sprintf("%s: is not a file", path))
	}
	if errors.Is(infoErr, os.ErrNotExist) {
		return false
	}
	panic(fmt.Sprintf("%s: %s.", path, infoErr))
}

func mkLastUpdateStamp(rootDir string) error {
	tsFile, openErr := os.Create(filepath.Join(rootDir, "last_update_stamp"))
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := tsFile.Close(); closeErr != nil {
			defPrinter.error("Unable to close stamp file: %s.", closeErr)
		}
	}()
	_, writeErr := tsFile.WriteString(fmt.Sprintf("%s\n", time.Now().Format(time.UnixDate)))
	if writeErr != nil {
		return writeErr
	}
	return tsFile.Sync()
}

func removeRedundantFiles(sectionDir, sectionName string, pkgs []pkgDesc) error {
	serviceFiles := map[string]struct{}{
		fmt.Sprintf("%s.db", sectionName):           {},
		fmt.Sprintf("%s.db.tar.gz", sectionName):    {},
		fmt.Sprintf("%s.files", sectionName):        {},
		fmt.Sprintf("%s.files.tar.gz", sectionName): {},
	}
	pkgNames := make(map[string]struct{}, len(pkgs))
	for _, desc := range pkgs {
		pkgNames[desc.name] = struct{}{}
	}
	entries, lookErr := os.ReadDir(sectionDir)
	if lookErr != nil {
		return lookErr
	}
	var found bool
	result := make([]string, 0, len(pkgs))
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		path := info.Name()
		if info.Mode().IsDir() {
			defPrinter.error("'%s' is a directory.", path)
			continue
		}
		name := filepath.Base(path)
		_, found = pkgNames[name]
		if found {
			continue
		}
		_, found = serviceFiles[name]
		if found {
			continue
		}
		result = append(result, path)
	}
	if len(result) == 0 {
		return nil
	}
	defPrinter.info("Removing redundant files...")
	sort.Strings(result)
	for _, path := range result {
		defPrinter.line("%s", path)
		if rmErr := os.Remove(filepath.Join(sectionDir, path)); rmErr != nil {
			defPrinter.error("Unable to remove redundant file: %s.", rmErr)
		}
	}
	defPrinter.info("Cleanup completed.")
	return nil
}

func fixupSymlinks(sectionDir, sectionName string) error {
	type pair struct {
		link, arc string
	}
	exts := []pair{
		{link: "db", arc: "tar.gz"},
		{link: "files", arc: "tar.gz"},
	}
	for _, ext := range exts {
		linkName := fmt.Sprintf("%s.%s", sectionName, ext.link)
		linkPath := filepath.Join(sectionDir, linkName)
		targetName := fmt.Sprintf("%s.%s.%s", sectionName, ext.link, ext.arc)
		targetPath := filepath.Join(sectionDir, targetName)
		if !isFileExist(targetPath) {
			return fmt.Errorf("'%s' is not exist as symlink target", targetPath)
		}
		if rmErr := rmFile(linkPath); rmErr != nil {
			return rmErr
		}
		if linkErr := os.Symlink(targetName, linkPath); linkErr != nil {
			return linkErr
		}
		defPrinter.info("Symlink '%s' updated.", linkName)
	}
	return nil
}
