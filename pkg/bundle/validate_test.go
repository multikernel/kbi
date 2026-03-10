package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_MissingVmlinuz(t *testing.T) {
	b := &Bundle{}
	err := b.Validate()
	if err == nil {
		t.Fatal("expected error for missing vmlinuz")
	}
}

func TestValidate_VmlinuzPathDoesNotExist(t *testing.T) {
	b := &Bundle{VmlinuzPath: "/nonexistent/vmlinuz"}
	err := b.Validate()
	if err == nil {
		t.Fatal("expected error for nonexistent vmlinuz path")
	}
}

func TestValidate_ValidVmlinuzOnly(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-kernel"), 0644); err != nil {
		t.Fatal(err)
	}
	b := &Bundle{VmlinuzPath: vmlinuz}
	if err := b.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidModulesPath(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-kernel"), 0644); err != nil {
		t.Fatal(err)
	}
	b := &Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: "/nonexistent/modules",
	}
	err := b.Validate()
	if err == nil {
		t.Fatal("expected error for nonexistent modules path")
	}
}
