package cli

import (
	"runtime/debug"
	"strings"
)

var (
	// Version is the current release version, injected via ldflags at build time.
	// Always stored without a "v" prefix — add "v" only at display boundaries.
	Version = "dev"
	// Commit is the git commit hash, injected via ldflags.
	Commit = "none"
	// Date is the build date, injected via ldflags.
	Date = "unknown"
)

func init() {
	Version = strings.TrimPrefix(Version, "v")
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = strings.TrimPrefix(info.Main.Version, "v")
	}

	rootCmd.Version = "v" + Version
}
