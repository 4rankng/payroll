#!/usr/bin/env sh
# Top up the mock OnePay balance.
#   Usage: ./topup.sh [amount] [host]
#   amount  VND to add (default 100,000,000)
#   host    mock base URL (default http://localhost:9001)
#
# The mock serves POST /admin/topup/1pay?amount=N — see handler.go:topupBalance.
set -eu

AMOUNT="${1:-100000000}"
HOST="${2:-http://localhost:9001}"

curl -fsS -X POST "$HOST/admin/topup/1pay?amount=$AMOUNT"
echo
