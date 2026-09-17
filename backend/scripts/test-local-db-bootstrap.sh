#!/usr/bin/env bash
# Exercise an empty MySQL instance; never connect to an existing local or
# production database. Requires Docker and uses no published ports.
set -euo pipefail
backend_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
container="payroll-bootstrap-test-$$"
cleanup() { docker rm -fv "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker run -d --name "$container" --platform linux/amd64 \
  -e MYSQL_ROOT_PASSWORD=bootstrap-test-only \
  -e MYSQL_DATABASE=payroll_bootstrap_test \
  -e PAYROLL_LOCAL_BOOTSTRAP=1 \
  -v "$backend_dir/scripts/init-local-db.sh:/docker-entrypoint-initdb.d/00-init-local-db.sh:ro" \
  -v "$backend_dir/migrations:/payroll-migrations:ro" \
  mysql:8.0 >/dev/null

ready=false
for ((attempt=0; attempt<120; attempt++)); do
  if docker logs "$container" 2>&1 | grep -q 'MySQL init process done'; then
    ready=true
    break
  fi
  if [[ $(docker inspect --format '{{.State.Running}}' "$container") != true ]]; then
    docker logs "$container"
    exit 1
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  docker logs "$container"
  echo 'Timed out bootstrapping the isolated MySQL instance.' >&2
  exit 1
fi

query() {
  docker exec -e MYSQL_PWD=bootstrap-test-only "$container" \
    mysql -uroot -N payroll_bootstrap_test -e "$1"
}
# Wait for the final server after MySQL's temporary initialization server exits.
for ((attempt=0; attempt<30; attempt++)); do
  if query 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 1
done
[[ $(query 'SELECT COUNT(*) FROM transactions') == 0 ]]
[[ $(query 'SELECT COUNT(*) FROM ledger_entries') == 0 ]]
[[ $(query "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND ((table_name='assets' AND column_name='updated_at') OR (table_name='ledger_entries' AND column_name='settlement_id'))") == 2 ]]
query "SELECT column_type FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name='role'" | grep -q accountant

# Reproduce a populated older installation, including an installation-specific
# role. The forward migration must preserve existing rows and enum members.
query "ALTER TABLE users MODIFY role ENUM('admin','partner','employee','adv_partner','local_auditor') NOT NULL DEFAULT 'employee';
ALTER TABLE ledger_entries DROP COLUMN settlement_id;
ALTER TABLE assets DROP COLUMN updated_at;
ALTER TABLE banks ALTER COLUMN branch_code DROP DEFAULT;
INSERT INTO users (id,username,password,fullname,role) VALUES (1,'legacy-user','test-only','Legacy user','local_auditor');
INSERT INTO banks (id,branch_name,branch_code) VALUES (1,'Legacy bank','BRANCH-42');
INSERT INTO assets (id,filename,file_path,uploaded_by) VALUES (1,'legacy.xlsx','/test/legacy.xlsx',1);
INSERT INTO transactions (id,description,transaction_type,amount,party,created_by,settled_amount) VALUES (1,'Legacy transaction','Expense',123456,'Legacy party',1,45678);
INSERT INTO ledger_entries (id,date,account,party,debit,credit,balance,transaction_id,created_by) VALUES (1,'2026-01-01','Expense','Legacy party',123456,0,123456,1,1);"
snapshot_query="SELECT CONCAT(id,':',amount,':',settled_amount,':',description) FROM transactions ORDER BY id;
SELECT CONCAT(id,':',debit,':',credit,':',balance,':',transaction_id,':',party) FROM ledger_entries ORDER BY id;
SELECT CONCAT(id,':',username,':',role) FROM users ORDER BY id;
SELECT CONCAT(id,':',branch_name,':',branch_code) FROM banks ORDER BY id;
SELECT CONCAT(id,':',filename,':',file_path,':',uploaded_by) FROM assets ORDER BY id;"
before=$(query "$snapshot_query")

# Replay twice to exercise both addition and already-applied paths.
for ((attempt=0; attempt<2; attempt++)); do
  docker exec -i -e MYSQL_PWD=bootstrap-test-only "$container" \
    mysql -uroot payroll_bootstrap_test < "$backend_dir/migrations/107_reconcile_runtime_schema.up.sql"
done
[[ $(query "$snapshot_query") == "$before" ]]
[[ $(query 'SELECT COUNT(*) FROM ledger_entries WHERE settlement_id IS NULL') == 1 ]]
query "INSERT INTO users (username,password,fullname,role) VALUES ('accountant-user','test-only','Accountant','accountant');
INSERT INTO banks (branch_name) VALUES ('New bank without legacy code');"
[[ $(query "SELECT branch_code='' FROM banks WHERE branch_name='New bank without legacy code'") == 1 ]]

# A restart/re-run must not reset populated schemas.
if docker exec "$container" /docker-entrypoint-initdb.d/00-init-local-db.sh 2>/dev/null; then
  echo 'Bootstrap unexpectedly accepted a non-empty database.' >&2
  exit 1
fi
if docker exec -e PAYROLL_LOCAL_BOOTSTRAP=0 "$container" /docker-entrypoint-initdb.d/00-init-local-db.sh 2>/dev/null; then
  echo 'Bootstrap unexpectedly accepted a missing local-only guard.' >&2
  exit 1
fi
echo 'PASS: fresh schema, populated legacy rows preserved, forward-migration replay, and existing-database guards.'
