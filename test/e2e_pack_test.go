package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"
	"github.com/multikernel/kbi/pkg/oci"
	"github.com/multikernel/kbi/pkg/pack"
)

func TestE2E_ModulePack_Bound(t *testing.T) {
	srcDir := t.TempDir()
	storeDir := t.TempDir()

	// Build a KBI image first
	vmlinuz := filepath.Join(srcDir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}
	btf := filepath.Join(srcDir, "btf")
	if err := os.WriteFile(btf, []byte("fake-btf"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		BTFPath:     btf,
		Kver:        "6.8.0",
		Arch:        "amd64",
	}
	kbiImg, err := oci.BuildImage(b)
	if err != nil {
		t.Fatalf("build kbi: %v", err)
	}

	store := oci.NewStore(storeDir)
	kbiRef := "test.io/kernel:6.8.0"
	if err := store.Save(kbiRef, kbiImg); err != nil {
		t.Fatalf("save kbi: %v", err)
	}

	kbiManifest, _ := kbiImg.Manifest()
	kbiID := kbiManifest.Annotations[oci.AnnotationKBIID]

	// Create modules with matching vermagic
	modDir := filepath.Join(srcDir, "mods")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	koContent := createFakeKO("6.8.0 SMP preempt")
	if err := os.WriteFile(filepath.Join(modDir, "mydriver.ko"), koContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Build modulepack bound to the KBI
	p := &pack.Pack{
		Type:       pack.PackTypeModule,
		SourcePath: modDir,
		ForKBIID:   kbiID,
		ForKver:    "6.8.0",
		Arch:       "amd64",
		Tag:        "test.io/mydriver:1.0",
	}

	packImg, err := pack.BuildPack(p)
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}

	if err := store.Save(p.Tag, packImg); err != nil {
		t.Fatalf("save pack: %v", err)
	}

	// Load and verify
	loaded, err := store.Load(p.Tag)
	if err != nil {
		t.Fatalf("load pack: %v", err)
	}

	manifest, _ := loaded.Manifest()
	if manifest.Annotations[pack.AnnotationPackType] != "modulepack" {
		t.Fatalf("wrong type: %s", manifest.Annotations[pack.AnnotationPackType])
	}
	if manifest.Annotations[pack.AnnotationPackForKBIID] != kbiID {
		t.Fatalf("wrong for_kbi_id: %s", manifest.Annotations[pack.AnnotationPackForKBIID])
	}
	if manifest.Annotations[pack.AnnotationPackForKver] != "6.8.0" {
		t.Fatalf("wrong for_kver: %s", manifest.Annotations[pack.AnnotationPackForKver])
	}
	if !strings.Contains(manifest.Annotations[pack.AnnotationPackContents], "mydriver.ko") {
		t.Fatalf("missing mydriver.ko in contents: %s", manifest.Annotations[pack.AnnotationPackContents])
	}
}

func TestE2E_ModulePack_VermagicMismatch(t *testing.T) {
	modDir := t.TempDir()
	koContent := createFakeKO("6.7.0 SMP preempt")
	if err := os.WriteFile(filepath.Join(modDir, "bad.ko"), koContent, 0644); err != nil {
		t.Fatal(err)
	}

	p := &pack.Pack{
		Type:       pack.PackTypeModule,
		SourcePath: modDir,
		ForKBIID:   "kbi:sha256:abc",
		ForKver:    "6.8.0",
		Arch:       "amd64",
		Tag:        "test.io/bad:1.0",
	}

	_, err := pack.BuildPack(p)
	if err == nil {
		t.Fatal("expected vermagic mismatch error")
	}
	if !strings.Contains(err.Error(), "vermagic") {
		t.Fatalf("expected vermagic error, got: %v", err)
	}
}

func TestE2E_BPFPack(t *testing.T) {
	bpfDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(bpfDir, "trace.o"), []byte("fake-bpf"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bpfDir, "filter.o"), []byte("fake-bpf2"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &pack.Pack{
		Type:       pack.PackTypeBPF,
		SourcePath: bpfDir,
		ForKBIID:   "kbi:sha256:abc",
		ForKver:    "6.8.0",
		Arch:       "amd64",
		Tag:        "test.io/mybpf:1.0",
	}

	img, err := pack.BuildPack(p)
	if err != nil {
		t.Fatalf("build bpf pack: %v", err)
	}

	manifest, _ := img.Manifest()
	if manifest.Annotations[pack.AnnotationPackType] != "bpfpack" {
		t.Fatalf("wrong type: %s", manifest.Annotations[pack.AnnotationPackType])
	}
	if manifest.Annotations[pack.AnnotationPackRequires] != "btf" {
		t.Fatalf("expected requires=btf, got: %s", manifest.Annotations[pack.AnnotationPackRequires])
	}
	contents := manifest.Annotations[pack.AnnotationPackContents]
	if !strings.Contains(contents, "trace.o") || !strings.Contains(contents, "filter.o") {
		t.Fatalf("missing programs in contents: %s", contents)
	}
}

// createFakeKO creates a byte slice with embedded vermagic string.
func createFakeKO(vermagic string) []byte {
	prefix := []byte("padding\x00")
	marker := append([]byte("vermagic="), []byte(vermagic)...)
	marker = append(marker, 0x00)
	return append(prefix, marker...)
}
