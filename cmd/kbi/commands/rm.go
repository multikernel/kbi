package commands

import (
	"fmt"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <reference>",
	Short: "Remove a KBI image or pack from local storage",
	Args:  cobra.ExactArgs(1),
	RunE:  runRm,
}

func init() {
	rootCmd.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store := oci.DefaultStore()

	if !store.Exists(ref) {
		return fmt.Errorf("image %q not found in local store", ref)
	}

	if err := store.Remove(ref); err != nil {
		return fmt.Errorf("removing image: %w", err)
	}

	fmt.Printf("Removed %s\n", ref)
	return nil
}
