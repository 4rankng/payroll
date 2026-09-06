---
title: "Employee ad banner: project-targeted campaigns end to end"
date: 2026-09-06
summary: "6-commit feature: ad_banners + click ledger (mig 106), domain predicates with 180-day cap, /me/ad-banner resolve with 60s cache, admin Quảng cáo tab on both settings pages, three-stage localStorage dismissal. Verification found three real bugs: GORM mangling CTAs into ct_as (INSERT-only failure; reads survived via SELECT *), per-IP rate limiters sharing one Redis counter (global API limiter tripped the click cap; fixed by per-account keying), and the frontend service double-unwrapping the ApiResponse envelope (create reported success as failure; employee resolve always null). api-test 315/292-pass/0-fail; E2E on localhost:3000 verified sheet→card→CTA-count→version-bump against the real LGD project (id 58) with employee sinhnd."
---

# Employee ad banner: project-targeted campaigns end to end

6-commit feature: ad_banners + click ledger (mig 106), domain predicates with 180-day cap, /me/ad-banner resolve with 60s cache, admin Quảng cáo tab on both settings pages, three-stage localStorage dismissal. Verification found three real bugs: GORM mangling CTAs into ct_as (INSERT-only failure; reads survived via SELECT *), per-IP rate limiters sharing one Redis counter (global API limiter tripped the click cap; fixed by per-account keying), and the frontend service double-unwrapping the ApiResponse envelope (create reported success as failure; employee resolve always null). api-test 315/292-pass/0-fail; E2E on localhost:3000 verified sheet→card→CTA-count→version-bump against the real LGD project (id 58) with employee sinhnd.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
