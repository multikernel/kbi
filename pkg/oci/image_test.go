package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"

	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestBuildImage_VmlinuzOnly(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		Kver:        "6.8.0",
		Arch:        "amd64",
	}

	img, err := BuildImage(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		t.Fatalf("failed to get manifest: %v", err)
	}

	// Should have exactly 1 layer (vmlinuz)
	if len(manifest.Layers) != 1 {
		t.Fatalf("expected 1 layer, got %d", len(manifest.Layers))
	}
	if manifest.Layers[0].MediaType != types.MediaType(MediaTypeVmlinuz) {
		t.Fatalf("expected vmlinuz media type, got %s", manifest.Layers[0].MediaType)
	}

	// Check annotations
	annotations := manifest.Annotations
	if annotations[AnnotationKver] != "6.8.0" {
		t.Fatalf("expected kver 6.8.0, got %s", annotations[AnnotationKver])
	}
	if annotations[AnnotationArch] != "amd64" {
		t.Fatalf("expected arch amd64, got %s", annotations[AnnotationArch])
	}
	if annotations[AnnotationKBIID] == "" {
		t.Fatal("expected non-empty KBI ID annotation")
	}
}

func TestBuildImage_AllComponents(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	files := map[string]string{
		"vmlinuz": "fake-vmlinuz",
		"initrd":  "fake-initrd",
		"config":  "fake-config",
		"btf":     "fake-btf",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create modules directory with a file
	modDir := filepath.Join(dir, "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), fakeKO("6.8.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: filepath.Join(dir, "vmlinuz"),
		InitrdPath:  filepath.Join(dir, "initrd"),
		ConfigPath:  filepath.Join(dir, "config"),
		BTFPath:     filepath.Join(dir, "btf"),
		ModulesPath: filepath.Join(dir, "modules"),
		Kver:        "6.8.0",
		Arch:        "amd64",
	}

	img, err := BuildImage(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		t.Fatalf("failed to get manifest: %v", err)
	}

	// Should have 5 layers: vmlinuz, initrd, config, btf, modules
	if len(manifest.Layers) != 5 {
		t.Fatalf("expected 5 layers, got %d", len(manifest.Layers))
	}
}

func TestBuildImage_RejectsMismatchedModules(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(dir, "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), fakeKO("6.7.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: filepath.Join(dir, "modules"),
		Kver:        "6.8.0",
		Arch:        "amd64",
	}

	if _, err := BuildImage(b); err == nil {
		t.Fatal("expected error for module vermagic mismatch")
	}
}

func fakeKO(vermagic string) []byte {
	prefix := []byte("some padding bytes here\x00")
	marker := append([]byte("vermagic="), []byte(vermagic)...)
	marker = append(marker, 0x00)
	suffix := []byte("\x00more padding")
	return append(append(prefix, marker...), suffix...)
}
