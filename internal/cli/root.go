package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

type ctxKey struct{}

// AppContext holds the runtime dependencies for CLI commands.
type AppContext struct {
	adapters     []manager.Adapter
	manifest     *manifest.Manifest
	manifestPath string
	manifestErr  error
	config       *Config
	yes          bool
	verbose      bool
	json         bool
	noColor      bool
}

// NoColor reports whether the NO_COLOR environment variable is set.
// Used by any code that outputs ANSI escape sequences.
func NoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

func xdgConfigDir() string {
	d := os.Getenv("XDG_CONFIG_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".config")
	}
	return filepath.Join(d, "stamp")
}

func manifestPath() string {
	return filepath.Join(xdgConfigDir(), "manifest.toml")
}

func configPath() string {
	return filepath.Join(xdgConfigDir(), "config.toml")
}

// RootOption configures the root command. Used for testing injection.
type RootOption func(*rootConfig)

type rootConfig struct {
	adapters     []manager.Adapter
	configPath   string
	manifestPath string
}

// WithAdapters injects mock adapters for testing instead of real system discovery.
func WithAdapters(a []manager.Adapter) RootOption {
	return func(c *rootConfig) { c.adapters = a }
}

// WithConfigPath overrides the config path. Used for test isolation.
func WithConfigPath(p string) RootOption {
	return func(c *rootConfig) { c.configPath = p }
}

// WithManifestPath overrides the manifest path. Used for test isolation.
func WithManifestPath(p string) RootOption {
	return func(c *rootConfig) { c.manifestPath = p }
}

func detectAdapters() []manager.Adapter {
	adapters := make([]manager.Adapter, 0)
	detect := func(bin string, fn func() manager.Adapter) {
		if _, err := exec.LookPath(bin); err == nil {
			adapters = append(adapters, fn())
		}
	}
	if runtime.GOOS == "linux" {
		detect("dnf", func() manager.Adapter { return manager.NewDNF() })
		detect("flatpak", func() manager.Adapter { return manager.NewFlatpak() })
	}
	detect("brew", func() manager.Adapter { return manager.NewBrew() })
	return adapters
}

func newAppContext(yes, verbose, json bool, adapters []manager.Adapter, cfgPath, mfPath string) (*AppContext, error) {
	ctx := &AppContext{yes: yes, verbose: verbose, json: json, adapters: adapters, manifestPath: mfPath}

	// Load config
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
		cfg = &Config{Precedence: []string{"dnf", "flatpak", "brew"}}
	}
	ctx.config = cfg

	// Load or create manifest
	m, err := manifest.Load(mfPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			ctx.manifestErr = fmt.Errorf("failed to load manifest: %w", err)
			m = &manifest.Manifest{
				Version:   1,
				System:    runtime.GOOS,
				Packages:  []manifest.Package{},
				UpdatedAt: time.Now(),
			}
		} else {
			m = &manifest.Manifest{
				Version:   1,
				System:    runtime.GOOS,
				Packages:  []manifest.Package{},
				UpdatedAt: time.Now(),
			}
		}
	}
	ctx.manifest = m

	ctx.noColor = NoColor()

	return ctx, nil
}

func (a *AppContext) saveManifest() error {
	return a.manifest.Save(a.manifestPath)
}

func appFromCtx(cmd *cobra.Command) *AppContext {
	v := cmd.Context().Value(ctxKey{})
	if v == nil {
		return nil
	}
	return v.(*AppContext)
}

// NewRootCmd creates a new root command with all subcommands registered.
// Pass WithAdapters(...) to inject mock adapters for testing.
func NewRootCmd(opts ...RootOption) *cobra.Command {
	var cfg rootConfig
	for _, o := range opts {
		o(&cfg)
	}

	root := &cobra.Command{
		Use:           "stamp",
		Short:         "A lightweight yet powerful wrapper for your native package managers",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			yes, _ := cmd.Flags().GetBool("yes")
			verbose, _ := cmd.Flags().GetBool("verbose")
			json, _ := cmd.Flags().GetBool("json")

			cPath := cfg.configPath
			if cPath == "" {
				cPath = configPath()
			}
			mPath := cfg.manifestPath
			if mPath == "" {
				mPath = manifestPath()
			}

			adapters := cfg.adapters
			if adapters == nil {
				adapters = detectAdapters()
			}

			app, err := newAppContext(yes, verbose, json, adapters, cPath, mPath)
			if err != nil {
				return fmt.Errorf("initialization failed: %w", err)
			}
			cmd.SetContext(context.WithValue(cmd.Context(), ctxKey{}, app))
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Don't know where to start? Try:\n\n")
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  stamp hello    — Learn about stamp and next steps")
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  stamp --help   — See all available commands")
			return cmd.Help()
		},
	}

	root.Version = Version
	root.PersistentFlags().BoolP("verbose", "v", false, "enable debug logging")
	root.PersistentFlags().BoolP("json", "j", false, "output results in JSON format")
	root.PersistentFlags().BoolP("yes", "y", false, "auto-accept all prompts")

	root.AddCommand(newInstallCmd())
	root.AddCommand(newRemoveCmd())
	root.AddCommand(newReinstallCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newInfoCmd())
	root.AddCommand(newHelloCmd())
	root.AddCommand(newRepoCmd())
	root.AddCommand(newReconcileCmd())
	root.AddCommand(newRestoreCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newCompletionCmd())
	root.AddCommand(newManCmd())

	return root
}

var rootCmd = NewRootCmd()

// Execute is the entry point for the CLI, called from cmd/stamp/main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Exit codes following sysexits.h conventions.
const (
	ExitUsage       = 64 // EX_USAGE
	ExitDataErr     = 65 // EX_DATAERR
	ExitUnavailable = 69 // EX_UNAVAILABLE
	ExitSoftware    = 70 // EX_SOFTWARE
	ExitConfig      = 78 // EX_CONFIG
)
