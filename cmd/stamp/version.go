package main

import "github.com/rossijonas/stamp/internal/cli"

// Version info is now defined in internal/cli/version.go.
// Build injection targets github.com/rossijonas/stamp/internal/cli.Version etc.

var _ = cli.Version // ensure import at build time
