package bundle

// Bundle represents a KBI kernel bundle with paths to artifacts.
type Bundle struct {
	VmlinuzPath  string
	InitrdPath   string
	ModulesPath  string
	FirmwarePath string
	ConfigPath   string
	BTFPath      string
	Kver         string
	Arch         string
	Tag          string
}
