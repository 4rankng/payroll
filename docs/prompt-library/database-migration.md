# Prompt: Database Migration

Use when creating a new database migration. Safety is paramount — this is a financial system.

## Prompt

```
Goal: Create migration [NNN]_[description].up.sql — [PURPOSE]

Context to gather:
1. Read docs/database.md for migration conventions and existing tables
2. Check the latest migration number: ls backend/migrations/ | sort | tail -5
3. Read docs/decisions/ADR-007-transaction-manager-unit-of-work.md for cache rules
4. If adding a column: check that the GORM model and domain entity will be updated
5. If adding an index: run EXPLAIN on the target query to confirm it helps

Constraints:
- File naming: NNN_description.up.sql (zero-padded, sequential)
- Raw SQL only — no migration framework
- MUST be idempotent: safe to run multiple times
  - Use CREATE TABLE IF NOT EXISTS
  - Use INSERT IGNORE or ON DUPLICATE KEY UPDATE
  - Use ALTER TABLE ... ADD COLUMN IF NOT EXISTS (MySQL 8 supports this)
- No destructive operations (DROP TABLE, DROP COLUMN) without explicit team approval
- Recovery/repair scripts must be idempotent
- Apply to demo/prod BEFORE backend deploy
- Prefer code-level fixes over schema migrations when possible

Migration types:

For a new table:
  CREATE TABLE IF NOT EXISTS table_name (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    ...
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

For a new column:
  ALTER TABLE table_name ADD COLUMN IF NOT EXISTS column_name TYPE [NOT NULL] [DEFAULT value];

For a new index:
  CREATE INDEX IF NOT EXISTS index_name ON table_name (columns);

For data repair (idempotent):
  INSERT IGNORE INTO table_name (...) VALUES (...);
  -- or
  UPDATE table_name SET col = val WHERE col IS NULL;

Output:
1. The .up.sql migration file
2. List of code files that need updating (GORM model, domain entity, repository)
3. Rollback plan (how to undo if something goes wrong)
4. Deployment order (migration before or after backend deploy)

Verification:
- Migration runs without error on local dev DB
- Migration is idempotent (run twice, no error on second run)
- EXPLAIN confirms new indexes are used
- Backend code updated to match schema
- go test ./... passes after code updates
```

## Idempotency Check

After writing the migration, verify it can run twice:

```bash
# Run once
cat backend/migrations/NNN_name.up.sql | docker exec -i payroll-mysql mysql -uroot -prootpassword payroll_db

# Run again — should NOT error
cat backend/migrations/NNN_name.up.sql | docker exec -i payroll-mysql mysql -uroot -prootpassword payroll_db
```

## Deployment Order

If the migration adds columns the deployed backend code expects:
1. Apply migration to demo/prod FIRST
2. Then deploy the new backend image

If the migration removes columns the old backend code needs:
1. Deploy backend code that doesn't use the column FIRST
2. Then apply the migration
