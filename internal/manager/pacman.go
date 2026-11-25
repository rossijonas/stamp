package manager

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Pacman implements the Adapter interface for Arch Linux's Pacman.
// Shared behavior lives in archHelper; Pacman adds transaction previews.
type Pacman struct {
	archHelper
}

// NewPacman creates a new Pacman adapter.
func NewPacman() *Pacman {
	return &Pacman{
		archHelper: archHelper{
			exec:     defaultExecutor,
			name:     "pacman",
			bin:      "pacman",
			escalate: true,
			strictQu: true,
		},
	}
}

// PreviewInstall previews installing pkg.
// -S --print prints the resolved targets without modifying the system and
// without requiring root.
func (m *Pacman) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidateSpecForManager(m.Name(), pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, "pacman", "-S", "--print", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview install %s: %w", pkg, err)
	}
	return Preview{Output: string(out)}, nil
}

// PreviewRemove previews removing pkg.
func (m *Pacman) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidateSpecForManager(m.Name(), pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, "pacman", "-R", "--print", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview remove %s: %w", pkg, err)
	}
	return Preview{Output: string(out), Noop: strings.Contains(string(out), "was not found")}, nil
}

// PreviewReinstall previews reinstalling pkg.
// pacman reinstalls via the same -S operation as install.
func (m *Pacman) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	return m.PreviewInstall(ctx, pkg)
}

var _ Previewer = (*Pacman)(nil)
var _ Adapter = (*Pacman)(nil)
