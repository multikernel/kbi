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

func TestValidate_ModulesMatchingKver(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-kernel"), 0644); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(dir, "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), fakeKO("6.8.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: filepath.Join(dir, "modules"),
		Kver:        "6.8.0",
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ModulesRequireKver(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-kernel"), 0644); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(dir, "modules")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), fakeKO("6.8.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: modDir,
	}
	if err := b.Validate(); err == nil {
		t.Fatal("expected error when modules are provided without kver")
	}
}

func TestValidate_ModulesRejectMismatchedKver(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-kernel"), 0644); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(dir, "modules", "6.8.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "test.ko"), fakeKO("6.7.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: modDir,
		Kver:        "6.8.0",
	}
	if err := b.Validate(); err == nil {
		t.Fatal("expected error for module vermagic mismatch")
	}
}

func TestValidate_ModulesParentDirIgnoresOtherKvers(t *testing.T) {
	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-kernel"), 0644); err != nil {
		t.Fatal(err)
	}
	modulesRoot := filepath.Join(dir, "modules")
	targetDir := filepath.Join(modulesRoot, "6.8.0")
	otherDir := filepath.Join(modulesRoot, "6.7.0")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "good.ko"), fakeKO("6.8.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "old.ko"), fakeKO("6.7.0 SMP preempt"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &Bundle{
		VmlinuzPath: vmlinuz,
		ModulesPath: modulesRoot,
		Kver:        "6.8.0",
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func fakeKO(vermagic string) []byte {
	prefix := []byte("some padding bytes here\x00")
	marker := append([]byte("vermagic="), []byte(vermagic)...)
	marker = append(marker, 0x00)
	suffix := []byte("\x00more padding")
	return append(append(prefix, marker...), suffix...)
}
