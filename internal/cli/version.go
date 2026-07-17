package cli

import (
	"runtime/debug"
)

var (
	// Version is the current release version, injected via ldflags at build time.
	Version = "dev"
	// Commit is the git commit hash, injected via ldflags.
	Commit = "none"
	// Date is the build date, injected via ldflags.
	Date = "unknown"
)

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}
