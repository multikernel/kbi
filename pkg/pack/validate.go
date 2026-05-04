package pack

import (
	"fmt"
	"strings"

	"github.com/multikernel/kbi/pkg/kmod"
)

// ValidateModules walks dir, finds kernel module files, extracts their
// vermagic, and checks that the kernel version prefix matches targetKver.
func ValidateModules(dir string, targetKver string) []error {
	return kmod.ValidateModules(dir, targetKver)
}

func extractVermagic(path string) (string, error) {
	return kmod.ExtractVermagic(path)
}

// ValidateBPF checks that the comma-separated components string contains "btf".
func ValidateBPF(components string) error {
	for _, c := range strings.Split(components, ",") {
		if strings.TrimSpace(c) == "btf" {
			return nil
		}
	}
	return fmt.Errorf("BTF not found in components %q; BPF programs require BTF for CO-RE", components)
}
