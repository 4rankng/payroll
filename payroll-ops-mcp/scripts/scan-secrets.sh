#!/usr/bin/env bash
# Secrets scan for the payroll-ops-mcp folder (red-team constraint 6:
# "no key material in config JSON, repo, or logs").
#
# Flags any pk_ops_* plaintext long enough to be a real key (the mint format
# is pk_ops_ + 43 base64url chars). Test fixtures deliberately use short fake
# values ("pk_ops_abcd") which the length threshold ignores.
#
# Usage: ./scripts/scan-secrets.sh [paths...]   (defaults to the repo's
#        git-tracked files under payroll-ops-mcp/ plus any MCP config JSON)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Sufficient length to be real key material; short enough to catch any paste.
PATTERN='pk_ops_[A-Za-z0-9_-]{20,}'
# Also catch mcp_* chat keys accidentally committed here (isolation rule).
PATTERN2='mcp_[A-Za-z0-9_-]{20,}'

targets=("payroll-ops-mcp" "mcp" "$@")
status=0

scan() {
  local label="$1" pattern="$2"
  local hits
  # Only git-tracked files: untracked local junk (real secrets files live
  # outside the repo, node_modules, logs) is out of scope by construction.
  hits=$(git ls-files -- "${targets[@]}" 2>/dev/null \
    | xargs grep -rEl "$pattern" 2>/dev/null || true)
  if [ -n "$hits" ]; then
    echo "❌ $label: potential key material found in tracked files:"
    echo "$hits" | sed 's/^/   /'
    echo "   Remove the key, rotate it if it was ever real, then re-run."
    status=1
  else
    echo "✅ $label: no tracked file matches $pattern"
  fi
}

scan "ops keys" "$PATTERN"
scan "chat keys" "$PATTERN2"

exit $status
