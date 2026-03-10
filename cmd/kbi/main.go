package main

import (
	"os"

	"github.com/multikernel/kbi/cmd/kbi/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
