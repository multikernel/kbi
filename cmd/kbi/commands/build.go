package commands

import (
	"fmt"
	"os"

	"github.com/multikernel/kbi/pkg/bundle"
	"github.com/multikernel/kbi/pkg/oci"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a KBI image from kernel artifacts",
	RunE:  runBuild,
}

var (
	buildVmlinuz  string
	buildInitrd   string
	buildModules  string
	buildFirmware string
	buildConfig   string
	buildBTF      string
	buildTag      string
	buildKver     string
	buildArch     string
)

func init() {
	buildCmd.Flags().StringVarP(&buildVmlinuz, "vmlinuz", "k", "", "path to vmlinuz (required)")
	buildCmd.Flags().StringVarP(&buildInitrd, "initrd", "i", "", "path to initrd")
	buildCmd.Flags().StringVarP(&buildModules, "modules", "m", "", "path to modules directory, usually /lib/modules/<kver>")
	buildCmd.Flags().StringVar(&buildFirmware, "firmware", "", "path to firmware directory")
	buildCmd.Flags().StringVarP(&buildConfig, "config", "c", "", "path to kernel .config")
	buildCmd.Flags().StringVarP(&buildBTF, "btf", "b", "", "path to BTF data")
	buildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "image reference tag (required)")
	buildCmd.Flags().StringVar(&buildKver, "kver", "", "kernel version (auto-detected if omitted)")
	buildCmd.Flags().StringVar(&buildArch, "arch", "", "target architecture (auto-detected if omitted)")
	buildCmd.MarkFlagRequired("vmlinuz")
	buildCmd.MarkFlagRequired("tag")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	b := &bundle.Bundle{
		VmlinuzPath:  buildVmlinuz,
		InitrdPath:   buildInitrd,
		ModulesPath:  buildModules,
		FirmwarePath: buildFirmware,
		ConfigPath:   buildConfig,
		BTFPath:      buildBTF,
		Kver:         buildKver,
		Arch:         buildArch,
		Tag:          buildTag,
	}

	if err := b.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if b.Kver == "" {
		fmt.Fprintln(os.Stderr, "warning: --kver not set and auto-detection not yet implemented, defaulting to 'unknown'")
		b.Kver = "unknown"
	}
	if b.Arch == "" {
		fmt.Fprintln(os.Stderr, "warning: --arch not set and auto-detection not yet implemented, defaulting to 'unknown'")
		b.Arch = "unknown"
	}

	img, err := oci.BuildImage(b)
	if err != nil {
		return fmt.Errorf("building image: %w", err)
	}

	store := oci.DefaultStore()
	if err := store.Save(buildTag, img); err != nil {
		return fmt.Errorf("saving image: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return err
	}

	fmt.Printf("Built KBI image: %s\n", buildTag)
	fmt.Printf("KBI ID: %s\n", manifest.Annotations[oci.AnnotationKBIID])
	fmt.Printf("Kernel: %s\n", manifest.Annotations[oci.AnnotationKver])
	fmt.Printf("Arch:   %s\n", manifest.Annotations[oci.AnnotationArch])
	fmt.Printf("Components: %s\n", manifest.Annotations[oci.AnnotationComponents])

	return nil
}
