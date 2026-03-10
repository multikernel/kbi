package commands

import (
	"fmt"
	"os"

	kbiinstall "github.com/multikernel/kbi/pkg/install"
	"github.com/multikernel/kbi/pkg/oci"
	"github.com/spf13/cobra"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var installCmd = &cobra.Command{
	Use:   "install <reference>",
	Short: "Install a KBI image to local filesystem",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

var installDest string

func init() {
	installCmd.Flags().StringVar(&installDest, "dest", "/", "destination root path")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	ref := args[0]

	// Validate destination early before potentially expensive pull
	if _, err := os.Stat(installDest); err != nil {
		return fmt.Errorf("destination %s does not exist: %w", installDest, err)
	}

	store := oci.DefaultStore()

	var img v1.Image
	var err error

	if store.Exists(ref) {
		img, err = store.Load(ref)
	} else {
		fmt.Printf("Image not found locally, pulling %s...\n", ref)
		img, err = oci.Pull(ref)
		if err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}
		if err := store.Save(ref, img); err != nil {
			return fmt.Errorf("saving image: %w", err)
		}
	}
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}
	kver := manifest.Annotations[oci.AnnotationKver]
	if kver == "" {
		return fmt.Errorf("image missing kver annotation")
	}

	if err := kbiinstall.Install(img, kver, installDest); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	fmt.Printf("Installed %s (kernel %s) to %s\n", ref, kver, installDest)
	return nil
}
