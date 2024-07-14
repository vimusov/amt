package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type pkgDesc struct {
	name   string
	chksum string
	size   uint64
}

func namesFromDescs(descs []pkgDesc) []string {
	result := make([]string, 0, len(descs))
	for _, desc := range descs {
		result = append(result, desc.name)
	}
	return result
}

func calcChkSum(path string) (string, error) {
	pkgFile, openErr := os.Open(path)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := pkgFile.Close(); closeErr != nil {
			panic(fmt.Sprintf("Unable calc chksum: %s.", path))
		}
	}()
	hasher := sha256.New()
	copySize, copyErr := io.Copy(hasher, pkgFile)
	if copyErr != nil {
		return "", copyErr
	}
	if copySize == 0 {
		return "", fmt.Errorf("empty file '%s'", path)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func getPkgsToUpdate(sectionDir string, allPkgs []pkgDesc) ([]pkgDesc, error) {
	total := len(allPkgs)
	broken := 0
	missing := 0
	prefix := "Checking packages"
	message := ""
	result := make([]pkgDesc, 0)
	if !defPrinter.isVerbose() {
		defPrinter.info("%s...", prefix)
	}
	for idx, desc := range allPkgs {
		message = fmt.Sprintf("\r%s: %d/%d", prefix, idx+1, total)
		path := filepath.Join(sectionDir, desc.name)
		if isFileExist(path) {
			chksum, calcErr := calcChkSum(path)
			if calcErr != nil {
				return nil, calcErr
			}
			if chksum != desc.chksum {
				broken++
				result = append(result, desc)
			}
		} else {
			missing++
			result = append(result, desc)
		}
		defPrinter.progress(message)
	}
	defPrinter.progress("\r" + strings.Repeat(" ", len(message)) + "\r")
	if broken == 0 && missing == 0 {
		defPrinter.line("%s: OK.", prefix)
	} else {
		defPrinter.line("%s: %d missing, %d broken.", prefix, missing, broken)
	}
	return result, nil
}
