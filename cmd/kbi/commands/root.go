package commands

import (
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:           "kbi",
	Short:         "Kernel Bundle Image — package kernels as OCI artifacts",
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}
