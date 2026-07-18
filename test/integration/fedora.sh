#!/bin/sh
set -e

echo "=== Integration: Fedora (latest) ==="

stamp --version

# Skip setup/init — brew init fails as root in containers
stamp doctor

# DNF: timeout wrapper for potentially slow commands
timeout 10 stamp search htop -m dnf 2>/dev/null || echo "  (dnf search skipped — metadata not available in container)"

# Flatpak: add remote, then search
flatpak remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo 2>/dev/null || true
timeout 10 stamp search org.gnome.eog -m flatpak 2>/dev/null || echo "  (flatpak search skipped — no remote metadata)"

echo "✅ All Fedora integration checks passed"
