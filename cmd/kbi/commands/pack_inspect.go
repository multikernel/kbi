package commands

import (
	"fmt"
	"os"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/multikernel/kbi/pkg/pack"
	"github.com/spf13/cobra"
)

var packInspectCmd = &cobra.Command{
	Use:   "inspect <reference>",
	Short: "Inspect a KBI pack image",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackInspect,
}

func init() {
	packCmd.AddCommand(packInspectCmd)
}

func runPackInspect(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store := oci.DefaultStore()

	img, err := loadImage(store, ref)
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

	packType, hasPackType := annotations[pack.AnnotationPackType]
	if !hasPackType {
		fmt.Fprintln(os.Stderr, "warning: image has no pack type annotation; may not be a KBI pack image")
	}

	fmt.Printf("Type:        %s\n", packType)
	if v := annotations[pack.AnnotationPackForKBIID]; v != "" {
		fmt.Printf("For KBI ID:  %s\n", v)
	}
	if v := annotations[pack.AnnotationPackForKver]; v != "" {
		fmt.Printf("For Kernel:  %s\n", v)
	}
	fmt.Printf("Arch:        %s\n", annotations[oci.AnnotationArch])
	fmt.Printf("Contents:    %s\n", annotations[pack.AnnotationPackContents])
	if v := annotations[pack.AnnotationPackRequires]; v != "" {
		fmt.Printf("Requires:    %s\n", v)
	}
	fmt.Printf("Digest:      %s\n", digest)

	return nil
}
