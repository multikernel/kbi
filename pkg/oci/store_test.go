package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")

	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}
	b := &bundle.Bundle{VmlinuzPath: vmlinuz, Kver: "6.8.0", Arch: "amd64"}

	img, err := BuildImage(b)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	store := NewStore(storeDir)

	ref := "registry.io/org/kernel:latest"
	if err := store.Save(ref, img); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	origManifest, _ := img.Manifest()
	loadedManifest, _ := loaded.Manifest()
	if len(origManifest.Layers) != len(loadedManifest.Layers) {
		t.Fatalf("layer count mismatch: %d vs %d", len(origManifest.Layers), len(loadedManifest.Layers))
	}
}

func TestLoad_NotFound(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Load("registry.io/org/kernel:nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent image")
	}
}

func TestSaveReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")

	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	first, err := BuildImage(&bundle.Bundle{VmlinuzPath: vmlinuz, Kver: "6.8.0", Arch: "amd64"})
	if err != nil {
		t.Fatalf("build first: %v", err)
	}

	store := NewStore(storeDir)
	ref := "registry.io/org/kernel:latest"
	if err := store.Save(ref, first); err != nil {
		t.Fatalf("save first: %v", err)
	}

	if err := os.WriteFile(vmlinuz, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	second, err := BuildImage(&bundle.Bundle{VmlinuzPath: vmlinuz, Kver: "6.8.0", Arch: "amd64"})
	if err != nil {
		t.Fatalf("build second: %v", err)
	}
	if err := store.Save(ref, second); err != nil {
		t.Fatalf("save second: %v", err)
	}

	loaded, err := store.Load(ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	wantDigest, _ := second.Digest()
	gotDigest, _ := loaded.Digest()
	if wantDigest != gotDigest {
		t.Fatalf("Load returned stale image: got %s, want %s", gotDigest, wantDigest)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")

	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}
	b := &bundle.Bundle{VmlinuzPath: vmlinuz, Kver: "6.8.0", Arch: "amd64"}
	img, _ := BuildImage(b)

	store := NewStore(storeDir)
	ref := "registry.io/org/kernel:v1"

	if store.Exists(ref) {
		t.Fatal("should not exist before save")
	}
	if err := store.Save(ref, img); err != nil {
		t.Fatal(err)
	}
	if !store.Exists(ref) {
		t.Fatal("should exist after save")
	}
}
