package bundle

import (
	"fmt"
	"os"
	"strings"

	"github.com/multikernel/kbi/pkg/kmod"
)

func (b *Bundle) Validate() error {
	if b.VmlinuzPath == "" {
		return fmt.Errorf("vmlinuz path is required")
	}
	if err := checkFileExists(b.VmlinuzPath); err != nil {
		return fmt.Errorf("vmlinuz: %w", err)
	}
	optionalFiles := map[string]string{
		"initrd": b.InitrdPath,
		"config": b.ConfigPath,
		"btf":    b.BTFPath,
	}
	for name, path := range optionalFiles {
		if path != "" {
			if err := checkFileExists(path); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	optionalDirs := map[string]string{
		"modules":  b.ModulesPath,
		"firmware": b.FirmwarePath,
	}
	for name, path := range optionalDirs {
		if path != "" {
			if err := checkDirExists(path); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	if b.ModulesPath != "" {
		if b.Kver == "" {
			return fmt.Errorf("modules: kver is required to validate modules")
		}
		if errs := kmod.ValidateModulesForKver(b.ModulesPath, b.Kver); len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, err := range errs {
				msgs[i] = err.Error()
			}
			return fmt.Errorf("modules validation failed:\n  %s", strings.Join(msgs, "\n  "))
		}
	}
	return nil
}

func checkFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path %s does not exist: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path %s is a directory, expected a file", path)
	}
	return nil
}

func checkDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path %s does not exist: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %s is not a directory", path)
	}
	return nil
}
