#!/usr/bin/env bash
#
# orphan-timesheet-check.sh — detect PAID timesheets whose revenue will never be collected (MONEY LOSS).
#
# A timesheet is "at risk" when the company already paid the employee (payment_status=paid)
# but hasn't collected the receivable from the client (revenue_paid=0).
# It is "LOST" when, on top of that, it can never be auto-settled by a future sao kê:
#   - has no transaction_id (can't be linked/settled), OR
#   - the next settlement-cycle exports don't list it (its window already rolled past).
#
# Usage:
#   ./orphan-timesheet-check.sh              # fast, DB-only instant report
#   ./orphan-timesheet-check.sh --probe      # also generate next-cycle exports & classify recoverability
#   ./orphan-timesheet-check.sh --probe --atdates 2026-06-26,2026-07-02   # custom export dates
#
# Read-only. Does NOT upload or mutate anything.

set -euo pipefail

API="${API:-http://localhost:8080}"
DB_C="payroll-mysql"
ADMIN_USER="${ADMIN_USER:-frankng}"
ADMIN_PASS="${ADMIN_PASS:-Admin123}"
TOKEN_FILE="${TOKEN_FILE:-/tmp/payroll_token.txt}"
TMP="${TMP:-/tmp/orphan-check}"
mkdir -p "$TMP"

PROBE=false
CUSTOM_ATDATES=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --probe) PROBE=true; shift ;;
    --atdates) CUSTOM_ATDATES="${2:-}"; shift 2 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1 (try --help)"; shift ;;
  esac
done

C_RED=$'\033[31m'; C_YEL=$'\033[33m'; C_GRN=$'\033[32m'; C_DIM=$'\033[2m'; C_BOLD=$'\033[1m'; C_RST=$'\033[0m'

dbq() { docker exec -i "$DB_C" mysql -uroot -prootpassword payroll_db -N -B -e "$1" 2>/dev/null; }

# ---- 1. Backend alive? ----
if ! curl -s -o /dev/null -w '%{http_code}' "$API/api/v1/auth/me" 2>/dev/null | grep -qE '401|200'; then
  echo "${C_RED}✘ Backend not reachable on $API (start it first, e.g. cd backend && make dev)${C_RST}"; exit 1
fi

# ---- 2. Login (reuse cached token if still valid) ----
get_token() {
  if [[ -f "$TOKEN_FILE" ]]; then
    local code; code=$(curl -s -o /dev/null -w '%{http_code}' "$API/api/v1/auth/me" -H "Authorization: Bearer $(cat "$TOKEN_FILE")")
    if [[ "$code" == "200" ]]; then cat "$TOKEN_FILE"; return; fi
  fi
  curl -s -X POST "$API/api/v1/auth/login" -H "Content-Type: application/json" \
    -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" \
    | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])" > "$TOKEN_FILE"
  cat "$TOKEN_FILE"
}
TOKEN="$(get_token)"

# ---- 3. Instant DB report: all paid-but-unsettled timesheets ----
echo "${C_BOLD}════════════════════════════════════════════════════════════════"
echo "  ORPHAN / MONEY-AT-RISK TIMESHEET CHECK"
echo "════════════════════════════════════════════════════════════════${C_RST}"

TOTAL_AT_RISK=$(dbq "SELECT IFNULL(SUM(revenue_receivable),0) FROM timesheets WHERE payment_status='paid' AND timesheet_status='approved' AND revenue_paid=0;")
COUNT_AT_RISK=$(dbq "SELECT COUNT(*) FROM timesheets WHERE payment_status='paid' AND timesheet_status='approved' AND revenue_paid=0;")
NO_TXN=$(dbq "SELECT COUNT(*) FROM timesheets WHERE payment_status='paid' AND timesheet_status='approved' AND revenue_paid=0 AND transaction_id IS NULL;")
NO_TXN_AMT=$(dbq "SELECT IFNULL(SUM(revenue_receivable),0) FROM timesheets WHERE payment_status='paid' AND timesheet_status='approved' AND revenue_paid=0 AND transaction_id IS NULL;")

fmt_vnd() { python3 -c "import sys;print(f'{int(sys.argv[1]):,} ₫')" "$1"; }
ago_days() {  # date YYYY-MM-DD -> days old vs today
  python3 -c "import sys,datetime;print((datetime.date.today()-datetime.date.fromisoformat(sys.argv[1])).days)" "$1"; }
TODAY=$(python3 -c "import datetime;print(datetime.date.today().isoformat())")

echo
echo "${C_BOLD}TOTAL AT RISK (paid to employee, revenue NOT collected):${C_RST}"
echo "   timesheets : $COUNT_AT_RISK"
echo "   receivable : $(fmt_vnd "$TOTAL_AT_RISK")"
echo
echo "${C_BOLD}Structural LOST (no transaction_id → cannot ever be auto-settled):${C_RST}"
if [[ "$NO_TXN" -gt 0 ]]; then
  echo "   ${C_RED}✘ $NO_TXN timesheets, $(fmt_vnd "$NO_TXN_AMT") at risk${C_RST}"
else
  echo "   ${C_GRN}✓ none${C_RST}"
fi

