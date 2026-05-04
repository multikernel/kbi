package commands

import (
	"fmt"

	"github.com/multikernel/kbi/pkg/oci"
	kbiresolve "github.com/multikernel/kbi/pkg/resolve"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve <kbi-reference> [pack-reference...]",
	Short: "Resolve a KBI image with compatible add-on packs",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runResolve,
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}

func runResolve(cmd *cobra.Command, args []string) error {
	store := oci.DefaultStore()
	kbiRef := args[0]

	kbiImg, err := loadImage(store, kbiRef)
	if err != nil {
		return fmt.Errorf("loading KBI image %s: %w", kbiRef, err)
	}

	packInputs := make([]kbiresolve.PackInput, 0, len(args)-1)
	for _, ref := range args[1:] {
		img, err := loadImage(store, ref)
		if err != nil {
			return fmt.Errorf("loading pack image %s: %w", ref, err)
		}
		packInputs = append(packInputs, kbiresolve.PackInput{Ref: ref, Image: img})
	}

	resolved, err := kbiresolve.Resolve(kbiRef, kbiImg, packInputs)
	if err != nil {
		return err
	}

	fmt.Printf("Resolved KBI: %s\n", resolved.KBIRef)
	fmt.Printf("KBI ID:       %s\n", resolved.KBIID)
	fmt.Printf("Kernel:       %s\n", resolved.Kver)
	fmt.Printf("Arch:         %s\n", resolved.Arch)
	fmt.Printf("Components:   %s\n", resolved.Components)
	if len(resolved.Packs) == 0 {
		fmt.Println("Packs:        (none)")
		return nil
	}

	fmt.Println("Packs:")
	for _, p := range resolved.Packs {
		fmt.Printf("  %s %s\n", p.Type, p.Ref)
		fmt.Printf("    Contents: %s\n", p.Contents)
		if p.Requires != "" {
			fmt.Printf("    Requires: %s\n", p.Requires)
		}
		if p.BPFManifest != "" {
			fmt.Printf("    BPF Manifest: %s\n", p.BPFManifest)
		}
		if p.BPFPrograms != "" {
			fmt.Printf("    BPF Programs: %s\n", p.BPFPrograms)
		}
		if p.BPFKfuncs != "" {
			fmt.Printf("    BPF Kfuncs: %s\n", p.BPFKfuncs)
		}
		if p.BPFTypes != "" {
			fmt.Printf("    BPF Types: %s\n", p.BPFTypes)
		}
	}

	return nil
}
