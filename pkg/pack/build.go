package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/multikernel/kbi/pkg/oci"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func BuildPack(p *Pack) (v1.Image, error) {
	if p.SourcePath == "" {
		return nil, fmt.Errorf("source path is required")
	}
	if _, err := os.Stat(p.SourcePath); err != nil {
		return nil, fmt.Errorf("source path %s does not exist: %w", p.SourcePath, err)
	}

	// Validate modules if bound
	if p.ForKver != "" && p.Type == PackTypeModule {
		if errs := ValidateModules(p.SourcePath, p.ForKver); len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			return nil, fmt.Errorf("module validation failed:\n  %s", strings.Join(msgs, "\n  "))
		}
	}

	var mediaType string
	switch p.Type {
	case PackTypeModule:
		mediaType = MediaTypeModulePack
	case PackTypeBPF:
		mediaType = MediaTypeBPFPack
	default:
		return nil, fmt.Errorf("unknown pack type: %s", p.Type)
	}

	contents := listFiles(p.SourcePath)
	if len(contents) == 0 {
		return nil, fmt.Errorf("no files found in %s", p.SourcePath)
	}

	tarData, err := oci.TarDirectory(p.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("creating tar: %w", err)
	}

	layer := oci.NewBlobLayer(tarData, mediaType)

	img, err := mutate.Append(empty.Image, mutate.Addendum{
		Layer:     layer,
		MediaType: types.MediaType(mediaType),
	})
	if err != nil {
		return nil, fmt.Errorf("appending layer: %w", err)
	}

	annotations := map[string]string{
		AnnotationPackType:     string(p.Type),
		oci.AnnotationArch:    p.Arch,
		AnnotationPackContents: strings.Join(contents, ","),
	}
	if p.ForKBIID != "" {
		annotations[AnnotationPackForKBIID] = p.ForKBIID
	}
	if p.ForKver != "" {
		annotations[AnnotationPackForKver] = p.ForKver
	}
	if p.Type == PackTypeBPF {
		annotations[AnnotationPackRequires] = "btf"
	}

	annotated, ok := mutate.Annotations(img, annotations).(v1.Image)
	if !ok {
		return nil, fmt.Errorf("unexpected type from mutate.Annotations")
	}
	return annotated, nil
}

func listFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, info.Name())
		return nil
	})
	return files
}
