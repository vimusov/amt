package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	netChunkSize = 65536
	userAgent    = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

func getFile(url, path string, idx, amount uint) error {
	fp, openErr := os.Create(path)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := fp.Close(); closeErr != nil {
			defPrinter.error("Unable to close pkg file: %s.", closeErr)
		}
	}()

	client := &http.Client{Timeout: 2 * time.Minute}
	request, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		return reqErr
	}
	request.Header.Add("User-Agent", userAgent)

	respose, respErr := client.Do(request)
	if respErr != nil {
		return respErr
	}
	defer func() {
		if closeErr := respose.Body.Close(); closeErr != nil {
			defPrinter.error("Unable to close response body: %s.", closeErr)
		}
	}()

	curSize := int64(0)
	totalSize := respose.ContentLength
	if totalSize < 1 {
		return fmt.Errorf("download too small")
	}
	buf := make([]byte, netChunkSize)

	pb := newProgressBar(idx, amount, filepath.Base(path), totalSize)
	pb.begin()
	for {
		readSize, readErr := respose.Body.Read(buf)
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if readSize == 0 {
			break
		}
		writeSize, writeError := fp.Write(buf[:readSize])
		if writeError != nil {
			return writeError
		}
		if writeSize != readSize {
			return fmt.Errorf("read/write size mismatch: %d/%d", readSize, writeSize)
		}
		curSize += int64(readSize)
		pb.draw(curSize)
	}
	pb.end()
	return fp.Sync()
}

func downloadFiles(baseUrl, sectionDir string, names []string) error {
	amount := uint(len(names))
	for i, name := range names {
		path := filepath.Join(sectionDir, name)
		if rmErr := rmFile(path); rmErr != nil {
			return rmErr
		}
		idx := uint(i + 1)
		url := fmt.Sprintf("%s/%s", baseUrl, name)
		if downErr := getFile(url, path, idx, amount); downErr != nil {
			defPrinter.error("Unable to download file: %s.", downErr)
			// Second attempt.
			if rmErr := rmFile(path); rmErr != nil {
				return rmErr
			}
			downErr = getFile(url, path, idx, amount)
			if downErr != nil {
				return downErr
			}
		}
	}
	return nil
}
