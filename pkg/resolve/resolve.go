package resolve

import (
	"fmt"
	"strings"

	"github.com/multikernel/kbi/pkg/oci"
	"github.com/multikernel/kbi/pkg/pack"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// PackInput pairs a pack reference with its OCI image.
type PackInput struct {
	Ref   string
	Image v1.Image
}

// PackResult describes a pack accepted into the resolved kernel view.
type PackResult struct {
	Ref      string
	Type     string
	ForKBIID string
	ForKver  string
	Arch     string
	Contents string
	Requires string
}

// Result is the resolved and compatibility-checked kernel view.
type Result struct {
	KBIRef     string
	KBIID      string
	Kver       string
	Arch       string
	Components string
	Packs      []PackResult
}

// Resolve enforces pack compatibility against a KBI image and returns a
// resolved view. Signature policy is intentionally out of scope for this first
// resolver pass; compatibility binding is enforced here.
func Resolve(kbiRef string, kbiImg v1.Image, packs []PackInput) (*Result, error) {
	kbiManifest, err := kbiImg.Manifest()
	if err != nil {
		return nil, fmt.Errorf("reading KBI manifest: %w", err)
	}

	kbiAnnotations := kbiManifest.Annotations
	result := &Result{
		KBIRef:     kbiRef,
		KBIID:      kbiAnnotations[oci.AnnotationKBIID],
		Kver:       kbiAnnotations[oci.AnnotationKver],
		Arch:       kbiAnnotations[oci.AnnotationArch],
		Components: kbiAnnotations[oci.AnnotationComponents],
	}

	if result.KBIID == "" {
		return nil, fmt.Errorf("KBI image %s is missing %s annotation", kbiRef, oci.AnnotationKBIID)
	}
	if result.Arch == "" {
		return nil, fmt.Errorf("KBI image %s is missing %s annotation", kbiRef, oci.AnnotationArch)
	}

	for _, input := range packs {
		packManifest, err := input.Image.Manifest()
		if err != nil {
			return nil, fmt.Errorf("reading pack manifest %s: %w", input.Ref, err)
		}

		annotations := packManifest.Annotations
		resolvedPack := PackResult{
			Ref:      input.Ref,
			Type:     annotations[pack.AnnotationPackType],
			ForKBIID: annotations[pack.AnnotationPackForKBIID],
			ForKver:  annotations[pack.AnnotationPackForKver],
			Arch:     annotations[oci.AnnotationArch],
			Contents: annotations[pack.AnnotationPackContents],
			Requires: annotations[pack.AnnotationPackRequires],
		}

		if resolvedPack.Type != string(pack.PackTypeModule) && resolvedPack.Type != string(pack.PackTypeBPF) {
			return nil, fmt.Errorf("pack %s has unknown or missing pack type %q", input.Ref, resolvedPack.Type)
		}
		if resolvedPack.ForKBIID == "" {
			return nil, fmt.Errorf("pack %s is missing %s annotation", input.Ref, pack.AnnotationPackForKBIID)
		}
		if resolvedPack.ForKBIID != result.KBIID {
			return nil, fmt.Errorf("pack %s targets KBI ID %s, not %s", input.Ref, resolvedPack.ForKBIID, result.KBIID)
		}
		if resolvedPack.Arch == "" {
			return nil, fmt.Errorf("pack %s is missing %s annotation", input.Ref, oci.AnnotationArch)
		}
		if resolvedPack.Arch != result.Arch {
			return nil, fmt.Errorf("pack %s targets arch %s, not %s", input.Ref, resolvedPack.Arch, result.Arch)
		}
		if resolvedPack.ForKver != "" && result.Kver != "" && resolvedPack.ForKver != result.Kver {
			return nil, fmt.Errorf("pack %s targets kernel %s, not %s", input.Ref, resolvedPack.ForKver, result.Kver)
		}
		if resolvedPack.Type == string(pack.PackTypeBPF) && !hasComponent(result.Components, "btf") {
			return nil, fmt.Errorf("BPF pack %s requires BTF, but KBI image %s does not include BTF", input.Ref, kbiRef)
		}

		result.Packs = append(result.Packs, resolvedPack)
	}

	return result, nil
}

func hasComponent(components, want string) bool {
	return listContains(components, want)
}

func listContains(list, want string) bool {
	for _, item := range strings.Split(list, ",") {
		if strings.TrimSpace(item) == want {
			return true
		}
	}
	return false
}
