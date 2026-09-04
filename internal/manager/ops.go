package manager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

const pacmanOptionsSection = "[options]"

// sudoExec runs a pre-built args slice (first element must be the binary,
// typically from sudoCmd) with streamed IO and wraps the error with errMsg.
func sudoExec(ctx context.Context, exec Executor, args []string, errMsg string) error {
	_, err := exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

// runSingle gates a destructive single-package operation: requireConsent,
// ValidatePackageName, then sudoExec with errMsg="failed to <verb> <pkg>".
func runSingle(ctx context.Context, exec Executor, args []string, verb, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	return sudoExec(ctx, exec, args, fmt.Sprintf("failed to %s %s", verb, pkg))
}

// runBatch gates a destructive batch operation: requireConsent,
// validatePackages, then sudoExec with errMsg="failed to <verb> packages".
func runBatch(ctx context.Context, exec Executor, args []string, verb string, pkgs []string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	return sudoExec(ctx, exec, args, fmt.Sprintf("failed to %s packages", verb))
}

const pacmanConfPath = "/etc/pacman.conf"

func pacmanConfRead(ctx context.Context, exec Executor) ([]string, error) {
	out, err := exec(ctx, "cat", pacmanConfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pacman.conf: %w", err)
	}
	return strings.Split(string(out), "\n"), nil
}

func pacmanConfWrite(ctx context.Context, exec Executor, lines []string) error {
	tmpPath := fmt.Sprintf("/tmp/stamp-pacman-conf.%d", time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()
	args := sudoCmd("cp", tmpPath, pacmanConfPath)
	_, err := exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to write pacman.conf: %w", err)
	}
	return nil
}

func pacmanIgnorePkg(ctx context.Context, exec Executor) ([]string, error) {
	lines, err := pacmanConfRead(ctx, exec)
	if err != nil {
		return nil, err
	}
	inOptions := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == pacmanOptionsSection {
			inOptions = true
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "[") && trimmed != pacmanOptionsSection {
			inOptions = false
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "IgnorePkg") {
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx < 0 {
				continue
			}
			value := strings.TrimSpace(trimmed[eqIdx+1:])
			return strings.Fields(value), nil
		}
	}
	return nil, nil
}

// ignorePkgLine returns the index of the IgnorePkg line within the [options]
// section and the index of its '='. found is false when no valid IgnorePkg line
// appears before the next section. Both hold and unhold share this scan.
func ignorePkgLine(lines []string) (idx, eqIdx int, found bool) {
	inOptions := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == pacmanOptionsSection {
			inOptions = true
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "[") && trimmed != pacmanOptionsSection {
			break
		}
		if inOptions && strings.HasPrefix(trimmed, "IgnorePkg") {
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx < 0 {
				continue
			}
			return i, eqIdx, true
		}
	}
	return 0, 0, false
}

func pacmanHold(ctx context.Context, exec Executor, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	lines, err := pacmanConfRead(ctx, exec)
	if err != nil {
		return err
	}

	if idx, eqIdx, ok := ignorePkgLine(lines); ok {
		value := strings.TrimSpace(lines[idx][eqIdx+1:])
		for _, p := range strings.Fields(value) {
			if p == pkg {
				return nil
			}
		}
		lines[idx] = lines[idx] + " " + pkg
		return pacmanConfWrite(ctx, exec, lines)
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == pacmanOptionsSection {
			lines = append(lines[:i+1], append([]string{"IgnorePkg = " + pkg}, lines[i+1:]...)...)
			return pacmanConfWrite(ctx, exec, lines)
		}
	}
	return fmt.Errorf("could not find [options] section in pacman.conf")
}

func pacmanUnhold(ctx context.Context, exec Executor, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	lines, err := pacmanConfRead(ctx, exec)
	if err != nil {
		return err
	}

	idx, eqIdx, ok := ignorePkgLine(lines)
	if !ok {
		return fmt.Errorf("package %s is not held", pkg)
	}
	trimmed := strings.TrimSpace(lines[idx])
	before := strings.TrimSpace(trimmed[:eqIdx])
	value := strings.TrimSpace(trimmed[eqIdx+1:])
	pkgs := strings.Fields(value)
	newPkgs := make([]string, 0, len(pkgs))
	found := false
	for _, p := range pkgs {
		if p == pkg {
			found = true
		} else {
			newPkgs = append(newPkgs, p)
		}
	}
	if !found {
		return fmt.Errorf("package %s is not held", pkg)
	}
	if len(newPkgs) == 0 {
		lines[idx] = before + " ="
	} else {
		lines[idx] = before + " = " + strings.Join(newPkgs, " ")
	}
	return pacmanConfWrite(ctx, exec, lines)
}
