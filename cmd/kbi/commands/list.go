package commands

import (
	"fmt"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/multikernel/kbi/pkg/pack"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local KBI images and packs",
	RunE:  runList,
}

var (
	listForKBIID string
	listType     string
)

func init() {
	listCmd.Flags().StringVar(&listForKBIID, "for-kbi-id", "", "filter packs by target KBI ID")
	listCmd.Flags().StringVar(&listType, "type", "", "filter by type: kbi, modulepack, bpfpack")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	store := oci.DefaultStore()

	refs, err := store.List()
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}

	if len(refs) == 0 {
		fmt.Println("No images in local store.")
		return nil
	}

	printed := 0
	for _, ref := range refs {
		img, err := store.Load(ref)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not load %s: %v\n", ref, err)
			continue
		}

		manifest, err := img.Manifest()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not read manifest for %s: %v\n", ref, err)
			continue
		}

		annotations := manifest.Annotations
		packType := annotations[pack.AnnotationPackType]
		kbiID := annotations[oci.AnnotationKBIID]
		forKBIID := annotations[pack.AnnotationPackForKBIID]

		// Determine the image kind
		kind := "kbi"
		if packType != "" {
			kind = packType
		}

		// Apply --type filter
		if listType != "" && kind != listType {
			continue
		}

		// Apply --for-kbi-id filter: match KBI images by their ID, packs by their target ID
		if listForKBIID != "" {
			if kbiID != listForKBIID && forKBIID != listForKBIID {
				continue
			}
		}

		printed++
		switch kind {
		case "kbi":
			fmt.Printf("%-12s %s\n", "kbi", ref)
			fmt.Printf("  KBI ID:  %s\n", kbiID)
			fmt.Printf("  Kernel:  %s  Arch: %s\n", annotations[oci.AnnotationKver], annotations[oci.AnnotationArch])
		default:
			fmt.Printf("%-12s %s\n", kind, ref)
			if forKBIID != "" {
				fmt.Printf("  For KBI: %s\n", forKBIID)
			}
			fmt.Printf("  Kernel:  %s  Arch: %s\n", annotations[pack.AnnotationPackForKver], annotations[oci.AnnotationArch])
			fmt.Printf("  Contents: %s\n", annotations[pack.AnnotationPackContents])
		}
	}

	if printed == 0 {
		fmt.Println("No matching images.")
	}

	return nil
}
