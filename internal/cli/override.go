package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

// overrideAdapter is an interface for adapters that support sandbox override.
// Currently only flatpak implements this, but using an interface allows
// mocking in tests without type-asserting to a concrete type.
type overrideAdapter interface {
	Override(ctx context.Context, appID string, flags manager.OverrideFlags) error
}

func newOverrideCmd() *cobra.Command {
	var managerFlag string
	var filesystem, socket, device, env []string
	var reset, show, systemScope bool

	cmd := &cobra.Command{
		Use:   "override <app-id>",
		Short: "Manage Flatpak sandbox permissions",
		Long: `Set, show, or reset Flatpak sandbox permissions for an application.

Requires --manager flatpak. Use repeatable flags for filesystem, socket,
device, and environment variables. At least one action flag is required.`,
		Example: `  # grant filesystem access (repeatable)
  stamp override firefox -m flatpak --filesystem=host

  # grant socket access (repeatable)
  stamp override firefox -m flatpak --socket=wayland

  # reset all overrides to defaults
  stamp override firefox -m flatpak --reset

  # show current overrides
  stamp override firefox -m flatpak --show

  # apply system-wide (requires sudo)
  stamp override firefox -m flatpak --system --filesystem=host`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			appID := args[0]

			if err := manager.ValidatePackageName(appID); err != nil {
				return fmt.Errorf("invalid app-id: %w", err)
			}

			// Resolve the adapter
			resolved := manager.ResolveManager(managerFlag)
			var adapter manager.Adapter
			for _, a := range app.adapters {
				if a.Name() == resolved {
					adapter = a
					break
				}
			}
			if adapter == nil {
				return fmt.Errorf("manager %q not available", managerFlag)
			}

			// Override is flatpak-only. Check both the adapter name and the optional interface.
			if adapter.Name() != "flatpak" {
				return fmt.Errorf("override is only supported for flatpak")
			}
			overrider, ok := adapter.(overrideAdapter)
			if !ok {
				return fmt.Errorf("override is only supported for flatpak")
			}

			flags := manager.OverrideFlags{
				Filesystem: filesystem,
				Socket:     socket,
				Device:     device,
				Env:        env,
				Reset:      reset,
				Show:       show,
				System:     systemScope,
			}

			if err := overrider.Override(cmd.Context(), appID, flags); err != nil {
				return fmt.Errorf("override failed: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use (required)")
	cmd.Flags().StringArrayVar(&filesystem, "filesystem", nil, "grant filesystem access (repeatable)")
	cmd.Flags().StringArrayVar(&socket, "socket", nil, "grant socket access (repeatable)")
	cmd.Flags().StringArrayVar(&device, "device", nil, "grant device access (repeatable)")
	cmd.Flags().StringArrayVar(&env, "env", nil, "set environment variable KEY=VALUE (repeatable)")
	cmd.Flags().BoolVar(&reset, "reset", false, "reset all overrides to defaults")
	cmd.Flags().BoolVar(&show, "show", false, "show current overrides")
	cmd.Flags().BoolVar(&systemScope, "system", false, "apply system-wide (requires sudo)")
	cmd.MarkFlagsMutuallyExclusive("reset", "show")
	return cmd
}
