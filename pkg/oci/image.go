package oci

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/multikernel/kbi/pkg/bundle"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// blobLayer implements v1.Layer for raw byte content with a custom media type.
// These layers are stored uncompressed — Compressed() and Uncompressed() return
// identical content, and Digest() == DiffID() by design. This is valid because
// KBI uses custom media types that are not expected to be gzip-compressed.
type blobLayer struct {
	content   []byte
	mediaType string
}

func (l *blobLayer) Digest() (v1.Hash, error) {
	sum := sha256.Sum256(l.content)
	return v1.NewHash("sha256:" + hex.EncodeToString(sum[:]))
}

func (l *blobLayer) DiffID() (v1.Hash, error) {
	return l.Digest()
}

func (l *blobLayer) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

func (l *blobLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

func (l *blobLayer) Size() (int64, error) {
	return int64(len(l.content)), nil
}

func (l *blobLayer) MediaType() (types.MediaType, error) {
	return types.MediaType(l.mediaType), nil
}

// BuildImage converts a Bundle into an OCI v1.Image.
func BuildImage(b *bundle.Bundle) (v1.Image, error) {
	var addenda []mutate.Addendum
	identityComponents := map[string][]byte{}
	var componentNames []string

	// vmlinuz is always required
	vmlinuzData, err := os.ReadFile(b.VmlinuzPath)
	if err != nil {
		return nil, fmt.Errorf("reading vmlinuz: %w", err)
	}
	addenda = append(addenda, mutate.Addendum{
		Layer:     &blobLayer{content: vmlinuzData, mediaType: MediaTypeVmlinuz},
		MediaType: types.MediaType(MediaTypeVmlinuz),
	})
	identityComponents["vmlinuz"] = vmlinuzData
	componentNames = append(componentNames, "vmlinuz")

	// initrd (optional, not an identity component)
	if b.InitrdPath != "" {
		initrdData, err := os.ReadFile(b.InitrdPath)
		if err != nil {
			return nil, fmt.Errorf("reading initrd: %w", err)
		}
		addenda = append(addenda, mutate.Addendum{
			Layer:     &blobLayer{content: initrdData, mediaType: MediaTypeInitrd},
			MediaType: types.MediaType(MediaTypeInitrd),
		})
		componentNames = append(componentNames, "initrd")
	}

	// config (optional, identity component)
	if b.ConfigPath != "" {
		configData, err := os.ReadFile(b.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		addenda = append(addenda, mutate.Addendum{
			Layer:     &blobLayer{content: configData, mediaType: MediaTypeKernelConfig},
			MediaType: types.MediaType(MediaTypeKernelConfig),
		})
		identityComponents["config"] = configData
		componentNames = append(componentNames, "config")
	}

	// btf (optional, identity component)
	if b.BTFPath != "" {
		btfData, err := os.ReadFile(b.BTFPath)
		if err != nil {
			return nil, fmt.Errorf("reading btf: %w", err)
		}
		addenda = append(addenda, mutate.Addendum{
			Layer:     &blobLayer{content: btfData, mediaType: MediaTypeBTF},
			MediaType: types.MediaType(MediaTypeBTF),
		})
		identityComponents["btf"] = btfData
		componentNames = append(componentNames, "btf")
	}

	// modules (optional, tar directory, not an identity component)
	if b.ModulesPath != "" {
		modData, err := TarDirectory(b.ModulesPath)
		if err != nil {
			return nil, fmt.Errorf("taring modules: %w", err)
		}
		addenda = append(addenda, mutate.Addendum{
			Layer:     &blobLayer{content: modData, mediaType: MediaTypeModules},
			MediaType: types.MediaType(MediaTypeModules),
		})
		componentNames = append(componentNames, "modules")
	}

	// firmware (optional, tar directory, not an identity component)
	if b.FirmwarePath != "" {
		fwData, err := TarDirectory(b.FirmwarePath)
		if err != nil {
			return nil, fmt.Errorf("taring firmware: %w", err)
		}
		addenda = append(addenda, mutate.Addendum{
			Layer:     &blobLayer{content: fwData, mediaType: MediaTypeFirmware},
			MediaType: types.MediaType(MediaTypeFirmware),
		})
		componentNames = append(componentNames, "firmware")
	}

	img, err := mutate.Append(empty.Image, addenda...)
	if err != nil {
		return nil, fmt.Errorf("appending layers: %w", err)
	}

	kbiID := bundle.ComputeKBIID(identityComponents)

	annotations := map[string]string{
		AnnotationKBIID:      kbiID,
		AnnotationKver:       b.Kver,
		AnnotationArch:       b.Arch,
		AnnotationComponents: strings.Join(componentNames, ","),
	}

	annotated, ok := mutate.Annotations(img, annotations).(v1.Image)
	if !ok {
		return nil, fmt.Errorf("unexpected type from mutate.Annotations")
	}
	return annotated, nil
}

// NewBlobLayer creates a blobLayer with the given content and media type.
func NewBlobLayer(content []byte, mediaType string) *blobLayer {
	return &blobLayer{content: content, mediaType: mediaType}
}

// TarDirectory creates a tar archive of all files in a directory.
// Symlinks are skipped — kernel module symlinks (build, source) are not
// preserved. This avoids following symlinks outside the directory tree
// and potential infinite loops from circular symlinks.
func TarDirectory(dir string) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip symlinks to avoid following links outside the directory
		// tree or entering infinite loops with circular symlinks.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
