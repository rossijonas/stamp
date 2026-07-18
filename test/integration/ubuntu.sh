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

echo "=== Integration: Ubuntu (latest) ==="

stamp --version

check "doctor runs" stamp doctor

check "search htop via apt" timeout $TIMEOUT stamp search htop -m apt

check "install htop via apt" timeout $TIMEOUT_LONG stamp install htop -m apt
check "list shows htop" timeout $TIMEOUT stamp list
check "remove htop via apt" timeout $TIMEOUT_LONG stamp remove htop -m apt

check "add PPA repo" timeout $TIMEOUT_LONG stamp repo add ppa:git-core/ppa -m apt
check "list includes PPA" timeout $TIMEOUT stamp repo list
check "remove PPA repo" timeout $TIMEOUT_LONG stamp repo remove ppa:git-core/ppa -m apt

echo
echo "  Results: $pass_count / $test_count passed"
[ "$pass_count" = "$test_count" ]
