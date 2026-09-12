# shellcheck shell=bash
# run_sudo_preflight_check — assert stamp's sudo preflight does not prompt when
# sudo is configured NOPASSWD. The password prompt only appears on a TTY, so the
# command runs under a pseudo-terminal (util-linux `script`). Only the absence of
# a prompt is asserted: the session is killed once the preflight window has
# passed, so a slow network cannot fail the check.
#
# usage: run_sudo_preflight_check <manager>   # apt|dnf|pacman|zypper
# env:   STAMP_BIN  override the stamp binary (default: stamp)
run_sudo_preflight_check() {
	local mgr="$1"
	case "$mgr" in
	apt | dnf | pacman | zypper) ;;
	*)
		echo "  ✗ unknown manager: ${mgr:-<empty>}" >&2
		return 1
		;;
	esac
	local bin="${STAMP_BIN:-stamp}"
	if ! command -v script >/dev/null 2>&1; then
		echo "  ⏭ sudo preflight check skipped (no pty tool)" >&2
		return 0
	fi
	local log
	log="$(mktemp)"
	# Bounded PTY run; the exit status is ignored because timeout kills the
	# command mid-refresh. The captured session is what matters.
	timeout -k 1 "${TIMEOUT:-10}" script -q -c "$bin update -m $mgr --check" "$log" >/dev/null 2>&1 || true
	if grep -qiE 'sudo password|password for' "$log"; then
		sed 's/^/      | /' "$log" >&2
		rm -f "$log"
		return 1
	fi
	rm -f "$log"
	return 0
}
