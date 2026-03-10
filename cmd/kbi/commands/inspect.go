package commands

import (
	"fmt"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/spf13/cobra"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <reference>",
	Short: "Inspect a KBI image",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store := oci.DefaultStore()

	var img v1.Image
	var err error

	if store.Exists(ref) {
		img, err = store.Load(ref)
	} else {
		img, err = oci.Pull(ref)
	}
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("reading digest: %w", err)
	}

	annotations := manifest.Annotations
	fmt.Printf("KBI ID:      %s\n", annotations[oci.AnnotationKBIID])
	fmt.Printf("Kernel:      %s\n", annotations[oci.AnnotationKver])
	fmt.Printf("Arch:        %s\n", annotations[oci.AnnotationArch])
	fmt.Printf("Components:  %s\n", annotations[oci.AnnotationComponents])
	fmt.Printf("Digest:      %s\n", digest)

	return nil
}
