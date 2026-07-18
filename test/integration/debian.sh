#!/bin/sh
set -e

echo "=== Integration: Debian (latest) ==="

stamp --version
stamp doctor

# APT: search
timeout 10 stamp search htop -m apt | grep htop

# APT: install, list, remove
stamp install htop -m apt
stamp list | grep htop
stamp remove htop -m apt

# APT: repo operations — PPA should fail gracefully (no add-apt-repository)
stamp repo add ppa:git-core/ppa -m apt 2>&1 || echo "  (PPA add skipped — add-apt-repository not available on Debian)"

echo "✅ All Debian integration checks passed"
