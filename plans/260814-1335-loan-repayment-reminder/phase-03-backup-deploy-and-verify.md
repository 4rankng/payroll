---
phase: 3
title: "Backup deploy and verify"
status: pending
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

- [ ] Complete intended worktree is committed and `origin/main` matches the deployed source commit.
- [ ] Fresh production backup exists and passes `gzip -t` before deployment begins.
- [ ] Production uses healthy amd64 images, endpoints and dependencies are healthy, and no new panic/fatal/error loop appears.
- [ ] `loan_repayment_reminder` is registered at `0 9 * * *` under Asia/Ho_Chi_Minh scheduling.
- [ ] Rollback target and any unverified functional behavior are stated explicitly.