# ---- 4. Breakdown by project × month (the working list) ----
echo
echo "${C_BOLD}─── At-risk breakdown by project × month (most recent first) ───${C_RST}"
printf '  %-26s %-7s %5s %14s %14s %s\n' "PROJECT" "MONTH" "CNT" "RECEIVABLE" "OLDEST_DAYS" "NO_TXN"
dbq "
SELECT p.name, DATE_FORMAT(t.date,'%Y-%m'), COUNT(*), SUM(t.revenue_receivable),
       MIN(t.date), SUM(CASE WHEN t.transaction_id IS NULL THEN 1 ELSE 0 END)
FROM timesheets t JOIN projects p ON p.id=t.project_id
WHERE t.payment_status='paid' AND t.timesheet_status='approved' AND t.revenue_paid=0
GROUP BY p.name, DATE_FORMAT(t.date,'%Y-%m')
ORDER BY DATE_FORMAT(t.date,'%Y-%m') DESC, SUM(t.revenue_receivable) DESC;" \
| while IFS=$'\t' read -r proj mo cnt amt oldest notxn; do
  [[ -z "$mo" ]] && continue
  age=$(ago_days "$oldest")
  flag=""
  [[ "$notxn" -gt 0 ]] && flag="${C_RED}<-- NO_TXN${C_RST}"
  [[ "$age" -ge 45 ]] && flag="$flag ${C_YEL}<-- STALE>${age}d${C_RST}"
  printf '  %-26s %-7s %5s %14s %14s %s\n' "$proj" "$mo" "$cnt" "$(fmt_vnd "$amt")" "${age}d" "$flag"
done

# ---- 5. Probe: will the next settlement cycle recover them? ----
if [[ "$PROBE" == "true" ]]; then
  echo
  echo "${C_BOLD}─── PROBE: generating next-cycle exports to classify recoverability ───${C_RST}"
  # Default atDates: next day>=24 and next day<=10 from today (the two export eligibility classes).
  if [[ -z "$CUSTOM_ATDATES" ]]; then
    read -r D_HI D_LO < <(python3 -c "
import datetime
t=datetime.date.today()
# next day>=24
hi=t.replace(day=28)
if t.day>=24: hi=(t.replace(day=1)+datetime.timedelta(days=33)).replace(day=26)
else: hi=t.replace(day=26)
# next day<=10
lo=(t.replace(day=1)+datetime.timedelta(days=33)).replace(day=2)
print(hi.isoformat(), lo.isoformat())
")
    ATDATES="$D_HI $D_LO"
  else
    ATDATES="${CUSTOM_ATDATES//,/ }"
  fi
  echo "   export atDates: ${ATDATES}"

  ALL_IDS_FILE="$TMP/all_at_risk_ids.txt"
  dbq "SELECT t.id FROM timesheets t WHERE t.payment_status='paid' AND t.timesheet_status='approved' AND t.revenue_paid=0;" > "$ALL_IDS_FILE"
  rm -f "$TMP"/exp_*.txt
  for d in $ATDATES; do
    f="$TMP/sao_ke_$d.xlsx"
    code=$(curl -s -o "$f" -w '%{http_code}' "$API/api/v1/timesheets/payroll/report?atDate=$d" -H "Authorization: Bearer $TOKEN")
    if [[ "$code" != "200" ]]; then echo "   $d: export FAILED ($code)"; continue; fi
    cnt=$(python3 -c "
import openpyxl
try:
    wb=openpyxl.load_workbook('$f',data_only=True)
    ws=wb['INTERNAL']
    ids=[int(r[0]) for i,r in enumerate(ws.iter_rows(values_only=True)) if i>0 and r[0] is not None]
    open('$TMP/exp_$d.txt','w').write('\n'.join(str(x) for x in ids)+'\n')
    print(len(ids))
except Exception:
    open('$TMP/exp_$d.txt','w').write(''); print(0)")
    echo "   $d: export OK, $cnt IDs in INTERNAL sheet"
  done

  python3 - "$ALL_IDS_FILE" "$TMP" <<'PY'
import sys, glob
at_risk=set(int(x) for x in open(sys.argv[1]).read().split() if x.strip())
covered=set()
for f in glob.glob(sys.argv[2]+'/exp_*.txt'):
    covered|=set(int(x) for x in open(f).read().split() if x.strip())
recoverable=at_risk & covered
uncovered=at_risk - covered
# "uncovered" = not listed by ANY next-cycle export → that window has rolled past it.
# (Note: no-txn timesheets already flagged as Structural LOST above — they show in exports
#  but can't be settled without linking a transaction_id.)
print()
print(f"   RECOVERABLE in next cycle : {len(recoverable)} of {len(at_risk)}")
print(f"   NOT in next-cycle export  : {len(uncovered)} of {len(at_risk)}")
print()
if uncovered:
    print("\033[31m\033[1m   ⚠️  ACTION NEEDED: %d at-risk timesheets NOT in the next settlement cycle.%s" % (len(uncovered), "\033[0m"))
    print("\033[31m   These are likely LOST money — either re-run the missed cycle's sao kê export,")
    print("   link missing transaction_ids, or collect manually. IDs:\033[0m")
    print("   " + ",".join(str(i) for i in sorted(uncovered)[:200]) + (" ..." if len(uncovered)>200 else ""))
else:
    print("\033[32m\033[1m   ✓ All at-risk timesheets are covered by the next settlement cycle.%s" % "\033[0m")
PY
else
  echo
  echo "${C_DIM}  (run with --probe to verify which orphans the next settlement cycle will recover)${C_RST}"
fi

echo
echo "${C_BOLD}════════════════════════════════════════════════════════════════${C_RST}"
echo "Report date: $TODAY   |   Run: $0 --probe  for recoverability classification"
