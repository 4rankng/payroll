---
title: "Backend: filename search + project ordering"
status: todo
priority: P1
effort: "3h"
dependencies: []
---

# Phase 1: Backend: filename search + project ordering

## Overview

Add two additive capabilities to the partner-import list path: substring search
on the stored original file name, and project-grouped ordering. No route, DTO,
or schema changes.

## Requirements

- Functional: `GET /timesheets/partner-import?search=<term>` returns only
  uploads whose `metadata.$.original_name` contains the term, **case- AND
  accent-insensitive** (`georim` matches `GEORIM-…xlsx`; `bang cong` matches
  `Bảng công.xlsx`). Case-insensitivity CANNOT come from the table collation:
  `assets.metadata` is a native JSON column and `JSON_UNQUOTE(JSON_EXTRACT(…))`
  results carry `utf8mb4_bin` regardless — the comparison must apply an
  explicit `COLLATE utf8mb4_0900_ai_ci`.
- Functional: `GET /timesheets/partner-import?sort=project` orders by
  `metadata.$.project_id` ASC (NULLs excluded), `created_at` DESC.
- Functional: rows with NULL or unparseable metadata are excluded by this
  endpoint's filters (the renderer drops them via `assetToImportItem` → nil;
  without the filter, Count totals count invisible rows and NULLs sort first
  under `sort=project`, corrupting page 1).
- Non-functional: LIKE wildcards in user input (`%`, `_`, `\`) must be escaped
  as literals; the order expression is a fixed SQL string (no user input in
  ORDER BY).
- Compatibility: omitting both params yields behavior identical to today
  (aside from the NULL-metadata exclusion).

## Architecture

Handler parses query params → `domain.AssetFilters` gains two fields →
`applyAssetFilters` / `List` translate them to SQL. Same call path as the
existing `MetadataQuery` flow.

## Related Code Files

- Modify: `backend/internal/domain/asset.go` (AssetFilters)
- Modify: `backend/internal/infra/persistence/asset_repository.go`
  (applyAssetFilters, List ordering)
- Modify: `backend/internal/transport/http/handlers/timesheet/bcc_import_handler.go`
  (ListPartnerImports params)

## Implementation Steps

1. `domain/asset.go` — extend `AssetFilters`:
   ```go
   MetadataLike   map[string]string // JSON path key → substring term (LIKE %term%)
   GroupByProject bool              // order by metadata project_id, then created_at DESC
   MetadataNotNull bool             // require metadata IS NOT NULL (renderer drops NULL rows)
   ```
   `applyAssetFilters` applies `query.Where("metadata IS NOT NULL")` when set.

2. `asset_repository.go` `applyAssetFilters` — after the `MetadataQuery` loop:
   ```go
   for key, term := range filters.MetadataLike {
       query = query.Where(
           "JSON_UNQUOTE(JSON_EXTRACT(metadata, ?)) COLLATE utf8mb4_0900_ai_ci LIKE ? ESCAPE '\\\\'",
           fmt.Sprintf("$.%s", key), "%"+escapeLike(term)+"%")
   }
   ```
   No existing LIKE-escape helper exists (verified: grep `ESCAPE|escapeLike`
   in backend/internal → 0 hits). Add a package-local helper with an EXACT
   contract:
   - Escape in this order: `\` first, then `%`, then `_` (wrong order turns
     `100\%` into literal-backslash + live wildcard — a silent filter bypass).
   - Emit SINGLE backslashes — the pattern travels as a bound `?` parameter,
     never as a SQL literal.
   - SQL text `ESCAPE '\\'` requires `"ESCAPE '\\\\'"` in Go source; one
     backslash fewer is a SQL syntax error on every search.

3. `asset_repository.go` `List` — ordering must REPLACE the default clause,
   not be added alongside it (routing the expression through
   `filters.SortBy` fails `SanitizeSortColumn`'s identifier regex and
   SILENTLY reverts to `created_at DESC`):
   ```go
   if filters.GroupByProject {
       query = query.Order("JSON_EXTRACT(metadata, '$.project_id') ASC, created_at DESC")
   } else {
       // existing sanitize-whitelist ordering, unchanged
   }
   ```

4. `bcc_import_handler.go` `ListPartnerImports`:
   ```go
   filters.MetadataQuery = make(map[string]string)
   filters.MetadataNotNull = true // exclude NULL/unparseable rows the renderer drops
   ...
   if search := strings.TrimSpace(c.Query("search")); search != "" {
       if utf8.RuneCountInString(search) > 100 {
           response.BadRequest(c, "search không được vượt quá 100 ký tự")
           return
       }
       if filters.MetadataLike == nil {
           filters.MetadataLike = make(map[string]string)
       }
       filters.MetadataLike["original_name"] = search
   }
   filters.GroupByProject = c.Query("sort") == "project"
   ```
   Cap is `utf8.RuneCountInString` (Go `len()` counts bytes; Vietnamese
   diacritics are 3 bytes/char and the message promises "ký tự").

5. Tests — the repo's persistence harness is SQLite in-memory, which cannot
   execute `JSON_UNQUOTE` (MySQL-only). Do NOT attempt a gorm repo test for
   the new SQL; it cannot exercise the real predicate. Coverage is:
   - Unit test the escape helper (pure Go, runs anywhere): fixture containing
     all three metacharacters mixed (`a\b%c_d`) asserting the one exact
     escaped output string.
   - Behavioral dev-curl gate (see Success Criteria) — this is the real test
     for the SQL, executed against the dev MySQL.

## Success Criteria

- [x] `go build ./...` passes; escape-helper unit test passes with exact-output
      assertion on `a\b%c_d`.
- [x] Dev curl: `?search=georim` (lowercase) matches `GEORIM-…xlsx` — proven
      over live HTTP (7/7 rows, totals 173→7). Accent branch had no `Bảng
      công` fixture in dev data (0 rows both levels); accent-insensitivity is
      a property of the `utf8mb4_0900_ai_ci` collation itself.
- [x] Dev curl: `?search=100%25` and `?search=data_8` match literal names only
      (no wildcard expansion; SQL-level check confirmed `100\%` did not
      expand to the 173-row set).
- [x] Dev curl: `?sort=project` returns rows grouped by project, newest-first
      within each project (project 7 block observed: 08.2026 → 07.2026 →
      06.2026); baseline request unchanged.
- [x] Boundary: 100-char search → 200; 101 chars → 400 (live HTTP).

## Risk Assessment

- Accent-insensitivity is now by design (`ai` collation) — if exact-diacritic
  matching is ever required, switch the COLLATE to a `_0900_ci` variant.
  Signal: user reports a search matching "too loosely".
- Renamed/legacy rows whose metadata is NULL no longer appear in this list at
  all — consistent with what the UI could render for them (nothing).
