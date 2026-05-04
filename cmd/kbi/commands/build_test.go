package commands

import (
	"testing"

	"github.com/multikernel/kbi/pkg/bundle"
)

func TestValidateBuildMetadataRequiresKver(t *testing.T) {
	err := validateBuildMetadata(&bundle.Bundle{Arch: "amd64"})
	if err == nil {
		t.Fatal("expected error when kver is missing")
	}
}

func TestValidateBuildMetadataRequiresArch(t *testing.T) {
	err := validateBuildMetadata(&bundle.Bundle{Kver: "6.8.0"})
	if err == nil {
		t.Fatal("expected error when arch is missing")
	}
}

func TestValidateBuildMetadataAcceptsKverAndArch(t *testing.T) {
	err := validateBuildMetadata(&bundle.Bundle{Kver: "6.8.0", Arch: "amd64"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
