package cli

import (
	"fmt"
	"regexp"

	"github.com/rossijonas/stamp/internal/manager"
)

// Resolver resolves which package manager to use for a given package.
type Resolver struct {
	adapters []manager.Adapter
	config   *Config
}

// NewResolver creates a new Resolver.
func NewResolver(adapters []manager.Adapter, config *Config) *Resolver {
	return &Resolver{adapters: adapters, config: config}
}

// Resolve applies the 3-tier resolution engine to select a manager.
// Returns the selected adapter or an error if no manager could be chosen.
func (r *Resolver) Resolve(pkg string, override string) (manager.Adapter, error) {
	// Tier 1: Explicit override
	if override != "" {
		resolved := manager.ResolveManager(override)
		for _, a := range r.adapters {
			if a.Name() == resolved {
				return a, nil
			}
		}
		return nil, fmt.Errorf("unknown manager %q", override)
	}

	// Tier 2: Pattern rules (highest priority in declarative mode)
	for _, rule := range r.config.Rules {
		matched, err := regexp.MatchString(rule.Pattern, pkg)
		if err != nil {
			continue
		}
		if matched {
			for _, a := range r.adapters {
				if a.Name() == rule.Prefer {
					return a, nil
				}
			}
		}
	}

	// Tier 2 cont.: Global precedence
	for _, name := range r.config.Precedence {
		for _, a := range r.adapters {
			if a.Name() == name {
				return a, nil
			}
		}
	}

	// Tier 3: Ambiguous — fail with instruction
	if len(r.adapters) > 0 {
		return nil, fmt.Errorf("package available in multiple managers; specify --manager")
	}

	return nil, fmt.Errorf("no package managers available")
}
