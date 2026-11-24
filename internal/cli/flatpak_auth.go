package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
)

// flatpakPolkitActions are the polkit actions a system-scope flatpak update
// can trigger: remote metadata and the appstream cache.
var flatpakPolkitActions = []string{
	"org.freedesktop.Flatpak.metadata-update",
	"org.freedesktop.Flatpak.appstream-update",
}

// polkitChallenge reports whether the given action requires interactive
// authentication. It never prompts: pkcheck without --allow-user-interaction
// only reports the outcome. Exit 2 means a challenge is required, so a prompt
// appears in the terminal when no graphical polkit agent is available.
// Overridable in tests.
var polkitChallenge = func(ctx context.Context, action string) (bool, error) {
	//nolint:gosec // fixed binary; action IDs come from flatpakPolkitActions
	cmd := exec.CommandContext(ctx, "pkcheck", "--action-id", action, "--process", strconv.Itoa(os.Getpid()))
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == 2, nil
	}
	return false, err
}

// needsPolkitAuth reports whether any flatpak system action needs interactive
// authentication. Missing pkcheck or unexpected errors report false so the
// update is not blocked by a failed probe.
func needsPolkitAuth(ctx context.Context) bool {
	for _, action := range flatpakPolkitActions {
		challenge, err := polkitChallenge(ctx, action)
		if err == nil && challenge {
			return true
		}
	}
	return false
}
