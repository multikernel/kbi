package bundle

import (
	"fmt"
	"os"
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
