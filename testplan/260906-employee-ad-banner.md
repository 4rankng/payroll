# Testplan — Employee Ad Banner (2026-09-06)

Plan: `plans/260906-1428-employee-ad-banner/` (spec.md = authoritative design).
Written BEFORE any verification, per repo rule. Every check below must be executed
and recorded before the feature is called done.

## 1. Backend unit — domain (`internal/domain/ad_banner_test.go`)

| # | Case | Expected |
|---|------|----------|
| D1 | `IsLiveAt` before `starts_at` | false |
| D2 | `IsLiveAt` exactly at `starts_at` | true (closed start) |
| D3 | `IsLiveAt` exactly at `ends_at` | false (open end — half-open interval) |
| D4 | `IsLiveAt` while `is_active=false` | false |
| D5 | `TargetsProject` with empty target list | true (broadcast) |
| D6 | `TargetsProject` matching / non-matching id | true / false |
| D7 | `Validate`: `ends_at <= starts_at` | error |
| D8 | `Validate`: window > 180 days | error |
| D9 | `Validate`: 7 bullets | error (max 6) |
| D10 | `Validate`: 4 CTAs | error (max 3) |
| D11 | `Validate`: CTA type not phone/url | error |
| D12 | `Validate`: phone value with non-digits | error |
| D13 | `Validate`: url value not https:// | error |
| D14 | `Validate`: duplicate/zero target project ids | error |
| D15 | `Validate`: LGD-shaped payload (title+body+3 bullets+2 CTAs+30d window) | passes |

## 2. Backend unit — service (`app/services/ad_banner/service_test.go`, mock repo + fake cache)

| # | Case | Expected |
|---|------|----------|
| S1 | Two live campaigns, priorities 10 vs 5 | priority 10 wins |
| S2 | Equal priority, different `created_at` | newest wins |
| S3 | Campaign live but targeting another project | excluded |
| S4 | Broadcast campaign + employee in any project | shown |
| S5 | Employee with no active project assignment | nil (no broadcast fallback) |
| S6 | Campaign whose `ends_at` just passed | excluded |
| S7 | After create/update/delete | `InvalidatePattern("ad_banners:*")` called AFTER success, never before |
| S8 | `RecordCTAClick` service error | handler still succeeds (swallowed) |

## 3. Backend gates

- `cd backend && go build ./...`
- `cd backend && go test ./... -race -cover`
- Migration: `go run cmd/migrate/main.go up` applies 106 cleanly once; `.down` drops both tables.

## 4. Integration flow (`tests/integration/flow_ad_banner.go`, live backend)

| # | Step | Expected |
|---|------|----------|
| I1 | Admin POST `/api/v1/ad-banners` (target project A, 30d) | 201 + payload echoed |
| I2 | Targeted employee GET `/api/v1/me/ad-banner` | campaign returned |
| I3 | Employee of project B GET | `null` |
| I4 | Employee POST `/api/v1/me/ad-banner/:id/click` `{cta_index:0}` twice | both 2xx; one row per (banner, employee, cta) — idempotent |
| I5 | Employee A POST click for B's id | still recorded for A's own identity only (id from token) |
| I6 | Admin GET `/api/v1/ad-banners` | list shows click count ≥1 |
| I7 | Create campaign with `ends_at` in the past → employee GET | `null` |
| I8 | Unauthenticated GET `/api/v1/me/ad-banner` | 401 |
| I9 | Partner token on `/api/v1/ad-banners` | 403 (admin-only) |
| I10 | Full suite `make api-test` (from `backend/`) | all flows green, no regressions |

## 5. Frontend unit (vitest, colocated)

| # | Case | Expected |
|---|------|----------|
| F1 | `EmployeeAdSheet` renders title/body/bullets/CTAs from payload | visible |
| F2 | `EmployeeAdBanner` fresh (no localStorage state) | sheet auto-opens |
| F3 | Dismiss sheet | state `card`; card visible, sheet closed |
| F4 | Dismiss card | state `hidden`; nothing rendered |
| F5 | Same banner id, new `updated_at` (version bump) | sheet re-opens once |
| F6 | phone CTA click | fires mutation, then `tel:` navigation attempted |
| F7 | url CTA click | `window.open(value, "_blank", "noopener,noreferrer")` |
| F8 | `settings-page-parity.test.tsx` | green on desktop + mobile with ads tab |

## 6. Frontend gates

- `cd frontend && pnpm vitest run`
- `cd frontend && npx tsc -p tsconfig.app.json --noEmit` (real gate)
- `cd frontend && pnpm lint`

## 7. Manual UI matrix (dev, then demo, then prod)

Seed: admin creates campaign "TING TING … LG Display" targeting LGD project,
bullets ×3, CTAs: `Gọi hotline` (phone 0914827988), `Nhóm Zalo` (url https://zalo.me/g/…), 30 ngày.

| # | Surface | Action | Expected |
|---|---------|--------|----------|
| M1 | Admin → Settings → Quảng cáo | open tab (desktop + mobile viewport) | tab list legible, 6×col-span-2 grid intact |
| M2 | Composer | type content, select 1 project, preset 14 ngày | live "Kết thúc: <date>" updates; preview (375px) shows the real sheet |
| M3 | Composer | submit with >180d custom window | client blocks; server would reject (double line of defense) |
| M4 | Campaign list | after create | chip "Còn N ngày", project name shown |
| M5 | EmployeePage (regular) login | targeted employee | sheet opens once with full copy |
| M6 | Sheet | dismiss | compact card at top of feed |
| M7 | Card | tap Zalo CTA | new tab opens; admin list click count +1 |
| M8 | Card | dismiss (×) | gone; reload → still gone |
| M9 | Admin | edit campaign body | employee reload → sheet re-opens once (republish) |
| M10 | Admin | pause campaign (active off) | employee reload → nothing |
| M11 | DB/clock | set/await `ends_at` passed (or create short window in dev) | resolves for nobody, chip "Hết hạn", greyed |
| M12 | FlexiblePayEmployeePage login | targeted flexpay employee | identical M5–M8 behavior (highest-risk omission) |
| M13 | Untargeted project employee | login both pages | no banner |
| M14 | Admin | "Gia hạn" on expired | composer prefilled, fresh window, old stats stay on old row |

## 8. Rollout checks

- Demo: mig 106 applied manually; `make demo`; M1–M13 on demo.tingting.vip.
- Prod: mig 106 manual; `make deploy` from repo root; prod tag == HEAD; babysit;
  re-run `make api-test`; create LGD campaign via admin UI (audit).
- Rollback: previous image; tables may remain (unread without code).

## 9. Out of scope

Impressions, server-side dismissal, partner access, images, push delivery — spec non-goals.
