package pack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultBPFManifestName = "kbi-bpf.json"

type BPFManifest struct {
	SchemaVersion int             `json:"schema_version"`
	Programs      []BPFProgram    `json:"programs"`
	Requires      BPFRequirements `json:"requires"`
}

type BPFProgram struct {
	File    string `json:"file"`
	Section string `json:"section"`
	Attach  string `json:"attach"`
	Target  string `json:"target"`
}

type BPFRequirements struct {
	BTF         bool            `json:"btf"`
	Kfuncs      []string        `json:"kfuncs,omitempty"`
	KernelTypes []BPFKernelType `json:"kernel_types,omitempty"`
}

type BPFKernelType struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields,omitempty"`
}

type BPFManifestSummary struct {
	Path     string
	Programs []string
	Kfuncs   []string
	Types    []string
}

func ValidateBPFManifest(sourceDir, manifestPath string) (*BPFManifestSummary, error) {
	if manifestPath == "" {
		manifestPath = filepath.Join(sourceDir, DefaultBPFManifestName)
	} else if !filepath.IsAbs(manifestPath) {
		manifestPath = filepath.Join(sourceDir, manifestPath)
	}
	sourceAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolving BPF source path: %w", err)
	}
	manifestAbs, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolving BPF manifest path: %w", err)
	}
	manifestRel, err := filepath.Rel(sourceAbs, manifestAbs)
	if err != nil {
		return nil, fmt.Errorf("resolving BPF manifest path: %w", err)
	}
	if err := checkRelativePath(manifestRel); err != nil {
		return nil, fmt.Errorf("BPF manifest path: %w", err)
	}

	data, err := os.ReadFile(manifestAbs)
	if err != nil {
		return nil, fmt.Errorf("reading BPF manifest %s: %w", manifestPath, err)
	}

	var manifest BPFManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing BPF manifest %s: %w", manifestPath, err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("BPF manifest %s: schema_version must be 1", manifestPath)
	}
	if len(manifest.Programs) == 0 {
		return nil, fmt.Errorf("BPF manifest %s: at least one program is required", manifestPath)
	}
	if !manifest.Requires.BTF {
		return nil, fmt.Errorf("BPF manifest %s: requires.btf must be true", manifestPath)
	}

	summary := &BPFManifestSummary{Path: filepath.ToSlash(filepath.Clean(manifestRel))}
	for i, program := range manifest.Programs {
		if strings.TrimSpace(program.File) == "" {
			return nil, fmt.Errorf("BPF manifest %s: programs[%d].file is required", manifestPath, i)
		}
		if strings.TrimSpace(program.Section) == "" {
			return nil, fmt.Errorf("BPF manifest %s: programs[%d].section is required", manifestPath, i)
		}
		if strings.TrimSpace(program.Attach) == "" {
			return nil, fmt.Errorf("BPF manifest %s: programs[%d].attach is required", manifestPath, i)
		}
		if strings.TrimSpace(program.Target) == "" {
			return nil, fmt.Errorf("BPF manifest %s: programs[%d].target is required", manifestPath, i)
		}
		if err := checkManifestFile(sourceDir, program.File); err != nil {
			return nil, fmt.Errorf("BPF manifest %s: programs[%d].file: %w", manifestPath, i, err)
		}
		summary.Programs = append(summary.Programs, program.File+":"+program.Section)
	}

	for i, kfunc := range manifest.Requires.Kfuncs {
		kfunc = strings.TrimSpace(kfunc)
		if kfunc == "" {
			return nil, fmt.Errorf("BPF manifest %s: requires.kfuncs[%d] is empty", manifestPath, i)
		}
		summary.Kfuncs = append(summary.Kfuncs, kfunc)
	}

	for i, typ := range manifest.Requires.KernelTypes {
		name := strings.TrimSpace(typ.Name)
		if name == "" {
			return nil, fmt.Errorf("BPF manifest %s: requires.kernel_types[%d].name is required", manifestPath, i)
		}
		for j, field := range typ.Fields {
			if strings.TrimSpace(field) == "" {
				return nil, fmt.Errorf("BPF manifest %s: requires.kernel_types[%d].fields[%d] is empty", manifestPath, i, j)
			}
		}
		summary.Types = append(summary.Types, summarizeKernelType(typ))
	}

	return summary, nil
}

func checkManifestFile(sourceDir, name string) error {
	cleanName := filepath.Clean(name)
	if err := checkRelativePath(cleanName); err != nil {
		return fmt.Errorf("path %q escapes BPF source directory", name)
	}
	info, err := os.Stat(filepath.Join(sourceDir, cleanName))
	if err != nil {
		return fmt.Errorf("path %q does not exist: %w", name, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path %q is a directory, expected an object file", name)
	}
	return nil
}

func checkRelativePath(name string) error {
	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == ".." || filepath.IsAbs(cleanName) ||
		strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path %q escapes source directory", name)
	}
	return nil
}

func summarizeKernelType(typ BPFKernelType) string {
	name := strings.TrimSpace(typ.Name)
	if len(typ.Fields) == 0 {
		return name
	}
	fields := make([]string, 0, len(typ.Fields))
	for _, field := range typ.Fields {
		fields = append(fields, strings.TrimSpace(field))
	}
	return name + ":" + strings.Join(fields, "|")
}
