#!/bin/bash
set -eo pipefail

TIMEOUT=10
TIMEOUT_LONG=30
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

echo "=== Integration: Debian (latest) ==="

stamp --version

check "doctor runs" stamp doctor

check "search htop via apt" timeout $TIMEOUT stamp search htop -m apt

check "install htop via apt" timeout $TIMEOUT_LONG stamp install htop -m apt
check "list shows htop" timeout $TIMEOUT stamp list
check "remove htop via apt" timeout $TIMEOUT_LONG stamp remove htop -m apt

# PPA should fail gracefully — no add-apt-repository on Debian
if stamp repo add ppa:git-core/ppa -m apt 2>/dev/null; then
	fail "PPA add unexpectedly succeeded"
else
	pass "PPA add fails gracefully (no add-apt-repository)"
fi

echo
echo "  Results: $pass_count / $test_count passed"
[ "$pass_count" = "$test_count" ]
