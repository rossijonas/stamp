#!/bin/bash
set -eo pipefail

TIMEOUT=10
test_count=0
pass_count=0

pass() {
	test_count=$((test_count + 1))
	pass_count=$((pass_count + 1))
	echo "  ✅ $1"
}

fail() {
	test_count=$((test_count + 1))
	echo "  ❌ $1"
}

check() {
	desc="$1"
	shift
	if "$@"; then
		pass "$desc"
	else
		fail "$desc"
	fi
}

echo "=== Integration: Fedora (latest) ==="

stamp --version

check "doctor runs" stamp doctor

check "search htop via dnf" timeout $TIMEOUT stamp search htop -m dnf

# flatpak search may return no results (no remote metadata in container)
timeout $TIMEOUT stamp search org.gnome.eog -m flatpak 2>/dev/null && \
	pass "flatpak EOG search" || \
	echo "  ⚠  flatpak search: no results (expected in container without remote metadata)"

echo
echo "  Results: $pass_count / $test_count passed"
[ "$pass_count" = "$test_count" ]
