package commands

import (
	"fmt"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <reference>",
	Short: "Pull a KBI image from an OCI registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	ref := args[0]
	img, err := oci.Pull(ref)
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	store := oci.DefaultStore()
	if err := store.Save(ref, img); err != nil {
		return fmt.Errorf("saving image: %w", err)
	}
	fmt.Printf("Pulled %s\n", ref)
	return nil
}
