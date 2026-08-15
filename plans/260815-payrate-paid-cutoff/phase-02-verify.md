---
phase: 2
title: "Verify"
status: pending
effort: "focused"
---

# Phase 2: Verify

## Overview

Run focused backend and frontend checks, then the project-required validation suitable for the changed financial behavior.

## Implementation Steps

1. Run the new focused Go regression test and affected backend package tests.
2. Run frontend type-check and lint after updating both render paths.
3. Run `make api-test`; report any environment-dependent limitation explicitly.
4. Refresh the project knowledge graph after the code change.

## Success Criteria

- [ ] Focused regression test passes.
- [ ] Backend and frontend static checks pass.
- [ ] Integration-test and graph refresh status is recorded.
