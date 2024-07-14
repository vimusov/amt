package main

import (
	"archive/tar"
	"bufio"
	"cmp"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const fileChunkSize = 16 * 1048576

func loadPkgDesc(desc string) (pkgDesc, error) {
	var size uint64
	var convErr error
	var prevLine, name, chksum string
	for _, rawLine := range strings.Split(desc, "\n") {
		line := strings.TrimSpace(rawLine)
		switch prevLine {
		case "%FILENAME%":
			name = line
		case "%CSIZE%":
			size, convErr = strconv.ParseUint(line, 10, 64)
			if convErr != nil {
				size = 0
			}
		case "%SHA256SUM%":
			chksum = line
		}
		prevLine = line
		if name != "" && size != 0 && chksum != "" {
			return pkgDesc{name: name, size: size, chksum: chksum}, nil
		}
	}
	return pkgDesc{}, fmt.Errorf("unable find fields")
}

func loadDescFromDB(path string) ([]pkgDesc, error) {
	defPrinter.info("Loading package descriptions from '%s'...", filepath.Base(path))

	dbFile, openErr := os.Open(path)
	if openErr != nil {
		return nil, openErr
	}
	defer func() {
		if closeErr := dbFile.Close(); closeErr != nil {
			defPrinter.error("Unable close DB file: %s.", closeErr)
		}
	}()

	gzReader, gzErr := gzip.NewReader(bufio.NewReaderSize(dbFile, fileChunkSize))
	if gzErr != nil {
		return nil, gzErr
	}
	defer func() {
		if closeErr := gzReader.Close(); closeErr != nil {
			defPrinter.error("Unable to close gzip reader: %s.", closeErr)
		}
	}()

	pkgs := make([]pkgDesc, 0)
	dbTar := tar.NewReader(gzReader)
	content := make([]byte, 131072)

	for {
		header, tarErr := dbTar.Next()
		if tarErr != nil {
			if tarErr == io.EOF {
				break
			}
			return nil, tarErr
		}
		if header == nil {
			return nil, fmt.Errorf("invalid TAR header in '%s'", path)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := header.Name
		if filepath.Base(name) != "desc" {
			continue
		}
		readSize, readErr := dbTar.Read(content)
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("desc file '%s' is too big", name)
		}
		if readSize == 0 {
			return nil, fmt.Errorf("zero bytes read from 'desc' file '%s'", name)
		}
		pd, loadErr := loadPkgDesc(string(content))
		if loadErr != nil {
			return nil, fmt.Errorf("%s: %w", name, loadErr)
		}
		pkgs = append(pkgs, pd)
	}

	slices.SortFunc(pkgs, func(one, two pkgDesc) int { return cmp.Compare(two.size, one.size) })
	return pkgs, nil
}
