package main

import (
	"fmt"
	"strings"
	"time"
)

type progressBar struct {
	idx         uint
	amount      uint
	fileName    string
	totalSize   float64
	baseTime    int64
	startTime   int64
	startSize   float64
	prevPercent int
	maxLineLen  int
}

func divmod(x, y int64) (int64, int64) {
	return x / y, x % y
}

func humanTime(etaTime float64) string {
	var days, hours, mins int64
	secs := int64(etaTime)
	mins, secs = divmod(secs, 60)
	hours, mins = divmod(mins, 60)
	days, hours = divmod(hours, 60)
	if days > 0 {
		return fmt.Sprintf("{%d}d%02d:%02d:%02d", days, hours, mins, secs)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, mins, secs)
}

func newProgressBar(idx, amount uint, fileName string, totalSize int64) *progressBar {
	return &progressBar{
		idx:         idx,
		amount:      amount,
		fileName:    fileName,
		totalSize:   float64(totalSize) / 1024.0,
		baseTime:    time.Now().Unix(),
		prevPercent: -1,
	}
}

func (pb *progressBar) begin() {
	if !defPrinter.isVerbose() {
		status := fmt.Sprintf("[%d/%d] %s: downloading...", pb.idx, pb.amount, pb.fileName)
		pb.maxLineLen = max(pb.maxLineLen, len(status))
		defPrinter.line(status)
	}
}

func (pb *progressBar) draw(curPos int64) {
	totalSize := pb.totalSize
	startSize := pb.startSize
	prevPercent := pb.prevPercent
	startTime := pb.startTime
	curSize := float64(curPos) / 1024.0
	curPercent := int(curSize * 100 / totalSize)
	if curPercent < 0 {
		curPercent = 0
	}
	curTime := time.Now().Unix() - pb.baseTime
	if prevPercent == 0 && startSize == 0 && startTime == 0 {
		startSize = curSize
		startTime = curTime
	}
	speed := 0.0
	timeElapsed := curTime - startTime
	if timeElapsed > 0 {
		speed = (curSize - startSize) / float64(timeElapsed)
	}
	etaTime := 0.0
	if speed > 0 {
		etaTime = (totalSize - curSize) / speed
	}
	speed *= 8
	speedPrefix := "Kbps"
	if speed >= 1000 {
		speed /= 1000
		speedPrefix = "Mbps"
	}
	if prevPercent != curPercent {
		sizeDelim := 1.0
		sizePrefix := "Kb"
		if totalSize >= 1024 {
			sizeDelim = 1024.0
			sizePrefix = "Mb"
		}
		status := fmt.Sprintf(
			"\r[%d/%d] %s: %.2f/%.2f %s (%d%%) @ %.0f %s ETA %s",
			pb.idx, pb.amount, pb.fileName,
			curSize/sizeDelim, totalSize/sizeDelim, sizePrefix,
			curPercent,
			speed, speedPrefix,
			humanTime(etaTime),
		)
		curLineLen := len(status)
		pb.maxLineLen = max(pb.maxLineLen, curLineLen)
		fillingLen := pb.maxLineLen - curLineLen
		if fillingLen < 0 {
			fillingLen = 0
		}
		defPrinter.progress(status + strings.Repeat(" ", fillingLen))
	}
	pb.prevPercent = curPercent
	pb.startSize = startSize
	pb.startTime = startTime
}

func (pb *progressBar) end() {
	status := fmt.Sprintf("[%d/%d] %s: done.", pb.idx, pb.amount, pb.fileName)
	fillingLen := pb.maxLineLen - len(status)
	if fillingLen < 0 {
		fillingLen = 0
	}
	prefix := ""
	if defPrinter.isVerbose() {
		prefix = "\r"
	}
	defPrinter.line(prefix + status + strings.Repeat(" ", fillingLen))
}
