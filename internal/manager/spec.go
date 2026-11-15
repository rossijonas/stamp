package manager

import (
	"fmt"
	"regexp"
	"strings"
)

// Spec is a validated package specification. Raw is passed to the native
// command; Name is the manifest identity (qualified forms are stripped).
type Spec struct {
	Raw  string
	Name string
}

var (
	genericSpecRegex  = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.+-]*$`)
	aptSpecRegex      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.+-]*(?:=[a-zA-Z0-9_.:+~-]+|/[a-zA-Z0-9_.-]+|:[a-zA-Z0-9_-]+)?$`)
	dnfSpecRegex      = regexp.MustCompile(`^[a-zA-Z0-9_/(][a-zA-Z0-9_./:+~*?\[\]()!<>= -]*$`)
	pacmanSpecRegex   = regexp.MustCompile(`^(?:[a-zA-Z0-9_.-]+/)?[a-zA-Z0-9_+.-]+(?:[<>=]+[a-zA-Z0-9_.:+-]+)?$`)
	zypperSpecRegex   = regexp.MustCompile(`^(?:([a-zA-Z0-9_.-]+):)?([a-zA-Z0-9_+.-]+)(?:\.([a-zA-Z0-9_]+))?((?:<=|>=|=|<|>)[a-zA-Z0-9_.:+-]+)?$`)
	snapSpecRegex     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*(?:_[a-zA-Z0-9_.-]+)?$`)
	flatpakSpecRegex  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	brewSpecRegex     = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.+@-]*(/[a-zA-Z0-9_][a-zA-Z0-9_.+@-]*){0,2}$`)
	macportsSpecRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.+-]*(?:@[a-zA-Z0-9_.+-]+)?$`)
	cargoSpecRegex    = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.+-]*(?:@[a-zA-Z0-9_.^~<>=*+-]+)?$`)
	pipxSpecRegex     = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.+-]*(?:\[[a-zA-Z0-9_,.-]+\])?(?:==[a-zA-Z0-9_.+!*-]+)?$`)
	uvSpecRegex       = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.+-]*(?:@[a-zA-Z0-9_.^~*+-]+)?(?:\[[a-zA-Z0-9_,.-]+\])?(?:==[a-zA-Z0-9_.+!*-]+)?$`)
)

var specParsers = map[string]func(string) (Spec, error){
	"apt":      parseAptSpec,
	"dnf":      parseDNFSpec,
	"yum":      parseDNFSpec,
	"pacman":   parsePacmanSpec,
	"paru":     parseParuSpec,
	"zypper":   parseZypperSpec,
	"snap":     parseSnapSpec,
	"flatpak":  parseFlatpakSpec,
	"brew":     parseBrewSpec,
	"macports": parseMacportsSpec,
	"go":       parseGoSpec,
	"cargo":    parseCargoSpec,
	"npm":      parseNpmSpec,
	"pipx":     parsePipxSpec,
	"uv":       parseUvSpec,
}

// ParseSpecForManager validates one package spec and derives its manifest
// identity. Validation is lexical; nothing is executed. Names are passed to
// exec without a shell, so the security boundary is the leading '-' check plus
// a per-manager allowlist.
func ParseSpecForManager(managerName, raw string) (Spec, error) {
	if err := validateSafeArg(raw); err != nil {
		return Spec{}, err
	}
	if parse, ok := specParsers[managerName]; ok {
		return parse(raw)
	}
	return parseGenericSpec(raw)
}

