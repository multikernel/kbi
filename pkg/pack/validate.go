package pack

import (
	"compress/gzip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// isModuleFile returns true for .ko, .ko.zst, .ko.gz, .ko.xz suffixes.
func isModuleFile(name string) bool {
	for _, ext := range []string{".ko", ".ko.zst", ".ko.gz", ".ko.xz"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// readModuleBytes reads and decompresses a kernel module file based on its extension.
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

	default: // plain .ko
		return os.ReadFile(path)
	}
}

// scanVermagic finds the "vermagic=" marker in data and returns the value up to the next null byte.
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

// extractVermagic reads the module at path and returns its vermagic string.
func extractVermagic(path string) (string, error) {
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

// ValidateModules walks dir, finds kernel module files, extracts their vermagic,
// and checks that the kernel version prefix matches targetKver.
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
		if !isModuleFile(info.Name()) {
			return nil
		}
		found++

		vermagic, err := extractVermagic(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			return nil
		}

		// The vermagic string starts with the kernel version followed by a space
		// (e.g. "6.8.0 SMP preempt mod_unload"). Check that the prefix matches.
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

// ValidateBPF checks that the comma-separated components string contains "btf".
func ValidateBPF(components string) error {
	for _, c := range strings.Split(components, ",") {
		if strings.TrimSpace(c) == "btf" {
			return nil
		}
	}
	return fmt.Errorf("BTF not found in components %q; BPF programs require BTF for CO-RE", components)
}
