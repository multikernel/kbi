package commands

import "github.com/spf13/cobra"

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Manage KBI add-on packs (modulepack, bpfpack)",
}

func init() {
	rootCmd.AddCommand(packCmd)
}
