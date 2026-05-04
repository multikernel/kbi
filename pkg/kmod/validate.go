package kmod

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// IsModuleFile returns true for .ko, .ko.zst, .ko.gz, .ko.xz suffixes.
func IsModuleFile(name string) bool {
	for _, ext := range []string{".ko", ".ko.zst", ".ko.gz", ".ko.xz"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// ExtractVermagic reads the module at path and returns its vermagic string.
func ExtractVermagic(path string) (string, error) {
	data, err := readModuleBytes(path)
	if err != nil {
		return "", fmt.Errorf("reading module bytes: %w", err)
	}
	v := scanVermagic(data)
	if v == "" {
		return "", fmt.Errorf("vermagic not found in %s", path)
	}
	return v, nil
}

// ValidateModules walks dir, finds kernel module files, extracts their
// vermagic, and checks that the kernel version prefix matches targetKver.
// Returns an error if no modules are found.
func ValidateModules(dir string, targetKver string) []error {
	var errs []error
	found := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !IsModuleFile(info.Name()) {
			return nil
		}
		found++

		vermagic, err := ExtractVermagic(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			return nil
		}

		kver := strings.SplitN(vermagic, " ", 2)[0]
		if kver != targetKver {
			errs = append(errs, fmt.Errorf("%s: vermagic kernel version %q does not match target %q (full vermagic: %q)",
				path, kver, targetKver, vermagic))
		}
		return nil
	})
	if err != nil {
		errs = append(errs, fmt.Errorf("walking directory %s: %w", dir, err))
	}

	if found == 0 {
		errs = append(errs, fmt.Errorf("no kernel module files found in %s", dir))
	}

	return errs
}

// ValidateModulesForKver validates the module tree for targetKver. The input
// may be either /lib/modules/<kver>, /lib/modules, or a staging directory.
func ValidateModulesForKver(dir, targetKver string) []error {
	if targetKver == "" {
		return []error{fmt.Errorf("kernel version is required to validate modules")}
	}
	if !validKverPathComponent(targetKver) {
		return []error{fmt.Errorf("invalid kernel version %q for module validation", targetKver)}
	}
	return ValidateModules(ModuleDirForKver(dir, targetKver), targetKver)
}

// ModuleDirForKver returns the directory that should be scanned for targetKver.
func ModuleDirForKver(dir, targetKver string) string {
	cleanDir := filepath.Clean(dir)
	cleanKver := filepath.Clean(targetKver)
	if filepath.Base(cleanDir) == cleanKver {
		return cleanDir
	}

	kverDir := filepath.Join(cleanDir, cleanKver)
	if info, err := os.Stat(kverDir); err == nil && info.IsDir() {
		return kverDir
	}

	return cleanDir
}

func validKverPathComponent(kver string) bool {
	cleanKver := filepath.Clean(kver)
	return cleanKver != "." &&
		cleanKver != ".." &&
		!filepath.IsAbs(cleanKver) &&
		!strings.HasPrefix(cleanKver, ".."+string(os.PathSeparator))
}

func readModuleBytes(path string) ([]byte, error) {
	switch {
	case strings.HasSuffix(path, ".ko.zst"):
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		dec, err := zstd.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("creating zstd reader for %s: %w", path, err)
		}
		defer dec.Close()
		return io.ReadAll(dec)

	case strings.HasSuffix(path, ".ko.gz"):
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}
		defer f.Close()
		gr, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader for %s: %w", path, err)
		}
		defer gr.Close()
		return io.ReadAll(gr)

	case strings.HasSuffix(path, ".ko.xz"):
		return nil, fmt.Errorf("xz decompression not supported")

	default:
		return os.ReadFile(path)
	}
}

func scanVermagic(data []byte) string {
	marker := []byte("vermagic=")
	idx := bytes.Index(data, marker)
	if idx < 0 {
		return ""
	}
	start := idx + len(marker)
	rest := data[start:]
	end := bytes.IndexByte(rest, 0x00)
	if end < 0 {
		return string(rest)
	}
	return string(rest[:end])
}
