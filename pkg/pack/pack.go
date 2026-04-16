package pack

// Pack represents a KBI add-on pack (ModulePack or BPF Pack).
type Pack struct {
	Type       PackType
	SourcePath string // modules dir or bpf dir
	ForRef     string // target KBI image reference (optional)
	ForKBIID   string // target KBI ID (optional, set directly or resolved from ForRef)
	ForKver    string // target kernel version (resolved from ForRef or set manually)
	Arch       string // architecture (resolved from ForRef or set manually)
	Tag        string // output image reference
}
