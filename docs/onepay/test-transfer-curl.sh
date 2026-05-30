#!/bin/bash
# OnePay funds transfer test via local backend
# Usage: bash test-transfer-curl.sh

BASE_URL="http://localhost:8080/api/v1"

# 1. Login
echo "--- LOGIN ---"
TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"frankng","password":"Admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

if [ -z "$TOKEN" ]; then
  echo "Login failed"
  exit 1
fi
echo "Token acquired"

# 2. Check account
echo ""
echo "--- CHECK ACCOUNT ---"
curl -s -X POST "$BASE_URL/admin/manual-disbursement/check-account" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bank_code":"ICBVVNVX","account_no":"103000614434"}' | python3 -m json.tool

# 3. Funds transfer
echo ""
echo "--- FUNDS TRANSFER ---"
curl -s -X POST "$BASE_URL/admin/manual-disbursement" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50000,
    "description": "test from curl",
    "bank_code": "ICBVVNVX",
    "account_no": "103000614434",
    "account_name": "Nguyen Danh Hoang"
  }' | python3 -m json.tool