func parseGenericSpec(raw string) (Spec, error) {
	if !genericSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parseGoSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "@") {
		return Spec{}, fmt.Errorf("invalid module path %q: version pins are not supported; stamp installs @latest (ADR-010)", raw)
	}
	if err := ValidateModulePath(raw); err != nil {
		return Spec{}, err
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parseNpmSpec(raw string) (Spec, error) {
	if err := ValidateNpmPackageName(raw); err != nil {
		return Spec{}, err
	}
	return Spec{Raw: raw, Name: npmSpecName(raw)}, nil
}

// ValidateSpecForManager is the validation-only wrapper.
func ValidateSpecForManager(managerName, spec string) error {
	_, err := ParseSpecForManager(managerName, spec)
	return err
}

func validateSafeArg(spec string) error {
	if spec == "" {
		return fmt.Errorf("invalid package name: cannot be empty")
	}
	if len(spec) > 512 {
		return fmt.Errorf("invalid package name %q: too long", spec)
	}
	if strings.HasPrefix(spec, "-") {
		return fmt.Errorf("invalid package name %q: cannot start with '-'", spec)
	}
	for _, r := range spec {
		if r == 0 || r == '\n' || r == '\r' || r == '\t' || r < 0x20 || r == 0x7f {
			return fmt.Errorf("invalid package name %q: contains control characters", spec)
		}
	}
	if strings.ContainsAny(spec, ";&|$`\\\"'") {
		return fmt.Errorf("invalid package name %q: contains shell metacharacters", spec)
	}
	return nil
}

func invalidSpec(raw string) error {
	return fmt.Errorf("invalid package name %q: contains invalid characters", raw)
}

func parseAptSpec(raw string) (Spec, error) {
	if strings.HasSuffix(raw, "+") && !strings.HasSuffix(raw, "++") {
		return Spec{}, fmt.Errorf("invalid package name %q: trailing '+' is an apt modifier; use stamp install", raw)
	}
	if strings.HasSuffix(raw, "-") {
		return Spec{}, fmt.Errorf("invalid package name %q: trailing '-' is an apt modifier; use stamp remove", raw)
	}
	if !aptSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	name := raw
	if i := strings.IndexAny(name, "=:/"); i >= 0 {
		name = name[:i]
	}
	return Spec{Raw: raw, Name: name}, nil
}

func parseDNFSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "://") || strings.HasSuffix(raw, ".rpm") {
		return Spec{}, fmt.Errorf("invalid package name %q: paths and URLs are not package names", raw)
	}
	if strings.Contains(raw, "/") && !strings.HasPrefix(raw, "/") {
		return Spec{}, fmt.Errorf("invalid package name %q: only file-provides paths may contain '/'", raw)
	}
	if strings.HasPrefix(raw, "@") {
		return Spec{}, fmt.Errorf("invalid package name %q: group syntax is not a package name; use --group", raw)
	}
	if strings.Contains(raw, " ") && !strings.ContainsAny(raw, "()<>=!") {
		return Spec{}, invalidSpec(raw)
	}
	if !dnfSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parsePacmanSpec(raw string) (Spec, error) {
	if !pacmanSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	name := raw
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.IndexAny(name, "<>="); i >= 0 {
		name = name[:i]
	}
	return Spec{Raw: raw, Name: name}, nil
}

func parseParuSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "/") {
		return Spec{}, fmt.Errorf("invalid package name %q: repo-qualified targets are not supported; use paru --repo", raw)
	}
	if !genericSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parseZypperSpec(raw string) (Spec, error) {
	m := zypperSpecRegex.FindStringSubmatch(raw)
	if m == nil {
		return Spec{}, invalidSpec(raw)
	}
	if kind := m[1]; kind != "" {
		switch kind {
		case "patch", "pattern", "product", "srcpackage", "application":
			return Spec{}, fmt.Errorf("invalid package name %q: %s: selectors are not package names; use zypper --type", raw, kind)
		}
	}
	name := m[2]
	if i := strings.LastIndexByte(name, '.'); i > 0 && isKnownArch(name[i+1:]) {
		name = name[:i]
	}
	return Spec{Raw: raw, Name: name}, nil
}

// isKnownArch reports whether s is a common RPM architecture suffix, so a
// dotted package name (e.g. python3.11) is not mistaken for an arch qualifier.
func isKnownArch(s string) bool {
	switch s {
	case "x86_64", "i386", "i486", "i586", "i686", "noarch", "aarch64",
		"armv6hl", "armv7hl", "armv7hnl", "ppc64", "ppc64le", "s390x", "src":
		return true
	default:
		return false
	}
}

func parseSnapSpec(raw string) (Spec, error) {
	if !snapSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parseFlatpakSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "/") {
		return Spec{}, fmt.Errorf("invalid package name %q: use an app ID and -m flatpak; flatpak install <remote> <ref> is not supported", raw)
	}
	if !flatpakSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parseBrewSpec(raw string) (Spec, error) {
	if !brewSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: raw}, nil
}

func parseMacportsSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "+") {
		return Spec{}, fmt.Errorf("invalid package name %q: port variants are separate arguments and are not supported", raw)
	}
	if !macportsSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	name := raw
	if i := strings.IndexByte(name, '@'); i >= 0 {
		name = name[:i]
	}
	return Spec{Raw: raw, Name: name}, nil
}

func parseCargoSpec(raw string) (Spec, error) {
	if !cargoSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	name := raw
	if i := strings.IndexByte(name, '@'); i >= 0 {
		name = name[:i]
	}
	return Spec{Raw: raw, Name: name}, nil
}

func parsePipxSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "@") || strings.Contains(raw, "/") {
		return Spec{}, fmt.Errorf("invalid package name %q: VCS/URL specs are not supported; use pipx directly", raw)
	}
	if !pipxSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: pythonToolName(raw)}, nil
}

func parseUvSpec(raw string) (Spec, error) {
	if strings.Contains(raw, "/") {
		return Spec{}, fmt.Errorf("invalid package name %q: VCS/URL specs are not supported; use uv directly", raw)
	}
	if !uvSpecRegex.MatchString(raw) {
		return Spec{}, invalidSpec(raw)
	}
	return Spec{Raw: raw, Name: pythonToolName(raw)}, nil
}

// pythonToolName strips version and extras from a pip/pipx/uv spec.
func pythonToolName(raw string) string {
	name := raw
	if i := strings.IndexAny(name, "[="); i >= 0 {
		name = name[:i]
	}
	if i := strings.IndexByte(name, '@'); i >= 0 {
		name = name[:i]
	}
	return name
}
