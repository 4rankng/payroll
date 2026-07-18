---
title: "Bank Transfer History"
status: approved
created: "2026-07-18"
---

# Bank Transfer History

## Problem

Admin and Partner need a simple read-only history that answers which completed bank transfers paid each employee in each fixed weekly payroll cycle. A worker may receive multiple bank transfers in one cycle, so the bank reference and amount for every transfer must remain visible.

## Approved Scope

- Completed bank transfers only.
- Aggregate by employee and weekly payroll cycle.
- Display every bank reference and its amount, plus the cycle total.
- Fixed cycles: Kỳ 1 days 1–7, Kỳ 2 days 8–14, Kỳ 3 days 15–21, Kỳ 4 days 22–28.
- Payment dates remain days 10, 17, 24, and day 1 of the following month.
- Admin sees all accessible data; Partner is server-scoped to accessible projects.

## Explicitly Out of Scope

- Reconciliation statuses or expected-versus-paid comparisons.
- Changes to payment percentage, payroll calculation, transfer execution, cycle schedule, or monthly-payment behavior elsewhere.
- Retry, refund, correction, or settlement actions.

## Recommended Design

Add a dedicated read-only history API built from completed `bulk_transfer_files.data` rows, then use one shared responsive screen for Admin and Partner. Each employee-cycle record contains a total and a visible list of bank-reference/amount pairs such as `FT26198846619959 · 1.548.000 ₫` and `FT26198940380850 · 450.000 ₫`.

## Acceptance Criteria

- Two completed transfers for the same employee and cycle render as two bank references and one summed total.
- Failed, pending, exported-only, or reference-less rows are not shown.
- Weekly cycle labels and date windows are correct, including Kỳ 4 ending on day 28.
- Partner cannot retrieve transfers outside accessible projects.
- No existing payment or payroll behavior changes.
