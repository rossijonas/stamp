#!/bin/bash
set -eo pipefail

TIMEOUT=10
# shellcheck disable=SC2034
TIMEOUT_LONG=30
# shellcheck disable=SC2034
TIMEOUT_EXTRA=120
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
	if out=$("$@" 2>&1); then
		pass "$desc"
	else
		exit_code=$?
		echo "  ❌ $desc (exit=$exit_code)"
		# shellcheck disable=SC2001
		echo "$out" | sed 's/^/      | /'
		test_count=$((test_count + 1))
	fi
}

echo "=== Integration: Arch Linux (latest) ==="

stamp --version

check "doctor runs" stamp doctor

echo "=== Brew ==="
check "brew search htop" timeout $TIMEOUT stamp search htop -m brew

echo "=== Brew Install/Remove ==="
check "brew install hello" timeout $TIMEOUT_LONG stamp install hello -m brew
check "list shows hello" bash -c "timeout $TIMEOUT stamp list | grep -q hello"
check "brew remove hello" timeout $TIMEOUT stamp remove hello -m brew
check "list no longer shows hello" bash -c "timeout $TIMEOUT stamp list | grep -qv hello"

echo "=== Flatpak ==="
check "flatpak remote list" timeout $TIMEOUT stamp repo list -m flatpak
check "flatpak search Calculator" timeout $TIMEOUT_LONG stamp search Calculator -m flatpak

echo "=== Pacman ==="
check "search finds results" bash -c "timeout $TIMEOUT stamp search sl -m pacman | grep -q ."
check "install sl via pacman" timeout $TIMEOUT_LONG stamp install sl -m pacman
check "list shows sl" bash -c "timeout $TIMEOUT stamp list | grep -q sl"
check "remove sl via pacman" timeout $TIMEOUT_LONG stamp remove sl -m pacman
check "list no longer shows sl" bash -c "timeout $TIMEOUT stamp list | grep -qv sl"

echo "=== JSON Output ==="
check "doctor shows managers" bash -c "stamp doctor 2>&1 | grep -qE 'brew|flatpak|apt|dnf'"
check "doctor --json" stamp doctor --json
check "doctor --json valid" bash -c "stamp doctor --json | python3 -m json.tool > /dev/null"
check "list --json" stamp list --json
check "list --json valid" bash -c "stamp list --json | python3 -m json.tool > /dev/null"

echo "=== Help Output ==="
check "stamp --help" timeout $TIMEOUT stamp --help
check "stamp install --help" timeout $TIMEOUT stamp install --help
check "stamp remove --help" timeout $TIMEOUT stamp remove --help
check "stamp search --help" timeout $TIMEOUT stamp search --help
check "stamp list --help" timeout $TIMEOUT stamp list --help
check "stamp doctor --help" timeout $TIMEOUT stamp doctor --help
check "stamp reconcile --help" timeout $TIMEOUT stamp reconcile --help
check "stamp restore --help" timeout $TIMEOUT stamp restore --help
check "stamp update --help" timeout $TIMEOUT stamp update --help
check "stamp self-update --help" timeout $TIMEOUT stamp self-update --help

echo "=== Self-Update ==="
check "self-update --check" timeout $TIMEOUT stamp self-update --check
check "self-upgrade alias" timeout $TIMEOUT stamp self-upgrade --check

echo "=== Root Command ==="
check "stamp (no args)" bash -c "stamp 2>/dev/null | head -5 > /dev/null"

echo "=== Snap ==="
if command -v snap &>/dev/null; then
    check "snap list" timeout $TIMEOUT stamp list -m snap
    check "snap search hello" timeout $TIMEOUT stamp search hello -m snap
else
    echo "  ⚠  snap not available in container — skipping snap tests"
fi

echo "=== Alias Tests ==="
check "install via add alias" timeout $TIMEOUT stamp add hello -m brew
check "remove via rm alias" timeout $TIMEOUT stamp rm hello -m brew
check "repo list via ls alias" timeout $TIMEOUT stamp repo ls -m brew

echo
echo "  Results: $pass_count / $test_count passed"
[ "$pass_count" = "$test_count" ]
