package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

// lookPath is overridable in tests to simulate binary presence.
var lookPath = exec.LookPath

// isTerminal reports whether the given reader is connected to a terminal.
// Declared as a variable so it can be overridden in tests.
// Uses term.IsTerminal for consistency with term.ReadPassword — if this
// returns true, ReadPassword on the same fd will succeed.
var isTerminal = func(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

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
		if _, err := lookPath(bin); err == nil {
			adapters = append(adapters, fn())
		}
	}
	if runtime.GOOS == "linux" {
		if _, err := lookPath("dnf"); err == nil {
			adapters = append(adapters, manager.NewDNF("dnf"))
		} else if _, err := lookPath("yum"); err == nil {
			adapters = append(adapters, manager.NewDNF("yum"))
		}
		if _, err := lookPath("apt"); err == nil {
			adapters = append(adapters, manager.NewAPT("apt"))
		}
		detect("flatpak", func() manager.Adapter { return manager.NewFlatpak() })
		detect("snap", func() manager.Adapter { return manager.NewSnap() })
		detect("zypper", func() manager.Adapter { return manager.NewZypper() })
		if _, err := lookPath("paru"); err == nil {
			adapters = append(adapters, manager.NewParu())
		} else if _, err := lookPath("pacman"); err == nil {
			adapters = append(adapters, manager.NewPacman())
		}
	}
	detect("brew", func() manager.Adapter { return manager.NewBrew() })
	detect("pipx", func() manager.Adapter { return manager.NewPipx() })
	detect("uv", func() manager.Adapter { return manager.NewUv() })
	detect("npm", func() manager.Adapter { return manager.NewNpm() })
	detect("cargo", func() manager.Adapter { return manager.NewCargo() })
	detect("go", func() manager.Adapter { return manager.NewGo() })
	if runtime.GOOS == "darwin" {
		detect("port", func() manager.Adapter { return manager.NewMacPorts() })
	}
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
		Short:         "A lightweight yet powerful tool that wraps many package managers into one",
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
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), stampBanner)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  stamp — A lightweight yet powerful tool that wraps many package managers into one.\n  Install, track, and restore without changing your tools.\n  One manifest. One command to restore it all.\n\n")
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Don't know where to start? Try:\n\n")
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  stamp setup    — Run first-time setup wizard")
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  stamp --help   — See all available commands")
			_, _ = fmt.Fprintln(cmd.ErrOrStderr())
			return cmd.Help()
		},
	}

	root.Version = Version
	root.SetHelpTemplate(`{{if .Parent}}{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)
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
	root.AddCommand(newInitCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newProvidesCmd())
	root.AddCommand(newAutoremoveCmd())
	root.AddCommand(newCleanCmd())
	root.AddCommand(newHoldCmd())
	root.AddCommand(newUnholdCmd())
	root.AddCommand(newHeldCmd())
	root.AddCommand(newOverrideCmd())
	root.AddCommand(newManCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newSelfUpdateCmd())
	root.AddCommand(newAutoReconcileCmd())

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
