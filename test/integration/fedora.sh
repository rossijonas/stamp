#!/bin/bash
set -eo pipefail

TIMEOUT=10
TIMEOUT_LONG=30
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

check_fail() {
	desc="$1"
	shift
	if out=$("$@" 2>&1); then
		fail "$desc"
		# shellcheck disable=SC2001
		echo "$out" | sed 's/^/      | /'
	else
		exit_code=$?
		echo "  ✅ $desc (exit=$exit_code)"
		# shellcheck disable=SC2001
		echo "$out" | sed 's/^/      | /'
		test_count=$((test_count + 1))
		pass_count=$((pass_count + 1))
	fi
}

echo "=== Integration: Fedora (latest) ==="

stamp --version

check "doctor runs" stamp doctor

check "search finds results" bash -c "timeout $TIMEOUT stamp search htop -m dnf | grep -q ."

echo "=== DNF Install/Remove ==="
check "install htop via dnf" timeout $TIMEOUT_LONG stamp install htop -m dnf
check "list shows htop" bash -c "timeout $TIMEOUT stamp list | grep -q htop"
check "reinstall htop via -m flag" timeout $TIMEOUT_LONG stamp reinstall htop -m dnf
check "remove htop via dnf" timeout $TIMEOUT_LONG stamp remove htop -m dnf
check "list no longer shows htop" bash -c "timeout $TIMEOUT stamp list | grep -qv htop"

echo "=== Brew ==="
check "brew search htop" timeout $TIMEOUT stamp search htop -m brew

echo "=== Brew Install/Remove ==="
check "brew install hello" timeout $TIMEOUT_LONG stamp install hello -m brew
check "list shows hello" bash -c "timeout $TIMEOUT stamp list | grep -q hello"
check "brew remove hello" timeout $TIMEOUT stamp remove hello -m brew
check "list no longer shows hello" bash -c "timeout $TIMEOUT stamp list | grep -qv hello"

echo "=== Flatpak ==="
check "flatpak remote list" timeout $TIMEOUT stamp repo list -m flatpak
check "flatpak search Calculator" timeout $TIMEOUT_EXTRA stamp search Calculator -m flatpak

echo "=== JSON Output ==="
check "doctor shows managers" bash -c "stamp doctor 2>&1 | grep -qE 'dnf|brew|flatpak|apt'"
check "doctor --json" stamp doctor --json
check "doctor --json valid" bash -c "stamp doctor --json | python3 -m json.tool > /dev/null"
check "list --json" stamp list --json
check "list --json valid" bash -c "stamp list --json | python3 -m json.tool > /dev/null"

echo "=== Init ==="
check "init runs" timeout $TIMEOUT stamp init
check "init re-init shows warning" bash -c "stamp init 2>&1 | grep -q 'already initialized'"

echo "=== Reconcile ==="
check "reconcile --dry-run" timeout $TIMEOUT stamp reconcile --dry-run -m dnf
check "reconcile runs" timeout $TIMEOUT stamp reconcile -m dnf
check "reconcile all managers" timeout $TIMEOUT stamp reconcile
check "reconcile --yes flag" timeout $TIMEOUT stamp reconcile -y -m dnf

echo "=== Flag Tests ==="
check "search --json" timeout $TIMEOUT_EXTRA stamp search htop --json
check "install --note" timeout $TIMEOUT_EXTRA stamp install hello -m dnf --note "test note"
check "note persisted in manifest" bash -c "stamp list --json | jq -e 'any(.Notes == \"test note\")' > /dev/null"
check "list -m dnf" timeout $TIMEOUT stamp list -m dnf

echo "=== Error Paths ==="
check_fail "install invalid name" timeout $TIMEOUT stamp install -invalid -m dnf
# dnf treats removing non-existent packages as a successful no-op (exit=0), so check_fail is skipped
check "search no results" bash -c "timeout $TIMEOUT stamp search xyznonexistent -m dnf 2>&1 | grep -q 'No matches'"
check "search without -m" timeout $TIMEOUT_EXTRA stamp search htop

echo "=== Repository Operations ==="
check "repo list (dnf)" timeout $TIMEOUT stamp repo list -m dnf
check "repo list (brew)" timeout $TIMEOUT stamp repo list -m brew
check "repo list (flatpak)" timeout $TIMEOUT stamp repo list -m flatpak

echo "=== Restore ==="
check "restore --dry-run shows results" bash -c "timeout $TIMEOUT stamp restore --dry-run 2>&1 | grep -q ."

echo "=== Info ==="
check "info shows results" bash -c "timeout $TIMEOUT stamp info htop -m dnf | grep -q ."
check "info --json" timeout $TIMEOUT stamp info htop --json

echo "=== Help Output ==="
check "stamp --help" timeout $TIMEOUT stamp --help
check "install --help shows -m flag" bash -c "timeout $TIMEOUT stamp install --help | grep -q -- '-m'"
check "remove --help shows -m flag" bash -c "timeout $TIMEOUT stamp remove --help | grep -q -- '-m'"
check "stamp search --help" timeout $TIMEOUT stamp search --help
check "stamp list --help" timeout $TIMEOUT stamp list --help
check "stamp repo --help" timeout $TIMEOUT stamp repo --help
check "stamp doctor --help" timeout $TIMEOUT stamp doctor --help
check "stamp reconcile --help" timeout $TIMEOUT stamp reconcile --help
check "stamp restore --help" timeout $TIMEOUT stamp restore --help
check "stamp update --help" timeout $TIMEOUT stamp update --help
check "stamp self-update --help" timeout $TIMEOUT stamp self-update --help

echo "=== Self-Update ==="
check "self-update --check" timeout $TIMEOUT stamp self-update --check
check "self-upgrade alias" timeout $TIMEOUT stamp self-upgrade --check

echo "=== Update ==="
check "update runs" timeout $TIMEOUT_EXTRA stamp update -m dnf

echo "=== Alias Tests ==="
check "install via add alias" timeout $TIMEOUT stamp add hello -m dnf
check "remove via rm alias" timeout $TIMEOUT stamp rm hello -m dnf
check "repo list via ls alias" timeout $TIMEOUT stamp repo ls -m dnf

echo "=== Snap ==="
if command -v snap &>/dev/null; then
    check "snap list" timeout $TIMEOUT stamp list -m snap
    check "snap search hello" timeout $TIMEOUT stamp search hello -m snap
else
    echo "  ⚠  snap not available in container — skipping snap tests"
fi

echo "=== Shell Completions ==="
check "completion --stdout bash" timeout $TIMEOUT stamp completion --stdout bash
check "completion output is valid bash" bash -c "timeout $TIMEOUT stamp completion --stdout bash | grep -q 'complete'"

echo "=== Root Command ==="
check "stamp (no args)" bash -c "stamp 2>/dev/null | head -5 > /dev/null"

echo
echo "  Results: $pass_count / $test_count passed"
[ "$pass_count" = "$test_count" ]
