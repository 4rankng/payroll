#!/usr/bin/env bash
# Resolve the GHCR push token for `make push` / `make deploy` in this repo.
#
# Resolution order (each candidate token is validated against ghcr.io before
# use — a rejected token falls through to the next source):
#   1. $GHCR_TOKEN from the environment (CI / manual override). NOTE: this may
#      hold the known-dead token exported from ~/.zshrc in interactive shells;
#      it is validated and skipped when ghcr.io rejects it.
#   2. The ghcr.io credential Docker already stores (macOS keychain via the
#      credsStore helper in ~/.docker/config.json) — refreshed by any
#      `docker login ghcr.io`.
#   3. Legacy `export GHCR_TOKEN=` in ~/.zshrc (dead since the 2026-08 token
#      revocation; reachable only in shells that never sourced it).
#
# Prints the token on stdout. Diagnostics and errors go to stderr only.
# The token is never echoed back in an error message.
#
# Usage: ghcr-token.sh [ghcr-package-name]   (default: payroll-backend)
#
# Callers run this at RECIPE time (not `$(shell)` parse time) so that a
# failure surfaces its stderr and stops the make target, instead of silently
# expanding to an empty token and failing later as a cryptic
# "Error response from daemon: ... denied: denied" at docker login.
set -euo pipefail

PACKAGE="${1:-payroll-backend}"
OWNER="${GHCR_OWNER:-4rankng}"
REGISTRY="https://ghcr.io"

# --- helpers ----------------------------------------------------------------

# Validate a candidate token against ghcr.io.
# Exit 0 = valid, 1 = definitively rejected (401/403), 2 = unknown (network).
token_status() {
	local candidate="$1" body http_code
	body=$(curl -s --max-time 10 -w '\n%{http_code}' \
		-u "$OWNER:$candidate" \
		"$REGISTRY/token?service=ghcr.io&scope=repository:$OWNER/$PACKAGE:pull" 2>/dev/null) || return 2
	http_code="${body##*$'\n'}"
	body="${body%$'\n'*}"
	if [[ "$http_code" == "200" && $(jq -e '.token != null' <<<"$body" >/dev/null 2>&1; echo $?) == "0" ]]; then
		return 0
	fi
	if [[ "$http_code" == "401" || "$http_code" == "403" ]]; then
		return 1
	fi
	return 2
}

die() {
	echo "ERROR: $*" >&2
	exit 1
}

# try_token <candidate> <source-label>: validate and emit on success.
# Falls through (returns 1) when ghcr.io definitively rejects the token.
# Emits unvalidated on network-unknown (with a warning) — a registry blip
# must not kill a deploy of a known-good token.
try_token() {
	local candidate="$1" source="$2" status=0
	token_status "$candidate" || status=$?
	case $status in
	0)
		printf '%s' "$candidate"
		exit 0
		;;
	1)
		echo "WARN: $source was rejected by ghcr.io — skipping it." >&2
		return 1
		;;
	*)
		echo "WARN: could not reach ghcr.io to pre-validate $source; using it unvalidated." >&2
		printf '%s' "$candidate"
		exit 0
		;;
	esac
}

# --- resolve ----------------------------------------------------------------

# 1. Environment token — validated; the dead ~/.zshrc export falls through.
if [[ -n "${GHCR_TOKEN:-}" ]]; then
	try_token "$GHCR_TOKEN" '$GHCR_TOKEN from the environment' || true
fi

# 2. Docker's stored ghcr.io credential. Read with retries: a concurrent
#    `docker login` (e.g. the other image's push in the same deploy) briefly
#    erases the keychain entry, which must not drop us to a worse token.
helper="desktop"
if [[ -f "$HOME/.docker/config.json" ]]; then
	stored_helper=$(jq -r '.credsStore // empty' "$HOME/.docker/config.json" 2>/dev/null || true)
	[[ -n "$stored_helper" ]] && helper="$stored_helper"
fi
secret=""
if command -v "docker-credential-$helper" >/dev/null 2>&1; then
	for attempt in 1 2 3; do
		secret=$(printf '%s' "$REGISTRY" | "docker-credential-$helper" get 2>/dev/null | jq -r '.Secret // empty' 2>/dev/null || true)
		[[ -n "$secret" ]] && break
		[[ $attempt -eq 1 ]] && echo "keychain credential not found on first read (possible concurrent docker login) — retrying" >&2
		sleep 1
		[ $attempt -eq 3 ] && break
	done
fi
if [[ -n "$secret" ]]; then
	try_token "$secret" "Docker's stored ghcr.io credential (keychain)" || \
		die "Docker's stored ghcr.io token for $OWNER was rejected by ghcr.io.
Fix: run 'docker login ghcr.io -u $OWNER' to refresh it, then retry make deploy.
     (or: export GHCR_TOKEN=<fresh GitHub PAT with write:packages>)"
fi

# 3. Legacy ~/.zshrc export (dead since 2026-08; only reachable in shells that
#    never source it — interactive shells carry it in the env and hit step 1).
legacy=$(awk -F= '/export GHCR_TOKEN=/{print $2}' "$HOME/.zshrc" 2>/dev/null | head -1 || true)
if [[ -n "$legacy" ]]; then
	try_token "$legacy" "the legacy GHCR_TOKEN exported in ~/.zshrc" || \
		die "GHCR_TOKEN exported in ~/.zshrc was rejected by ghcr.io (known-dead since the 2026-08 revocation).
Fix: run 'docker login ghcr.io -u $OWNER' so Docker stores a fresh credential, then retry make deploy.
     (or: export GHCR_TOKEN=<fresh GitHub PAT with write:packages>)"
fi

die "no usable ghcr.io credential found for $OWNER.
Fix: run 'docker login ghcr.io -u $OWNER' (stored in the macOS keychain and picked up automatically on every deploy).
     (or: export GHCR_TOKEN=<GitHub PAT with write:packages>)"
