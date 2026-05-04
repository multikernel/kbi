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

// version is set at link time by the release workflow via
// -ldflags "-X github.com/multikernel/kbi/cmd/kbi/commands.version=vX.Y.Z".
// When unset, resolveVersion falls back to module/VCS build info.
var version string

// resolveVersion returns the binary's version. Precedence:
//  1. -ldflags-injected `version` (used by release artifacts).
//  2. Module version from `go install …@vX.Y.Z`.
//  3. VCS commit (with -dirty suffix if the tree was modified) for `go build`.
func resolveVersion() string {
	if version != "" {
		return version
	}
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
