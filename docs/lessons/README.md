# Lessons Learned

Durable engineering knowledge extracted from real experiences. Each lesson captures a decision, its context, and the takeaway — so the same mistake is not repeated and the same insight is not rediscovered.

## Relationship to Journals

- **[Journals](../journals/)** — Session narratives: what happened, what was tried, what shipped. Written during/after implementation.
- **Lessons** (this directory) — Durable takeaways: the generalized insight extracted from a journal or incident. Written when a lesson is clear enough to apply to future work.

A journal may produce zero or one lessons. A lesson may draw from multiple journals.

## Format

```
# [Topic]

**Date:** YYYY-MM-DD
**Source:** Commit hash, journal, or incident
**Tags:** [area, area, ...]

## Context
What happened. What problem was encountered.

## Decision / Outcome
What was decided or what resulted.

## Lesson
The generalized takeaway. What should be done differently (or the same) next time.

## References
Links to journals, commits, ADRs, or code.
```

## Index

| Lesson | Date | Topic |
|--------|------|-------|
| [Statistical forecast over ML](2026-07-11-statistical-forecast-over-ml.md) | 2026-07-11 | Simple statistical methods beat ML for short-horizon, near-constant series |
| [Google OAuth no-OTP residual risk](2026-07-04-google-oauth-no-otp-residual-risk.md) | 2026-07-04 | OAuth no-OTP convenience trade-off and replay/issuer hardening |
| [Security: IDOR, spray, token revocation](2026-07-04-security-idor-spray-token-revocation.md) | 2026-07-04 | Red-team findings: ownership validation, rate limiting, token blacklist |
| [DB performance: N+1 elimination](2026-07-04-db-performance-n-plus-1-elimination.md) | 2026-07-04 | N+1 elimination via projections, NOT EXISTS, pagination, indexes |

## When to Add a Lesson

- A bug fix reveals a pattern that could recur.
- An architectural decision has non-obvious trade-offs worth documenting.
- A performance optimization technique proves effective.
- A security issue is found and fixed.
- A testing approach saves significant time.

You do **not** need a lesson for:
- Routine feature implementation.
- Bug fixes with obvious causes.
- Cosmetic changes.
