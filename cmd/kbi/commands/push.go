package commands

import (
	"fmt"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <reference>",
	Short: "Push a KBI image to an OCI registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store := oci.DefaultStore()
	if !store.Exists(ref) {
		return fmt.Errorf("image %s not found locally — run 'kbi build' first", ref)
	}
	img, err := store.Load(ref)
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}
	if err := oci.Push(ref, img); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	fmt.Printf("Pushed %s\n", ref)
	return nil
}
