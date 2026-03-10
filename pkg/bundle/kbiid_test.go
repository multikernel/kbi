package bundle

import (
	"testing"
)

func TestComputeKBIID_VmlinuzOnly(t *testing.T) {
	components := map[string][]byte{
		"vmlinuz": []byte("fake-vmlinuz-content"),
	}
	id := ComputeKBIID(components)
	if id == "" {
		t.Fatal("expected non-empty KBI ID")
	}
	if len(id) != len("kbi:sha256:")+64 {
		t.Fatalf("unexpected KBI ID format: %s", id)
	}
	id2 := ComputeKBIID(components)
	if id != id2 {
		t.Fatalf("expected deterministic ID, got %s and %s", id, id2)
	}
}

func TestComputeKBIID_MultipleComponents(t *testing.T) {
	components := map[string][]byte{
		"vmlinuz": []byte("fake-vmlinuz"),
		"config":  []byte("fake-config"),
		"btf":     []byte("fake-btf"),
	}
	id := ComputeKBIID(components)
	if id == "" {
		t.Fatal("expected non-empty KBI ID")
	}
	id2 := ComputeKBIID(components)
	if id != id2 {
		t.Fatalf("expected deterministic ID regardless of order, got %s and %s", id, id2)
	}
}

func TestComputeKBIID_DifferentInputsDifferentIDs(t *testing.T) {
	id1 := ComputeKBIID(map[string][]byte{"vmlinuz": []byte("kernel-a")})
	id2 := ComputeKBIID(map[string][]byte{"vmlinuz": []byte("kernel-b")})
	if id1 == id2 {
		t.Fatal("different inputs should produce different IDs")
	}
}

func TestComputeKBIID_AddingComponentChangesID(t *testing.T) {
	id1 := ComputeKBIID(map[string][]byte{"vmlinuz": []byte("kernel")})
	id2 := ComputeKBIID(map[string][]byte{"vmlinuz": []byte("kernel"), "config": []byte("cfg")})
	if id1 == id2 {
		t.Fatal("adding a component should change the ID")
	}
}
