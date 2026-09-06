---
phase: 1
title: "Backend vertical slice"
status: pending
priority: P1
effort: "1d"
dependencies: []
---

# Phase 1: Backend vertical slice

## Overview
Migration + domain + repository + service + handler + routes + DI + Casbin for the ad
banner feature, per spec §Data Model / §Domain Layer / §Application Service / §HTTP API.

## Requirements
- Functional: admin CRUD on `/api/v1/ad-banners`; employee `GET /api/v1/me/ad-banner`
  resolving the single winning campaign; `POST /api/v1/me/ad-banner/:id/click`
  recording a click with the employee id from auth context; 60s cache invalidated
  after successful mutation.
- Non-functional: ADR-001 (no frameworks in domain), ADR-007, `clock.Now()` only,
  no CHECK constraints, audit events for admin CRUD, rate limit on click POST.

## Related Code Files
- Create: `backend/migrations/106_add_ad_banners.{up,down}.sql`,
  `backend/internal/domain/ad_banner.go`, `ad_banner_test.go`,
  `event_factory_ad_banner.go`, `backend/internal/infra/persistence/ad_banner_repository.go`,
  `backend/internal/app/services/ad_banner/service.go`, `service_test.go`,
  `backend/internal/app/dto/ad_banner.go`,
  `backend/internal/transport/http/handlers/ad_banner.go`,
  `backend/internal/app/bootstrap/routes_ad_banner.go`
- Modify: `backend/internal/domain/events.go` (structs + `EntityTypeAdBanner`),
  `backend/internal/app/bootstrap/repositories/init.go`, `.../services/init.go`,
  `.../container.go`, `.../routes.go` (call `setupAdBannerRoutes`),
  `backend/configs/casbin_policy.csv` (ONE line: employee POST `/api/v1/me/ad-banner/*`)

## Implementation Steps
1. Migration 106: `ad_banners` + `ad_banner_cta_clicks` per spec; index
   `(is_active, starts_at, ends_at)` and clicks `(banner_id)`.
2. Domain entity + port + predicates (`IsLiveAt` half-open, `TargetsProject`
   empty=broadcast) + `Validate()` (incl. 180-day ceiling, CTA shapes) + unit tests.
3. Event factory + events.go structs.
4. GORM repository (`ListLiveAt` window predicate in SQL matching `IsLiveAt`;
   `RecordCTAClick` append-only; `CTAClickCounts` GROUP BY).
5. Service: `ResolveForEmployee` (reuse `GetActiveProjectsForEmployee`,
   `domain/project_employee.go:93`; order `priority DESC, created_at DESC`; nil when
   no active project), admin CRUD + audit events, cache `ad_banners:*`.
6. DTOs, handler (click handler logs+swallows service errors), routes
   (copy `routes_advance_payment.go`; explicit `.Use(Authorize())` on admin group),
   DI registration ×4, Casbin line.

## Success Criteria
- [ ] `cd backend && go build ./... && go test ./... -race -cover` green
- [ ] Domain tests cover both window edges, targeting cases, every Validate rule
- [ ] Service tests cover precedence, expiry, wrong-project, no-assignment→nil,
      broadcast, invalidate-after-success

## Risk Assessment
JSON serializer columns must match the `projects.shift_names` precedent exactly
(`serializer:json`), else GORM scans fail at runtime — covered by repository-level
integration in Phase 2. If `GetActiveProjectsForEmployee`'s `project_status='running'`
filter proves too narrow for real data, the fallback is a direct `last_date IS NULL`
query in the repo — decided only if the testplan matrix shows a miss.
