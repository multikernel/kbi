package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// ComputeKBIID computes a deterministic KBI ID from identity components.
// Components map: name -> content. Only identity components (vmlinuz, btf, config)
// should be passed. The result is "kbi:sha256:<hex>".
func ComputeKBIID(components map[string][]byte) string {
	var entries []string
	for name, data := range components {
		h := sha256.Sum256(data)
		entries = append(entries, fmt.Sprintf("%s:%s", name, hex.EncodeToString(h[:])))
	}
	sort.Strings(entries)
	combined := strings.Join(entries, "\n")
	final := sha256.Sum256([]byte(combined))
	return "kbi:sha256:" + hex.EncodeToString(final[:])
}
