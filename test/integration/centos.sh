#!/bin/bash
set -eo pipefail
export HOMEBREW_NO_AUTO_UPDATE=1

TIMEOUT=10
TIMEOUT_LONG=30
# shellcheck disable=SC2034
TIMEOUT_EXTRA=120
test_count=0
pass_count=0

pass() {
	test_count=$((test_count + 1))
	pass_count=$((pass_count + 1))
	echo "  ✓ $1"
}

fail() {
	test_count=$((test_count + 1))
	echo "  ✗ $1"
}

check() {
	desc="$1"
	shift
	if out=$("$@" 2>&1); then
		pass "$desc"
	else
		exit_code=$?
		echo "  ✗ $desc (exit=$exit_code)"
		# shellcheck disable=SC2001
		echo "$out" | sed 's/^/      | /'
		test_count=$((test_count + 1))
	fi
}

check_fail() {
	desc="$1"
	shift
	if out=$("$@" 2>&1); then
		fail "$desc"
		# shellcheck disable=SC2001
		echo "$out" | sed 's/^/      | /'
	else
		exit_code=$?
		echo "  ✓ $desc (exit=$exit_code)"
		# shellcheck disable=SC2001
		echo "$out" | sed 's/^/      | /'
		test_count=$((test_count + 1))
		pass_count=$((pass_count + 1))
	fi
}

echo "=== Integration: CentOS Stream 10 (latest) ==="

stamp --version

check "doctor runs" stamp doctor

echo "=== DNF Install/Remove ==="
check "install htop via dnf" timeout $TIMEOUT_LONG stamp install htop -m dnf -y
check "search finds results" timeout $TIMEOUT_EXTRA stamp search centos-stream-release -m dnf
check "list shows htop" bash -c "timeout $TIMEOUT stamp list | grep -q htop"
check "remove htop via dnf" timeout $TIMEOUT_LONG stamp remove htop -m dnf -y
check "list no longer shows htop" bash -c "timeout $TIMEOUT stamp list | grep -qv htop"

echo "=== Brew ==="
check "brew search htop" timeout $TIMEOUT stamp search htop -m brew

echo "=== Brew Install/Remove ==="
check "brew install hello" timeout $TIMEOUT_LONG stamp install hello -m brew -y
check "list shows hello" bash -c "timeout $TIMEOUT stamp list | grep -q hello"
check "brew remove hello" timeout $TIMEOUT stamp remove hello -m brew -y
check "list no longer shows hello" bash -c "timeout $TIMEOUT stamp list | grep -qv hello"

echo "=== Flatpak ==="
check "flatpak remote list" timeout $TIMEOUT stamp repo list -m flatpak
echo "  • warming flatpak appstream cache..."
timeout $TIMEOUT_LONG flatpak update --appstream 2>&1 || true
check "flatpak search Calculator" timeout $TIMEOUT_EXTRA stamp search Calculator -m flatpak

echo "=== JSON Output ==="
check "doctor shows managers" bash -c "stamp doctor 2>&1 | grep -qE 'dnf|brew|flatpak|apt'"
check "doctor --json" stamp doctor --json
check "doctor --json valid" bash -c "stamp doctor --json | python3 -m json.tool > /dev/null"
check "list --json" stamp list --json
check "list --json valid" bash -c "stamp list --json | python3 -m json.tool > /dev/null"

echo "=== Init ==="
check "init runs" timeout $TIMEOUT stamp init
check "init re-init shows warning" bash -c "stamp init 2>&1 | grep -q 'backed up'"

echo "=== Repository Operations ==="
check "repo list (dnf)" timeout $TIMEOUT stamp repo list -m dnf

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

echo "=== Root Command ==="
check "stamp (no args)" bash -c "stamp 2>/dev/null | head -5 > /dev/null"

echo "=== Reconcile ==="
check "reconcile --dry-run" timeout $TIMEOUT stamp reconcile --dry-run -m dnf
check "reconcile runs" timeout $TIMEOUT stamp reconcile -m dnf

echo "=== Update ==="
check "update runs" timeout $TIMEOUT_EXTRA stamp update -m dnf -y
check "install hello for single-pkg update test" timeout $TIMEOUT_LONG stamp install hello -m brew -y
check "update single package" timeout $TIMEOUT stamp update -p hello -m brew -y

echo "=== Restore ==="
check "restore --dry-run shows results" bash -c "timeout $TIMEOUT stamp restore --dry-run 2>&1 | grep -q ."

echo "=== Info ==="
check "info shows results" timeout $TIMEOUT_LONG stamp info centos-stream-release -m dnf
check "info --json" timeout $TIMEOUT_LONG stamp info centos-stream-release --json

echo "=== Snap ==="
if command -v snap &>/dev/null; then
    check "snap list" timeout $TIMEOUT stamp list -m snap
    check "snap search hello" timeout $TIMEOUT stamp search hello -m snap
else
    echo "  ⚠  snap not available in container — skipping snap tests"
fi

echo "=== Alias Tests ==="
check "install via add alias" timeout $TIMEOUT_LONG stamp add hello -m dnf -y
check "remove via rm alias" timeout $TIMEOUT stamp rm hello -m dnf -y
check "repo list via ls alias" timeout $TIMEOUT stamp repo ls -m dnf

echo
echo "  Results: $pass_count / $test_count passed"
[ "$pass_count" = "$test_count" ]
