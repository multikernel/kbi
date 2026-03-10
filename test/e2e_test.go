package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"
	"github.com/multikernel/kbi/pkg/install"
	"github.com/multikernel/kbi/pkg/oci"
)

func TestE2E_BuildInspectInstall(t *testing.T) {
	srcDir := t.TempDir()
	storeDir := t.TempDir()
	destDir := t.TempDir()

	// Create kernel artifacts
	vmlinuz := filepath.Join(srcDir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz-content"), 0644); err != nil {
		t.Fatal(err)
	}

	initrd := filepath.Join(srcDir, "initrd")
	if err := os.WriteFile(initrd, []byte("fake-initrd-content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := filepath.Join(srcDir, "config")
	if err := os.WriteFile(config, []byte("CONFIG_SMP=y\nCONFIG_MODULES=y\n"), 0644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(srcDir, "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), []byte("fake-module"), 0644); err != nil {
		t.Fatal(err)
	}

	// Build
	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		InitrdPath:  initrd,
		ConfigPath:  config,
		ModulesPath: filepath.Join(srcDir, "modules"),
		Kver:        "6.8.0",
		Arch:        "amd64",
		Tag:         "test.io/mykernel:latest",
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	img, err := oci.BuildImage(b)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// Save to store
	store := oci.NewStore(storeDir)
	if err := store.Save(b.Tag, img); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load and inspect
	loaded, err := store.Load(b.Tag)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	manifest, err := loaded.Manifest()
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}

	if manifest.Annotations[oci.AnnotationKver] != "6.8.0" {
		t.Fatalf("wrong kver: %s", manifest.Annotations[oci.AnnotationKver])
	}
	if manifest.Annotations[oci.AnnotationArch] != "amd64" {
		t.Fatalf("wrong arch: %s", manifest.Annotations[oci.AnnotationArch])
	}
	if manifest.Annotations[oci.AnnotationKBIID] == "" {
		t.Fatal("missing KBI ID")
	}
	components := manifest.Annotations[oci.AnnotationComponents]
	for _, c := range []string{"vmlinuz", "initrd", "config", "modules"} {
		if !strings.Contains(components, c) {
			t.Fatalf("missing component %s in %s", c, components)
		}
	}

	// Install
	if err := install.Install(loaded, "6.8.0", destDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Verify installed files
	checks := map[string]string{
		filepath.Join(destDir, "boot", "vmlinuz-6.8.0"):              "fake-vmlinuz-content",
		filepath.Join(destDir, "boot", "initrd.img-6.8.0"):           "fake-initrd-content",
		filepath.Join(destDir, "boot", "config-6.8.0"):               "CONFIG_SMP=y\nCONFIG_MODULES=y\n",
		filepath.Join(destDir, "lib", "modules", "6.8.0", "test.ko"): "fake-module",
	}
	for path, expected := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("file %s not found: %v", path, err)
		}
		if string(data) != expected {
			t.Fatalf("file %s: expected %q, got %q", path, expected, string(data))
		}
	}
}
