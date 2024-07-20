package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const (
	netChunkSize    = 65536
	minThreadedSize = 10485760
	rangeUnits      = "bytes"
	userAgent       = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

func downPart(ek *errKeeper, url string, fp *os.File, start, end int64, report chan<- int64) {
	defer ek.done()

	request, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		ek.set(reqErr)
		return
	}
	var rangeVal string
	if end == -1 {
		rangeVal = fmt.Sprintf("%s=%d-", rangeUnits, start)
	} else {
		rangeVal = fmt.Sprintf("%s=%d-%d", rangeUnits, start, end-1)
	}
	request.Header.Set("Range", rangeVal)
	request.Header.Set("User-Agent", userAgent)

	client := http.Client{}
	respose, respErr := client.Do(request)
	if respErr != nil {
		ek.set(respErr)
		return
	}
	defer func() {
		if closeErr := respose.Body.Close(); closeErr != nil {
			defPrinter.error("Unable to close response body: %s.", closeErr)
		}
	}()

	buf := make([]byte, netChunkSize)
	off := start
	for {
		readSize, readErr := respose.Body.Read(buf)
		if readErr != nil && readErr != io.EOF {
			ek.set(readErr)
			return
		}
		if readSize == 0 {
			break
		}
		writeSize, writeError := fp.WriteAt(buf[:readSize], off)
		if writeError != nil {
			ek.set(writeError)
			return
		}
		if writeSize != readSize {
			ek.set(fmt.Errorf("read/write size mismatch: %d/%d", readSize, writeSize))
			return
		}
		off += int64(writeSize)
		report <- int64(readSize)
	}
}

func getFile(url, path string, threads, idx, amount uint) error {
	request, reqErr := http.NewRequest("HEAD", url, nil)
	if reqErr != nil {
		return reqErr
	}
	request.Header.Add("User-Agent", userAgent)

	client := http.Client{}
	respose, respErr := client.Do(request)
	if respErr != nil {
		return respErr
	}
	if closeErr := respose.Body.Close(); closeErr != nil {
		defPrinter.error("Unable to close response body: %s.", closeErr)
	}

	totalSize := respose.ContentLength
	if totalSize < 1 {
		return fmt.Errorf("download too small")
	}
	if respose.Header.Get("Accept-Ranges") != rangeUnits {
		return fmt.Errorf("server not support Range header")
	}
	if totalSize <= minThreadedSize {
		threads = 1 // Limit threads to one for small files.
	}

	fp, openErr := os.Create(path)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := fp.Close(); closeErr != nil {
			defPrinter.error("Unable to close pkg file: %s.", closeErr)
		}
	}()
	if truncErr := truncFile(fp, totalSize); truncErr != nil {
		return truncErr
	}

	report := make(chan int64, 4096)
	errKeep := newErrKeeper(int(threads))

	partSize := totalSize / int64(threads)
	rangeStart := int64(0)
	rangeEnd := partSize
	for i := uint(0); i < threads-1; i++ {
		go downPart(errKeep, url, fp, rangeStart, rangeEnd, report)
		rangeStart += partSize
		rangeEnd += partSize
	}
	go downPart(errKeep, url, fp, rangeStart, -1, report)

	barWg := sync.WaitGroup{}
	barWg.Add(1)
	go func() {
		defer barWg.Done()
		pb := newProgressBar(idx, amount, filepath.Base(path), totalSize)
		pb.begin()
		curSize := int64(0)
		for readSize := range report {
			curSize += readSize
			pb.draw(curSize)
		}
		pb.end()
	}()

	downErr := errKeep.get()
	close(report)
	barWg.Wait()

	if syncErr := fp.Sync(); syncErr != nil {
		return syncErr
	}
	return downErr
}

func downloadFiles(baseUrl, sectionDir string, names []string, threads uint) error {
	var lastErr error = nil
	amount := uint(len(names))
	for i, name := range names {
		path := filepath.Join(sectionDir, name)
		if rmErr := rmFile(path); rmErr != nil {
			return rmErr
		}
		idx := uint(i + 1)
		url := fmt.Sprintf("%s/%s", baseUrl, name)
		if downErr := getFile(url, path, threads, idx, amount); downErr != nil {
			defPrinter.error("Unable to download file: %s.", downErr)
			// Second attempt.
			if rmErr := rmFile(path); rmErr != nil {
				return rmErr
			}
			downErr = getFile(url, path, threads, idx, amount)
			if downErr != nil {
				lastErr = downErr
			}
		}
	}
	return lastErr
}
