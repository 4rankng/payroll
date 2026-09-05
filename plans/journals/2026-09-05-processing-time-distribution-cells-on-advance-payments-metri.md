---
title: Processing-time distribution cells on advance-payments metrics
date: 2026-09-05
summary: "Replaced the all-or-nothing 'Can toi uu/Tot' badge (any request >30s flagged the period) on the admin advance-payments 'Thoi gian xu ly TB' card with real distribution data. Backend: 4 additive bucket fields (completed30sTo2m/2mTo5m/5mTo15m/Over15m) threaded repo SQL -> domain -> DTO -> both mappers. UI per user direction: three compact cells (<30 giay / <5 phut / >5 phut, cells 2-3 sum two buckets each) instead of a 5-bar chart that wasted vertical space. Verified: 201/201 scoped go tests, 488/488 vitest, live-DB bucket partition proof (297==297), app-tsconfig clean on touched file. Code review caught: footer prop declared required but omitted on the new card (TS2741 masked by the repo's no-op tsc lint step - root tsconfig files:[] compiles zero files). Fixed with optional footer + conditional render. 4 prod deploys same day (d38eea82, 3e7af58b, 45a1fcba, footer fix), each verified tag==HEAD + containers healthy."
---

# Processing-time distribution cells on advance-payments metrics

Replaced the all-or-nothing 'Can toi uu/Tot' badge (any request >30s flagged the period) on the admin advance-payments 'Thoi gian xu ly TB' card with real distribution data. Backend: 4 additive bucket fields (completed30sTo2m/2mTo5m/5mTo15m/Over15m) threaded repo SQL -> domain -> DTO -> both mappers. UI per user direction: three compact cells (<30 giay / <5 phut / >5 phut, cells 2-3 sum two buckets each) instead of a 5-bar chart that wasted vertical space. Verified: 201/201 scoped go tests, 488/488 vitest, live-DB bucket partition proof (297==297), app-tsconfig clean on touched file. Code review caught: footer prop declared required but omitted on the new card (TS2741 masked by the repo's no-op tsc lint step - root tsconfig files:[] compiles zero files). Fixed with optional footer + conditional render. 4 prod deploys same day (d38eea82, 3e7af58b, 45a1fcba, footer fix), each verified tag==HEAD + containers healthy.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
