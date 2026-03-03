package main

import (
	"os"

	"github.com/aschreifels/cwt/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
