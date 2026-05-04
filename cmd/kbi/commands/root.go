package commands

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:          "kbi",
	Short:        "Kernel Bundle Image — package kernels as OCI artifacts",
	Version:      resolveVersion(),
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}

// resolveVersion returns the binary's version. When installed via `go install
// …@vX.Y.Z` the module version is recorded in build info. For local `go build`
// from a checkout there's no module version, so we fall back to the VCS commit.
func resolveVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if modified == "true" {
		return rev + "-dirty"
	}
	return rev
}
