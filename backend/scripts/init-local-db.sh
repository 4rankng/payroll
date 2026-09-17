#!/usr/bin/env bash
# Fresh local databases only. Historical files include down migrations and
# production data repairs, so never mount migrations/ as MySQL's init folder.
set -euo pipefail

if [[ "${PAYROLL_LOCAL_BOOTSTRAP:-}" != "1" ]]; then
  echo 'Refusing bootstrap: PAYROLL_LOCAL_BOOTSTRAP=1 is required.' >&2
  exit 1
fi
database=${MYSQL_DATABASE:-payroll_db}
migrations=${PAYROLL_MIGRATIONS_DIR:-/payroll-migrations}
export MYSQL_PWD=${MYSQL_ROOT_PASSWORD:?MYSQL_ROOT_PASSWORD is required}
mysql_cmd=(mysql --protocol=socket -uroot --database="$database")
table_count=$("${mysql_cmd[@]}" -N -e 'SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()')
if [[ "$table_count" != "0" ]]; then
  echo "Refusing bootstrap: $database already has tables; existing data is never reset." >&2
  exit 1
fi

# These historical index cleanups refer to indexes that were absent in some
# installations. Tolerate only the named, understood index errors; everything
# else stops initialization. Never use a blanket --force for schema changes.
apply_index_cleanup() {
  local file=$1 allowed=$2 output
  output=$(mktemp)
  if ! "${mysql_cmd[@]}" --force < "$file" > "$output" 2>&1; then
    cat "$output" >&2
    rm -f "$output"
    return 1
  fi
  cat "$output"
  if grep '^ERROR ' "$output" | grep -Ev "^ERROR ($allowed) "; then
    rm -f "$output"
    return 1
  fi
  rm -f "$output"
}

for file in "$migrations"/*.up.sql; do
  name=${file##*/}
  echo "Local bootstrap: $name"
  case "$name" in
    016_remove_outbox_event_id_from_transactions.up.sql)
      # 001 never contained this obsolete column.
      ;;
    017_fix_missing_bulk_transfer_transaction.up.sql)
      echo 'Skipping production-specific transaction repair on an empty local database.'
      ;;
    024_optimize_api_metrics_performance.up.sql)
      # Original DROP INDEX statements omit their table. Preserve the intended
      # cleanup locally, without rewriting the historical production file.
      corrected=$(mktemp)
      sed -E 's/^(DROP INDEX [^;]+);/\1 ON api_metrics;/' "$file" > "$corrected"
      apply_index_cleanup "$corrected" '1091|1061'
      rm -f "$corrected"
      ;;
    025_cleanup_duplicate_and_unused_indexes.up.sql)
      apply_index_cleanup "$file" '1091|1553'
      ;;
    057_wallet_provider.up.sql)
      # 054 already calls its invoice index uk_wp_provider_invoice_no.
      apply_index_cleanup "$file" '1091'
      ;;
    103_split_loan_schedule_principal_and_interest.up.sql)
      # 103 assumes settlement linkage existed in the deployed schema. The
      # forward compatibility migration supplies the omitted schema first.
      "${mysql_cmd[@]}" < "$migrations/107_reconcile_runtime_schema.up.sql"
      "${mysql_cmd[@]}" < "$file"
      ;;
    *) "${mysql_cmd[@]}" < "$file" ;;
  esac
done
