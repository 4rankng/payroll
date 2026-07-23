#!/usr/bin/env bash
# Sealed wrapper: runs the node Lighthouse benchmark for the self-improve loop.
# Do NOT modify — the improvement loop ranks on this script's output.
set -euo pipefail
exec node "$(dirname "$0")/lighthouse-benchmark.mjs" "$@"
