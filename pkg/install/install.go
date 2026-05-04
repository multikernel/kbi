package install

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/multikernel/kbi/pkg/oci"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Install extracts kernel artifacts from an OCI image into a bare metal filesystem rooted at dest.
func Install(img v1.Image, kver string, dest string) error {
	if _, err := os.Stat(dest); err != nil {
		return fmt.Errorf("dest %q does not exist: %w", dest, err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("getting layers: %w", err)
	}

	for i, desc := range manifest.Layers {
		layer := layers[i]
		switch desc.MediaType {
		case types.MediaType(oci.MediaTypeVmlinuz):
			destPath := filepath.Join(dest, "boot", "vmlinuz-"+kver)
			if err := extractBlob(layer, destPath); err != nil {
				return fmt.Errorf("extracting vmlinuz: %w", err)
			}
		case types.MediaType(oci.MediaTypeInitrd):
			destPath := filepath.Join(dest, "boot", "initrd.img-"+kver)
			if err := extractBlob(layer, destPath); err != nil {
				return fmt.Errorf("extracting initrd: %w", err)
			}
		case types.MediaType(oci.MediaTypeModules):
			destDir := filepath.Join(dest, "lib", "modules")
			if err := extractTar(layer, destDir); err != nil {
				return fmt.Errorf("extracting modules: %w", err)
			}
		case types.MediaType(oci.MediaTypeFirmware):
			destDir := filepath.Join(dest, "lib", "firmware")
			if err := extractTar(layer, destDir); err != nil {
				return fmt.Errorf("extracting firmware: %w", err)
			}
		case types.MediaType(oci.MediaTypeKernelConfig):
			destPath := filepath.Join(dest, "boot", "config-"+kver)
			if err := extractBlob(layer, destPath); err != nil {
				return fmt.Errorf("extracting kernel config: %w", err)
			}
		case types.MediaType(oci.MediaTypeBTF):
			destPath := filepath.Join(dest, "boot", "btf-"+kver)
			if err := extractBlob(layer, destPath); err != nil {
				return fmt.Errorf("extracting btf: %w", err)
			}
		}
	}

	return nil
}

// extractBlob reads the uncompressed layer content and writes it to destPath,
// creating parent directories as needed.
func extractBlob(layer v1.Layer, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creating parent dirs for %q: %w", destPath, err)
	}

	rc, err := layer.Uncompressed()
	if err != nil {
		return fmt.Errorf("reading layer content: %w", err)
	}
	defer rc.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %q: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("writing %q: %w", destPath, err)
	}
	return nil
}

// extractTar reads the layer as a tar archive and extracts entries to destDir,
// including path traversal protection.
func extractTar(layer v1.Layer, destDir string) error {
	rc, err := layer.Uncompressed()
	if err != nil {
		return fmt.Errorf("reading layer content: %w", err)
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Path traversal protection: clean the name and ensure it stays within destDir.
		name := filepath.Clean(hdr.Name)
		if name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry %q would escape destination directory", hdr.Name)
		}

		target := filepath.Join(destDir, name)
		// Double-check resolved path is within destDir.
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) && target != filepath.Clean(destDir) {
			return fmt.Errorf("tar entry %q resolves outside destination directory", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("creating directory %q: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent dirs for %q: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("creating file %q: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("writing file %q: %w", target, err)
			}
			f.Close()
		case tar.TypeSymlink, tar.TypeLink:
			// KBI tarballs do not contain links (TarDirectory skips symlinks at pack time),
			// so encountering one means the archive was produced by something untrusted.
			return fmt.Errorf("tar entry %q is a link (type %d), which is not supported", hdr.Name, hdr.Typeflag)
		}
	}
	return nil
}
