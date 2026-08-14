---
phase: 3
title: "Backup deploy and verify"
status: done
effort: ""
---

# Phase 3: Backup deploy and verify

## Overview

Commit the complete stabilized worktree, synchronize source, then perform a backup-first production deployment with independent evidence.

## Implementation Steps

1. Reconcile all plan phases and documentation, inspect the complete worktree for secrets/unrelated accidental artifacts, and obtain the final Git diff summary.
2. Use the required git-manager to stage all intended worktree changes, create one conventional commit, and push `main` to `origin` as explicitly authorized.
3. Run `make backup`, identify the new backup artifact, and validate it with `gzip -t` before deployment.
4. Run `make deploy`, which builds/pushes amd64 frontend/backend images and deploys them over SSH.
5. Verify `https://tingting.vip`, `/api/healthz`, `/api/ready`, container health/restarts, MySQL, Redis, deployed image identity/digests, scheduler timezone/job presence, and critical backend logs.
6. Report commit, push, backup, image publication, deployment, and functional/health evidence as separate outcomes.

## Success Criteria

- [x] Complete intended worktree is committed and `origin/main` matches the deployed source commit. Commit `7eee0b4` (2026-08-14T15:06:49+08:00), pushed `809c522..7eee0b4`; image published as `ghcr.io/4rankng/payroll-backend:7eee0b4` + `:latest` (digest `88db8aa4…`).
- [x] Fresh production backup exists and passes `gzip -t` before deployment begins. `payroll_mysql_backup_2026-08-14_150711.sql.gz` (3.4M, OneDrive), valid MySQL 8.0 dump header.
- [x] Production uses healthy amd64 images, endpoints and dependencies are healthy, and no new panic/fatal/error loop appears. Image created 15:08:05+08 (76s after commit), container recreated 15:08:25+08; `/api/healthz` 200, `/api/ready` 200, frontend 200; MySQL/Redis healthy (5 weeks up); 0 panic/FATAL in last 5m.
- [x] `loan_repayment_reminder` is registered at `0 9 * * *` under Asia/Ho_Chi_Minh scheduling. Confirmed in prod container logs: `Job registered name=loan_repayment_reminder cron=0 9 * * *`.
- [x] Rollback target and any unverified functional behavior are stated explicitly. Rollback: redeploy previous healthy image (prior `:latest`, digest `1cb83db…` pruned locally on prod but rebuildable from `809c522`) — no migration to reverse. Unverified: actual first reminder send (fires tomorrow 09:00 ICT only if a pending schedule is due 2026-08-15); channel behavior proven by unit/integration tests, not live send.
