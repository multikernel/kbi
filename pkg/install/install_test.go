package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"
	"github.com/multikernel/kbi/pkg/oci"
)

func TestInstall_VmlinuzOnly(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	vmlinuz := filepath.Join(srcDir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{VmlinuzPath: vmlinuz, Kver: "6.8.0", Arch: "amd64"}
	img, err := oci.BuildImage(b)
	if err != nil {
		t.Fatal(err)
	}

	if err := Install(img, "6.8.0", destDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	installed := filepath.Join(destDir, "boot", "vmlinuz-6.8.0")
	data, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("vmlinuz not installed: %v", err)
	}
	if string(data) != "fake-vmlinuz" {
		t.Fatalf("unexpected content: %s", data)
	}
}

func TestInstall_WithModules(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	vmlinuz := filepath.Join(srcDir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(srcDir, "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	moduleContent := fakeKO("6.8.0 SMP preempt")
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), moduleContent, 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: filepath.Join(srcDir, "modules"),
		Kver:        "6.8.0",
		Arch:        "amd64",
	}
	img, err := oci.BuildImage(b)
	if err != nil {
		t.Fatal(err)
	}

	if err := Install(img, "6.8.0", destDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	modFile := filepath.Join(destDir, "lib", "modules", "6.8.0", "test.ko")
	if _, err := os.Stat(modFile); err != nil {
		t.Fatalf("module not installed: %v", err)
	}
}

func TestInstall_WithModulesVersionDir(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	vmlinuz := filepath.Join(srcDir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(srcDir, "lib", "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	moduleContent := fakeKO("6.8.0 SMP preempt")
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), moduleContent, 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: modDir,
		Kver:        "6.8.0",
		Arch:        "amd64",
	}
	img, err := oci.BuildImage(b)
	if err != nil {
		t.Fatal(err)
	}

	if err := Install(img, "6.8.0", destDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	modFile := filepath.Join(destDir, "lib", "modules", "6.8.0", "test.ko")
	data, err := os.ReadFile(modFile)
	if err != nil {
		t.Fatalf("module not installed under kver directory: %v", err)
	}
	if string(data) != string(moduleContent) {
		t.Fatalf("unexpected module content: %s", data)
	}
}

func TestInstall_DestDoesNotExist(t *testing.T) {
	srcDir := t.TempDir()
	vmlinuz := filepath.Join(srcDir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}
	b := &bundle.Bundle{VmlinuzPath: vmlinuz, Kver: "6.8.0", Arch: "amd64"}
	img, _ := oci.BuildImage(b)

	err := Install(img, "6.8.0", "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent dest")
	}
}

func fakeKO(vermagic string) []byte {
	prefix := []byte("some padding bytes here\x00")
	marker := append([]byte("vermagic="), []byte(vermagic)...)
	marker = append(marker, 0x00)
	suffix := []byte("\x00more padding")
	return append(append(prefix, marker...), suffix...)
}
