---
title: "Admin wallet employee bank lookup brainstorm"
status: approved
created: 2026-08-18
---

# Admin Wallet Employee Bank Lookup Brainstorm

## Problem

Admin needs a current provider-confirmed view of an employee's bank account before operational use. Existing data shows the stored tuple; the manual-transfer form can verify arbitrary input, but `/admin/wallet` has no employee-oriented read-only lookup.

## Assumptions and Evidence

- Employee list already supports debounced server search and pagination: strong code evidence.
- Employee persists bank, account number, holder name, and validation status: strong code evidence.
- Active production provider implements OnePay `AccountVerifier`: strong code and provider-test evidence.
- Lookup should not edit data or transfer money: user-approved scope.

## Alternatives Considered

1. Dedicated server-authoritative employee lookup: approved. Safest authority boundary and clearest Admin workflow.
2. Reuse generic client-supplied check endpoint: rejected. Client can substitute the tuple and employee intent is implicit.
3. Put lookup only inside `Chuyển tiền`: rejected. Hides the requested standalone read-only action and couples lookup to money movement.

## Approved Design

Add one desktop/mobile wallet action and one shared dialog. Client selects only an employee ID. Backend loads employee plus bank, calls the active verifier, returns stored and confirmed information, and performs no write or transfer.

## Validation and Kill Criteria

- Kill or redesign if employee bank data cannot be resolved authoritatively without a client-supplied tuple.
- Block completion if missing data reaches OnePay, invalid and unavailable outcomes collapse together, or mobile lacks action parity.
- Verify focused tests, lint/type-check, integration, authenticated widths, authorization, and log privacy.

## Stakeholder Message

The lookup will use the same OnePay verification capability already trusted by disbursement, but behind an employee-specific read-only endpoint so Admin sees current bank-confirmed information without changing employee data or starting a transfer.
