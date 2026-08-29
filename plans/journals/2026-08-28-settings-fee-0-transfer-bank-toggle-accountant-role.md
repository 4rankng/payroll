---
title: "Settings fee 0% + transfer-bank toggle + accountant role"
date: 2026-08-28
summary: "Shipped 3-feature wave: fee schedule accepts 0%/min-fee-0 (Validate + form-helpers); transfer_bank_visible setting (default on) hides beneficiary rows in 2 Excel templates (labels+values cleared) and 2 email templates (payroll ShowBank conditional, FlexPay bank block swap) via TransferBankInfo.Hidden zero-value-safe; new accountant role (casbin allow-list 4 feature groups, /accountant no-sidebar 4-tab page reusing existing dialogs, role plumbing FE). go build/tests/lint/vitest green; pre-existing forecast+persistence failures verified unrelated via stash; reviewer subagent unresponsive -> independent blast-radius checks; ForceChangePasswordDialog.tsx carries unrelated parallel-session edit - do not co-commit."
---

# Settings fee 0% + transfer-bank toggle + accountant role

Shipped 3-feature wave: fee schedule accepts 0%/min-fee-0 (Validate + form-helpers); transfer_bank_visible setting (default on) hides beneficiary rows in 2 Excel templates (labels+values cleared) and 2 email templates (payroll ShowBank conditional, FlexPay bank block swap) via TransferBankInfo.Hidden zero-value-safe; new accountant role (casbin allow-list 4 feature groups, /accountant no-sidebar 4-tab page reusing existing dialogs, role plumbing FE). go build/tests/lint/vitest green; pre-existing forecast+persistence failures verified unrelated via stash; reviewer subagent unresponsive -> independent blast-radius checks; ForceChangePasswordDialog.tsx carries unrelated parallel-session edit - do not co-commit.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
