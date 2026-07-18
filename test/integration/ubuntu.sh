#!/bin/sh
set -e

echo "=== Integration: Ubuntu (latest) ==="

stamp --version

# Skip setup/init — brew init fails as root in containers
stamp doctor

# APT: search
timeout 10 stamp search htop -m apt | grep htop

# APT: install, list, remove
stamp install htop -m apt
stamp list | grep htop
stamp remove htop -m apt

# APT: repo operations
stamp repo add ppa:git-core/ppa -m apt
stamp repo list | grep ppa
stamp repo remove ppa:git-core/ppa -m apt

echo "✅ All Ubuntu integration checks passed"
