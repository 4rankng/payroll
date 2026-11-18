#!/usr/bin/env bash
# Restore a mysqldump backup into the local payroll-mysql container.
#
# Authenticates as the application user (MYSQL_USER/MYSQL_PASSWORD), not root:
# the local container's root password is not managed by this repo, while the
# app user holds ALL PRIVILEGES on payroll_db.* — enough to drop and recreate
# every table inside the database. The dump is table-level (DROP TABLE IF
# EXISTS + CREATE TABLE, no CREATE DATABASE), so the database itself is never
# dropped.
set -euo pipefail

container=${MYSQL_CONTAINER:-payroll-mysql}
database=${MYSQL_DATABASE:-payroll_db}
backup=$1

if [[ ! -f "$backup" ]]; then
  echo "❌ Backup file not found: $backup" >&2
  exit 1
fi

mysql_exec() {
  docker exec -i "$container" sh -c 'exec mysql -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$@"' _ "$@"
}

# Ensure the container exists and is running; start it and wait for MySQL
# to accept connections if it was stopped (e.g. Docker Desktop restart).
container_state=$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || echo "missing")
if [[ "$container_state" == "missing" ]]; then
  echo "❌ Container '$container' not found. Create it first: docker compose -f docker-compose.dev.yml up -d mysql" >&2
  exit 1
fi
if [[ "$container_state" != "true" ]]; then
  echo "🚀 Container '$container' is stopped — starting it..."
  docker start "$container" >/dev/null
  echo -n "⏳ Waiting for MySQL to accept connections"
  for _ in $(seq 1 60); do
    if docker exec "$container" sh -c 'mysql -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1"' >/dev/null 2>&1; then
      echo " ✅"
      break
    fi
    echo -n "."
    sleep 1
  done
  if ! docker exec "$container" sh -c 'mysql -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1"' >/dev/null 2>&1; then
    echo "" >&2
    echo "❌ MySQL in '$container' did not become ready within 60s." >&2
    exit 1
  fi
fi

echo "[$(date +%H:%M:%S)] 🗑️  Dropping existing tables in $database (this can take a few minutes)..."
docker exec "$container" sh -c '
  mysql -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -N -e "
    SELECT CONCAT('"'"'DROP TABLE IF EXISTS '"'"', table_name, '"'"';'"'"')
    FROM information_schema.tables
    WHERE table_schema = '"'"'"$1"'"'"'
  " | {
    echo "SET FOREIGN_KEY_CHECKS=0;"
    cat
    echo "SET FOREIGN_KEY_CHECKS=1;"
  } | mysql -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$1"
' _ "$database"

echo "[$(date +%H:%M:%S)] ⏳ Decompressing and restoring to $database..."
gunzip -c "$backup" | mysql_exec "$database"

table_count=$(docker exec "$container" sh -c \
  'mysql -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '"'"'"$1"'"'"'"' \
  _ "$database")
echo "[$(date +%H:%M:%S)] ✅ Restored $table_count tables into $database"
