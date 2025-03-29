package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	netChunkSize = 65536
	userAgent    = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

func getFile(url, path string, idx, amount uint) (string, error) {
	request, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		return "", reqErr
	}
	request.Header.Set("User-Agent", userAgent)

	client := http.Client{}
	respose, respErr := client.Do(request)
	if respErr != nil {
		return "", respErr
	}
	defer func() {
		if closeErr := respose.Body.Close(); closeErr != nil {
			defPrinter.putError("Unable to close response body: %s.", closeErr)
		}
	}()

	fp, openErr := os.Create(path)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := fp.Close(); closeErr != nil {
			defPrinter.putError("Unable to close pkg file: %s.", closeErr)
		}
	}()

	totalSize := respose.ContentLength
	if totalSize < 1 {
		return "", fmt.Errorf("download too small")
	}

	hasher := sha256.New()
	buf := make([]byte, netChunkSize)
	pb := newProgressBar(idx, amount, filepath.Base(path), totalSize)
	pb.begin()
	curSize := int64(0)
	for {
		readSize, readErr := respose.Body.Read(buf)
		if readErr != nil && readErr != io.EOF {
			return "", readErr
		}
		if readSize == 0 {
			break
		}
		writeSize, writeError := fp.Write(buf[:readSize])
		if writeError != nil {
			return "", writeError
		}
		if writeSize != readSize {
			return "", fmt.Errorf("read/write size mismatch: %d/%d", readSize, writeSize)
		}
		_, _ = hasher.Write(buf[:readSize])
		curSize += int64(writeSize)
		pb.draw(curSize)
	}
	pb.end()
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func downloadBases(baseUrl, sectionDir string, names []string) error {
	amount := uint(len(names))
	for i, name := range names {
		path := filepath.Join(sectionDir, name)
		if rmErr := rmFile(path); rmErr != nil {
			return rmErr
		}
		url := fmt.Sprintf("%s/%s", baseUrl, name)
		if _, downErr := getFile(url, path, uint(i+1), amount); downErr != nil {
			defPrinter.putError("Unable to download file: %s.", downErr)
			return downErr
		}
	}
	return nil
}

func downloadPkgs(baseUrl, sectionDir string, pkgs []pkgDesc) ([]pkgDesc, error) {
	var broken []pkgDesc
	amount := uint(len(pkgs))
	for i, pkg := range pkgs {
		name := pkg.name
		idx := uint(i + 1)
		url := fmt.Sprintf("%s/%s", baseUrl, name)
		path := filepath.Join(sectionDir, name)
		wrongSum := true
		downFailed := true
		for attemptsLeft := 2; attemptsLeft > 0; attemptsLeft-- {
			if rmErr := rmFile(path); rmErr != nil {
				return nil, rmErr
			}
			realSum, downErr := getFile(url, path, idx, amount)
			if downErr != nil {
				defPrinter.putError("Unable to download file '%s': %s.", name, downErr)
				downFailed = true
				continue
			}
			downFailed = false
			if realSum != pkg.chksum {
				defPrinter.putError("Checksum mismatch: %s vs %s.", realSum, pkg.chksum)
				wrongSum = true
				continue
			}
			wrongSum = false
			break
		}
		if downFailed || wrongSum {
			broken = append(broken, pkg)
		}
	}
	return broken, nil
}
