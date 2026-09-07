# shellcheck shell=bash
# run_restore_batch_test — restore batch smoke test shared by platform scripts.
# Requires the including script's harness (check) to be defined BEFORE sourcing.
# usage: run_restore_batch_test <manager> <fixture> <expected_count>
run_restore_batch_test() {
	echo "=== Restore (batch) ==="
	check "copy restore manifest" bash -c \
		"mkdir -p ~/.config/stamp && cp /test/fixtures/$2 ~/.config/stamp/manifest.toml"
	check "restore batches per manager" bash -c \
		"timeout $TIMEOUT_EXTRA stamp restore -y 2>&1 | grep -q 'restored $3 package(s) via $1'"
	check "restored packages present" bash -c "command -v htop && command -v tree"
}