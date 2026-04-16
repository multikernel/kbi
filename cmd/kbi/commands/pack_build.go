package commands

import (
	"fmt"
	"os"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/multikernel/kbi/pkg/pack"
	"github.com/spf13/cobra"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var packBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a KBI add-on pack image",
	RunE:  runPackBuild,
}

var (
	packBuildType      string
	packBuildModules   string
	packBuildBPF       string
	packBuildFor       string
	packBuildForKBIID  string
	packBuildTag       string
	packBuildArch      string
)

func init() {
	packBuildCmd.Flags().StringVar(&packBuildType, "type", "", "pack type: modulepack or bpfpack (required)")
	packBuildCmd.Flags().StringVarP(&packBuildModules, "modules", "m", "", "modules directory (for modulepack)")
	packBuildCmd.Flags().StringVar(&packBuildBPF, "bpf", "", "BPF programs directory (for bpfpack)")
	packBuildCmd.Flags().StringVar(&packBuildFor, "for", "", "target KBI image reference (optional, triggers validation)")
	packBuildCmd.Flags().StringVar(&packBuildForKBIID, "for-kbi-id", "", "stamp KBI ID directly (optional)")
	packBuildCmd.Flags().StringVarP(&packBuildTag, "tag", "t", "", "output image reference (required)")
	packBuildCmd.Flags().StringVar(&packBuildArch, "arch", "", "architecture (required if --for not specified)")
	packBuildCmd.MarkFlagRequired("type")
	packBuildCmd.MarkFlagRequired("tag")
	packCmd.AddCommand(packBuildCmd)
}

func loadImage(store *oci.Store, ref string) (v1.Image, error) {
	if store.Exists(ref) {
		return store.Load(ref)
	}
	img, err := oci.Pull(ref)
	if err != nil {
		return nil, err
	}
	if err := store.Save(ref, img); err != nil {
		// Non-fatal: log warning but return the image anyway
		fmt.Fprintf(os.Stderr, "warning: could not cache image %s: %v\n", ref, err)
	}
	return img, nil
}

func runPackBuild(cmd *cobra.Command, args []string) error {
	// Validate --type
	packType := pack.PackType(packBuildType)
	if packType != pack.PackTypeModule && packType != pack.PackTypeBPF {
		return fmt.Errorf("--type must be %q or %q, got %q", pack.PackTypeModule, pack.PackTypeBPF, packBuildType)
	}

	// Reject if both --for and --for-kbi-id are set
	if packBuildFor != "" && packBuildForKBIID != "" {
		return fmt.Errorf("--for and --for-kbi-id are mutually exclusive")
	}

	// Require source directory based on type
	var sourcePath string
	switch packType {
	case pack.PackTypeModule:
		if packBuildModules == "" {
			return fmt.Errorf("--modules/-m is required for modulepack")
		}
		sourcePath = packBuildModules
	case pack.PackTypeBPF:
		if packBuildBPF == "" {
			return fmt.Errorf("--bpf is required for bpfpack")
		}
		sourcePath = packBuildBPF
	}

	p := &pack.Pack{
		Type:       packType,
		SourcePath: sourcePath,
		Tag:        packBuildTag,
	}

	store := oci.DefaultStore()

	if packBuildFor != "" {
		// Load or pull the target KBI image and read its annotations
		img, err := loadImage(store, packBuildFor)
		if err != nil {
			return fmt.Errorf("loading target KBI image %s: %w", packBuildFor, err)
		}

		manifest, err := img.Manifest()
		if err != nil {
			return fmt.Errorf("reading manifest of %s: %w", packBuildFor, err)
		}

		annotations := manifest.Annotations
		p.ForRef = packBuildFor
		p.ForKBIID = annotations[oci.AnnotationKBIID]
		p.ForKver = annotations[oci.AnnotationKver]
		p.Arch = annotations[oci.AnnotationArch]

		// For bpfpack, validate that the target KBI image has BTF
		if packType == pack.PackTypeBPF {
			components := annotations[oci.AnnotationComponents]
			if err := pack.ValidateBPF(components); err != nil {
				return fmt.Errorf("BPF validation against target KBI: %w", err)
			}
		}
	} else {
		// No --for: use --for-kbi-id and --arch directly
		if packBuildArch == "" {
			return fmt.Errorf("--arch is required when --for is not specified")
		}
		p.ForKBIID = packBuildForKBIID
		p.Arch = packBuildArch
	}

	img, err := pack.BuildPack(p)
	if err != nil {
		return fmt.Errorf("building pack: %w", err)
	}

	if err := store.Save(packBuildTag, img); err != nil {
		return fmt.Errorf("saving pack image: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return err
	}

	digest, err := img.Digest()
	if err != nil {
		return err
	}

	annotations := manifest.Annotations
	fmt.Printf("Built pack image: %s\n", packBuildTag)
	fmt.Printf("Type:        %s\n", annotations[pack.AnnotationPackType])
	if v := annotations[pack.AnnotationPackForKBIID]; v != "" {
		fmt.Printf("For KBI ID:  %s\n", v)
	}
	if v := annotations[pack.AnnotationPackForKver]; v != "" {
		fmt.Printf("For Kernel:  %s\n", v)
	}
	fmt.Printf("Arch:        %s\n", annotations[oci.AnnotationArch])
	fmt.Printf("Contents:    %s\n", annotations[pack.AnnotationPackContents])
	fmt.Printf("Digest:      %s\n", digest)

	return nil
}
