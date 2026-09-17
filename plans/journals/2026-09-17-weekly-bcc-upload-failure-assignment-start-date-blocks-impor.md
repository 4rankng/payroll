---
title: Weekly BCC upload failure — assignment start_date blocks imports
date: 2026-09-17
summary: "Prod: partner-created assignment defaulted start_date=today (frontend sends it explicitly; backend defaults today), rejecting weekly BCC uploads covering Sept 10-14. All-or-nothing replacement rolled back 101 valid rows; weekly paths hid per-row reasons. Partner self-recovered by backdating. Fixes on branch investigate-prod-upload-fail: per-row error surfacing in weekly BCC + weekly payment paths; feature plan 260917-1351-assignment-smart-start-date adds smart start defaults (last-timesheet+1, else month start) across all 7 creation paths + self-healing import backdate."
---

# Weekly BCC upload failure — assignment start_date blocks imports

Prod: partner-created assignment defaulted start_date=today (frontend sends it explicitly; backend defaults today), rejecting weekly BCC uploads covering Sept 10-14. All-or-nothing replacement rolled back 101 valid rows; weekly paths hid per-row reasons. Partner self-recovered by backdating. Fixes on branch investigate-prod-upload-fail: per-row error surfacing in weekly BCC + weekly payment paths; feature plan 260917-1351-assignment-smart-start-date adds smart start defaults (last-timesheet+1, else month start) across all 7 creation paths + self-healing import backdate.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
