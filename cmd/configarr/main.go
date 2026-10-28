package main

import (
	"os"

	"configarr/internal/app"
)

// main runs ConfigArr and reports failures with a nonzero exit status.
func main() {
	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
