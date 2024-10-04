package main

import (
	"fmt"
	"os"
)

type printer struct {
	showProgress bool
}

func (p *printer) setQuiet() {
	p.showProgress = false
}

func (p *printer) line(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func (p *printer) info(format string, args ...any) {
	fmt.Printf(">>> "+format+"\n", args...)
}

func (p *printer) error(format string, args ...any) {
	if _, writeErr := fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...); writeErr != nil {
		panic(writeErr)
	}
}

func (p *printer) progress(value string) {
	if p.showProgress {
		if _, writeErr := os.Stdout.WriteString(value); writeErr != nil {
			panic(writeErr)
		}
	}
}

func (p *printer) eol() {
	fmt.Println()
}

func (p *printer) isVerbose() bool {
	return p.showProgress
}

var defPrinter = &printer{showProgress: true}
