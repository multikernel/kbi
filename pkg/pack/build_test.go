package pack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestBuildPack_ModulePack_RequiresKBIID(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "modules")
	os.MkdirAll(modDir, 0755)
	os.WriteFile(filepath.Join(modDir, "test.ko"), createFakeKO("6.8.0 SMP"), 0644)

	p := &Pack{
		Type:       PackTypeModule,
		SourcePath: modDir,
		Arch:       "amd64",
		Tag:        "test.io/mydriver:1.0",
	}

	_, err := BuildPack(p)
	if err == nil {
		t.Fatal("expected error when for_kbi_id is missing")
	}
}

func TestBuildPack_ModulePack_Bound(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "modules")
	os.MkdirAll(modDir, 0755)
	os.WriteFile(filepath.Join(modDir, "test.ko"), createFakeKO("6.8.0 SMP"), 0644)

	p := &Pack{
		Type:       PackTypeModule,
		SourcePath: modDir,
		ForKBIID:   "kbi:sha256:abc123",
		ForKver:    "6.8.0",
		Arch:       "amd64",
		Tag:        "test.io/mydriver:1.0",
	}

	img, err := BuildPack(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, _ := img.Manifest()
	if manifest.Annotations[AnnotationPackForKBIID] != "kbi:sha256:abc123" {
		t.Fatalf("wrong for_kbi_id: %s", manifest.Annotations[AnnotationPackForKBIID])
	}
	if manifest.Annotations[AnnotationPackForKver] != "6.8.0" {
		t.Fatalf("wrong for_kver: %s", manifest.Annotations[AnnotationPackForKver])
	}
}

func TestBuildPack_BPFPack(t *testing.T) {
	dir := t.TempDir()
	bpfDir := filepath.Join(dir, "bpf")
	os.MkdirAll(bpfDir, 0755)
	os.WriteFile(filepath.Join(bpfDir, "trace.o"), []byte("fake-bpf"), 0644)
	writeBPFManifest(t, bpfDir, "trace.o")

	p := &Pack{
		Type:       PackTypeBPF,
		SourcePath: bpfDir,
		ForKBIID:   "kbi:sha256:abc123",
		Arch:       "amd64",
		Tag:        "test.io/mybpf:1.0",
	}

	img, err := BuildPack(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, _ := img.Manifest()
	if manifest.Layers[0].MediaType != types.MediaType(MediaTypeBPFPack) {
		t.Fatalf("wrong media type: %s", manifest.Layers[0].MediaType)
	}
	if manifest.Annotations[AnnotationPackType] != string(PackTypeBPF) {
		t.Fatalf("wrong pack type: %s", manifest.Annotations[AnnotationPackType])
	}
	if manifest.Annotations[AnnotationPackForKBIID] != "kbi:sha256:abc123" {
		t.Fatalf("wrong for_kbi_id: %s", manifest.Annotations[AnnotationPackForKBIID])
	}
	if manifest.Annotations[AnnotationPackRequires] != "btf" {
		t.Fatalf("expected requires=btf: %s", manifest.Annotations[AnnotationPackRequires])
	}
	if manifest.Annotations[AnnotationBPFManifest] != DefaultBPFManifestName {
		t.Fatalf("expected BPF manifest annotation: %s", manifest.Annotations[AnnotationBPFManifest])
	}
	if manifest.Annotations[AnnotationBPFPrograms] != "trace.o:fentry/do_sys_openat2" {
		t.Fatalf("wrong BPF programs annotation: %s", manifest.Annotations[AnnotationBPFPrograms])
	}
	if manifest.Annotations[AnnotationBPFKfuncs] != "bpf_task_acquire" {
		t.Fatalf("wrong BPF kfuncs annotation: %s", manifest.Annotations[AnnotationBPFKfuncs])
	}
	if manifest.Annotations[AnnotationBPFTypes] != "task_struct:pid|comm" {
		t.Fatalf("wrong BPF types annotation: %s", manifest.Annotations[AnnotationBPFTypes])
	}
}

func TestBuildPack_BPFPackRequiresManifest(t *testing.T) {
	dir := t.TempDir()
	bpfDir := filepath.Join(dir, "bpf")
	os.MkdirAll(bpfDir, 0755)
	os.WriteFile(filepath.Join(bpfDir, "trace.o"), []byte("fake-bpf"), 0644)

	p := &Pack{
		Type:       PackTypeBPF,
		SourcePath: bpfDir,
		ForKBIID:   "kbi:sha256:abc123",
		Arch:       "amd64",
		Tag:        "test.io/mybpf:1.0",
	}

	if _, err := BuildPack(p); err == nil {
		t.Fatal("expected error when BPF manifest is missing")
	}
}

func TestBuildPack_EmptySourceDir(t *testing.T) {
	dir := t.TempDir()
	p := &Pack{
		Type:       PackTypeModule,
		SourcePath: dir,
		ForKBIID:   "kbi:sha256:abc123",
		Arch:       "amd64",
		Tag:        "test.io/empty:1.0",
	}
	_, err := BuildPack(p)
	if err == nil {
		t.Fatal("expected error for empty source dir")
	}
}

func writeBPFManifest(t *testing.T, dir, file string) {
	t.Helper()
	manifest := `{
  "schema_version": 1,
  "programs": [
    {
      "file": "` + file + `",
      "section": "fentry/do_sys_openat2",
      "attach": "fentry",
      "target": "do_sys_openat2"
    }
  ],
  "requires": {
    "btf": true,
    "kfuncs": ["bpf_task_acquire"],
    "kernel_types": [
      {"name": "task_struct", "fields": ["pid", "comm"]}
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, DefaultBPFManifestName), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
}
