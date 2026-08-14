---
title: "Zalo token renewal incident"
date: "2026-08-14"
component: "Zalo OA token persistence and renewal"
status: "Documented"
---

# Zalo token renewal incident

## Context

Zalo OA access tokens have a normal finite lifetime. The operational concern was not expiry itself, but a stored refresh token that could be invalid while the current access token still worked.

## What happened

An invalid refresh token remained undetected until the access token expired, turning a recoverable configuration problem into an outage at renewal time. Concurrent renewal attempts also needed one shared owner across application instances, while stale writers could otherwise restore an older token pair after a successful rotation.

## Reflection

Credential persistence is part of the renewal protocol, not a passive settings write. Validate and rotate the token pair when it is saved, before normal expiry can hide a broken refresh path. Coordinate renewal through Redis so one instance owns the exchange; use compare-and-swap with binary token comparison at the database boundary so a stale owner cannot resurrect superseded credentials. Parameterized database logs are required to keep token material out of diagnostic output.

## Decisions

- Treat access-token expiry as expected lifecycle behavior, not an error condition.
- Validate the refresh path and persist the rotated pair on save.
- Use Redis-coordinated renewal ownership and database CAS over the previously persisted token pair.
- Keep credential values out of logs through parameterized logging.

## Next

The external token exchange and database update cannot be atomic. If the process crashes after the provider rotates the pair but before persistence succeeds, the old refresh token may already be invalid. Recovery must therefore accept a fresh operator-provided token pair; it cannot be made fully automatic from the stale pair.

## Follow-up (2026-08-14, second incident)

After deploying the durable-renewal fix, saving a fresh valid pair still failed with "Không thể xác thực token Zalo". Zalo's OAuth v4 endpoint returned HTTP 200 with `expires_in` (and sometimes the error code) as a **quoted string**; the strict typed decode rejected the whole body as "malformed JSON", so validation aborted before persistence and the UI blamed the tokens.

Fix (commit 87236a3): `expires_in` decodes as advisory — an absent, invalid, non-positive, or overflowing value falls back to the conservative one-hour expiry instead of discarding a valid rotated pair — and the error code accepts number or quoted number. Lesson: when an upstream API's response typing is inconsistent, decode leniently at the boundary and treat advisory fields (like expiry) as best-effort, never as gatekeepers for accepting an otherwise valid response.
