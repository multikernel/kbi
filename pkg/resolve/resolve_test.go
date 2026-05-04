package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"
	"github.com/multikernel/kbi/pkg/oci"
	"github.com/multikernel/kbi/pkg/pack"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func TestResolve_AcceptsCompatiblePack(t *testing.T) {
	kbiImg := buildTestKBI(t, true, "amd64")
	kbiID := kbiID(t, kbiImg)
	packImg := buildTestPack(t, pack.PackTypeModule, kbiID, "amd64")

	resolved, err := Resolve("test.io/kernel:6.8.0", kbiImg, []PackInput{
		{Ref: "test.io/driver:1.0", Image: packImg},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.KBIID != kbiID {
		t.Fatalf("wrong KBI ID: %s", resolved.KBIID)
	}
	if len(resolved.Packs) != 1 {
		t.Fatalf("expected 1 pack, got %d", len(resolved.Packs))
	}
}

func TestResolve_RejectsWrongKBIID(t *testing.T) {
	kbiImg := buildTestKBI(t, true, "amd64")
	packImg := buildTestPack(t, pack.PackTypeModule, "kbi:sha256:wrong", "amd64")

	_, err := Resolve("test.io/kernel:6.8.0", kbiImg, []PackInput{
		{Ref: "test.io/driver:1.0", Image: packImg},
	})
	if err == nil {
		t.Fatal("expected KBI ID mismatch error")
	}
}

func TestResolve_RejectsWrongArch(t *testing.T) {
	kbiImg := buildTestKBI(t, true, "amd64")
	kbiID := kbiID(t, kbiImg)
	packImg := buildTestPack(t, pack.PackTypeModule, kbiID, "arm64")

	_, err := Resolve("test.io/kernel:6.8.0", kbiImg, []PackInput{
		{Ref: "test.io/driver:1.0", Image: packImg},
	})
	if err == nil {
		t.Fatal("expected arch mismatch error")
	}
}

func TestResolve_RejectsBPFWithoutBTF(t *testing.T) {
	kbiImg := buildTestKBI(t, false, "amd64")
	kbiID := kbiID(t, kbiImg)
	packImg := buildTestPack(t, pack.PackTypeBPF, kbiID, "amd64")

	_, err := Resolve("test.io/kernel:6.8.0", kbiImg, []PackInput{
		{Ref: "test.io/bpf:1.0", Image: packImg},
	})
	if err == nil {
		t.Fatal("expected BTF requirement error")
	}
}

func buildTestKBI(t *testing.T, withBTF bool, arch string) v1.Image {
	t.Helper()

	dir := t.TempDir()
	vmlinuz := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(vmlinuz, []byte("fake-vmlinuz"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &bundle.Bundle{
		VmlinuzPath: vmlinuz,
		Kver:        "6.8.0",
		Arch:        arch,
	}
	if withBTF {
		btf := filepath.Join(dir, "btf")
		if err := os.WriteFile(btf, []byte("fake-btf"), 0644); err != nil {
			t.Fatal(err)
		}
		b.BTFPath = btf
	}

	img, err := oci.BuildImage(b)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func buildTestPack(t *testing.T, packType pack.PackType, forKBIID, arch string) v1.Image {
	t.Helper()

	dir := t.TempDir()
	switch packType {
	case pack.PackTypeModule:
		if err := os.WriteFile(filepath.Join(dir, "artifact.ko"), fakeKO("6.8.0 SMP"), 0644); err != nil {
			t.Fatal(err)
		}
	case pack.PackTypeBPF:
		if err := os.WriteFile(filepath.Join(dir, "artifact.o"), []byte("fake-bpf"), 0644); err != nil {
			t.Fatal(err)
		}
		writeBPFManifest(t, dir)
	}

	p := &pack.Pack{
		Type:       packType,
		SourcePath: dir,
		ForKBIID:   forKBIID,
		ForKver:    "6.8.0",
		Arch:       arch,
		Tag:        "test.io/pack:1.0",
	}
	img, err := pack.BuildPack(p)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func fakeKO(vermagic string) []byte {
	prefix := []byte("padding\x00")
	marker := append([]byte("vermagic="), []byte(vermagic)...)
	marker = append(marker, 0x00)
	return append(prefix, marker...)
}

func writeBPFManifest(t *testing.T, dir string) {
	t.Helper()
	manifest := `{
  "schema_version": 1,
  "programs": [
    {
      "file": "artifact.o",
      "section": "fentry/do_sys_openat2",
      "attach": "fentry",
      "target": "do_sys_openat2"
    }
  ],
  "requires": {
    "btf": true,
    "kernel_types": [
      {"name": "task_struct", "fields": ["pid", "comm"]}
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, pack.DefaultBPFManifestName), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
}

func kbiID(t *testing.T, img v1.Image) string {
	t.Helper()
	manifest, err := img.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	return manifest.Annotations[oci.AnnotationKBIID]
}
