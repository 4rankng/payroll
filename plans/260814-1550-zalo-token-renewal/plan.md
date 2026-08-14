---
title: Durable Zalo token renewal
status: completed
priority: P0
effort: medium
branch: main
tags: [backend, zalo, oauth, redis]
created: 2026-08-14
---

# Durable Zalo Token Renewal

## Overview

Make Payroll validate a pasted refresh token immediately and keep Zalo's rotating token chain authoritative across concurrent requests and API instances.

## Phases

| Phase | Status | Detail |
|---|---|---|
| 01 | Completed | [Token authority and persistence](phase-01-token-authority.md) |
| 02 | Completed | [Verification and operational recovery](phase-02-verification.md) |

## Dependencies

- Existing `zaloconnect.Service` settings-backed credential source
- Existing `zalo.Provider` refresh and `-124` retry flow
- Existing production Redis client
- Zalo OAuth v4 token endpoint and rotating refresh-token contract

## Success Criteria

- Saving a supplied refresh token exchanges it immediately; invalid input is rejected before the connection is reported healthy.
- A successful exchange persists the returned access token, returned-or-retained refresh token, and expiry before any send uses it.
- Under healthy Redis coordination, concurrent refreshes across Provider instances spend one refresh token once; waiters re-read and reuse the winner's credentials.
- Concurrent metadata/error updates cannot overwrite a newer rotated token pair.
- Existing admin API payloads, ZNS send behavior, and one-retry `-124` behavior remain compatible.
- Bound Zalo credentials never appear in SQL logs.
- `-14014` is handled as an invalid refresh-token condition; temporary OAuth
  transport failures remain retryable and do not by themselves require a
  credential replacement.
- Focused race tests, affected package tests, vet, and repository-required broader checks are reported honestly.

## Scope Boundary

- No Zalo settings UI redesign.
- No credential sharing with ChatBot; Payroll remains bound to its own Zalo App ID.
- No commit or production deployment without explicit authorization.
- After any authorized production deployment, an administrator must still
  supply and validate one fresh Payroll-specific access/refresh pair; no plan
  artifact is evidence that this live action has occurred.

## Known External Boundary

Zalo's token exchange and Payroll's database write cannot be one transaction.
A process crash, ambiguous upstream timeout, or database failure after Zalo
consumes the refresh token but before the successor is stored still requires a
fresh pair. Redis lease renewal and CAS close application concurrency paths;
they cannot make an external OAuth exchange transactionally atomic.
