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

// resolveByOverride matches an explicit --manager override to an adapter.
func resolveByOverride(adapters []manager.Adapter, override string) (manager.Adapter, error) {
	resolved := manager.ResolveManager(override)
	for _, a := range adapters {
		if a.Name() == resolved {
			return a, nil
		}
	}
	return nil, fmt.Errorf("unknown manager %q", override)
}

// resolveByRules matches a package against declarative pattern rules.
func resolveByRules(config *Config, pkg string, adapters []manager.Adapter) manager.Adapter {
	for _, rule := range config.Rules {
		matched, err := regexp.MatchString(rule.Pattern, pkg)
		if err != nil {
			continue
		}
		if matched {
			for _, a := range adapters {
				if a.Name() == rule.Prefer {
					return a
				}
			}
		}
	}
	return nil
}

// resolveByPrecedence picks the first adapter on the global precedence list.
func resolveByPrecedence(adapters []manager.Adapter, config *Config) manager.Adapter {
	for _, name := range config.Precedence {
		for _, a := range adapters {
			if a.Name() == name {
				return a
			}
		}
	}
	return nil
}

// Resolve applies the 3-tier resolution engine to select a manager.
// Returns the selected adapter or an error if no manager could be chosen.
func (r *Resolver) Resolve(pkg string, override string) (manager.Adapter, error) {
	// Tier 1: Explicit override
	if override != "" {
		return resolveByOverride(r.adapters, override)
	}

	// Tier 2: Pattern rules (highest priority in declarative mode)
	if a := resolveByRules(r.config, pkg, r.adapters); a != nil {
		return a, nil
	}

	// Tier 2 cont.: Global precedence
	if a := resolveByPrecedence(r.adapters, r.config); a != nil {
		return a, nil
	}

	// Tier 3: Ambiguous — fail with instruction
	if len(r.adapters) > 0 {
		return nil, catErr(ErrUsage, "package available in multiple managers; specify --manager")
	}

	return nil, catErr(ErrUnavailable, "no package managers available")
}
