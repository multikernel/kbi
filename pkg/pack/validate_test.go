package pack

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExtractVermagic(t *testing.T) {
	koPath := findTestModule(t)
	if koPath == "" {
		t.Skip("no kernel module found for testing")
	}
	vermagic, err := extractVermagic(koPath)
	if err != nil {
		t.Fatalf("extractVermagic: %v", err)
	}
	if vermagic == "" {
		t.Fatal("expected non-empty vermagic")
	}
	t.Logf("vermagic: %s", vermagic)
}

func TestValidateModules_Matching(t *testing.T) {
	dir := t.TempDir()
	koContent := createFakeKO("6.8.0 SMP preempt")
	if err := os.WriteFile(filepath.Join(dir, "test.ko"), koContent, 0644); err != nil {
		t.Fatal(err)
	}
	errs := ValidateModules(dir, "6.8.0")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateModules_Mismatch(t *testing.T) {
	dir := t.TempDir()
	koContent := createFakeKO("6.7.0 SMP preempt")
	if err := os.WriteFile(filepath.Join(dir, "test.ko"), koContent, 0644); err != nil {
		t.Fatal(err)
	}
	errs := ValidateModules(dir, "6.8.0")
	if len(errs) == 0 {
		t.Fatal("expected vermagic mismatch error")
	}
}

func TestValidateModules_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	errs := ValidateModules(dir, "6.8.0")
	if len(errs) == 0 {
		t.Fatal("expected error for empty directory")
	}
}

func TestValidateBPF_WithBTF(t *testing.T) {
	err := ValidateBPF("vmlinuz,btf,config")
	if err != nil {
		t.Fatalf("expected no error when BTF present: %v", err)
	}
}

func TestValidateBPF_WithoutBTF(t *testing.T) {
	err := ValidateBPF("vmlinuz,config")
	if err == nil {
		t.Fatal("expected error when BTF missing")
	}
}

func createFakeKO(vermagic string) []byte {
	prefix := []byte("some padding bytes here\x00")
	marker := append([]byte("vermagic="), []byte(vermagic)...)
	marker = append(marker, 0x00)
	suffix := []byte("\x00more padding")
	return append(append(prefix, marker...), suffix...)
}

func findTestModule(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return ""
	}
	kver := string(out[:len(out)-1])
	modDir := "/lib/modules/" + kver + "/kernel"
	var found string
	filepath.Walk(modDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if found == "" && !info.IsDir() && (filepath.Ext(path) == ".ko" || filepath.Ext(path) == ".zst") {
			found = path
		}
		return nil
	})
	return found
}
