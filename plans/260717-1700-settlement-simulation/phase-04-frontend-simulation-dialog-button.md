---
phase: 4
title: "Frontend: Simulation Dialog & Button"
status: pending
priority: P1
effort: "M"
dependencies: [3]
---

# Phase 4: Frontend: Simulation Dialog & Button

## Overview

Add a **"Mô phỏng đối soát"** button to both the desktop (`TransactionPageHeader`) and mobile (`LedgerPageHeader`) headers on `/admin/ledger`. Clicking opens a `SettlementSimulationDialog` (mirroring `OnePayFeeReportDialog`'s pattern) that calls `POST /api/v1/payrolls/simulate-settlement`, renders an answer-first verdict banner, a per-cycle table, drill-downs for included/excluded/remaining, reconciliation, and warnings.

## Requirements

- **Functional:** Button visible to admins (route already admin-gated). Dialog defaults to 4 cycles, allows 1–6. Shows dry-run caveat. Renders full result shape from Phase 3.
- **Non-functional:** No new dependencies. Reuse existing shadcn components. TanStack Query mutation pattern. Vietnamese inline strings (no i18n system — matches codebase convention).

## Architecture

### File layout

```
frontend/src/
├── components/ledger/
│   ├── SettlementSimulationDialog.tsx        ← NEW (main dialog, mirrors OnePayFeeReportDialog)
│   └── settlement-simulation/
│       ├── SimulationVerdictBanner.tsx       ← NEW (verdict + summary)
│       ├── SimulationCycleTable.tsx          ← NEW (per-cycle table)
│       ├── SimulationRowDrilldown.tsx        ← NEW (Tabs: included/excluded/remaining)
│       ├── SimulationRemainders.tsx          ← NEW (op-loss classification — most critical)
│       └── SimulationFindings.tsx            ← NEW (warnings + blocking)
├── types/api/
│   └── settlement-simulation.types.ts        ← NEW (mirrors backend DTO exactly)
├── config/api.config.ts                       ← MODIFY (add endpoint)
├── services/api/ledger.service.ts             ← MODIFY (add method)
├── hooks/ledger/useLedgerManagement.ts        ← MODIFY (add mutation)
├── components/transaction/TransactionPageHeader.tsx  ← MODIFY (desktop button)
├── components/ledger/LedgerPageHeader.tsx            ← MODIFY (mobile button)
├── pages/admin/TransactionsPage/index.tsx            ← MODIFY (wire dialog)
└── pages/mobile/admin/LedgerEntriesPage/index.tsx    ← MODIFY (wire dialog)
```

### Endpoint + service + hook

**`api.config.ts`** — add to the existing `payrolls` endpoint group (already has `exportBulkTransfer`):
```ts
payrolls: {
  // ... existing ...
  simulateSettlement: '/payrolls/simulate-settlement',
}
```

**`ledger.service.ts`** — add method (note: payroll endpoints live on the PayrollService client if one exists; if not, follow how `exportBulkTransfer` is called today and mirror that exactly):
```ts
async simulateSettlement(params: SettlementSimulationRequest): Promise<SettlementSimulationResult> {
  const response = await apiClient.post<SettlementSimulationResult>(
    API_ENDPOINTS.payrolls.simulateSettlement, params
  );
  if (!response.data) throw new Error('Không nhận được kết quả mô phỏng');
  return response.data;
}
```

**`useLedgerManagement.ts`** — add mutation. Note: this is a **read** masquerading as a POST; do **not** invalidate any query keys on success (no data changed). Only show a toast on error.
```ts
const simulateSettlementMutation = useMutation({
  mutationFn: (params: SettlementSimulationRequest) => ledgerService.simulateSettlement(params),
  onError: (error: Error) => {
    toast({ title: 'Lỗi', description: error.message, variant: 'destructive' });
  },
  // onSuccess: NO invalidation — simulation is read-only
});
// return: simulateSettlement: simulateSettlementMutation.mutateAsync,
//         isSimulatingSettlement: simulateSettlementMutation.isPending,
```

### Types (`settlement-simulation.types.ts`)

Generate strictly from the backend DTO in Phase 3. Key types:
```ts
export type SettlementVerdict = 'AN_TOAN_DE_XUAT' | 'CAN_KIEM_TRA' | 'KHONG_THE_TAT_TOAN';
export type RemainderClass = 'UNPAID_WAGES' | 'OP_LOSS' | 'STUCK_IN_FLIGHT';
export interface SettlementSimulationRequest { project_ids?: number[]; employee_ids?: number[]; projected_cycle_count?: number; for_month?: string; }
export interface SettlementSimulationResult { verdict: SettlementVerdict; summary: ...; reconciliation: ...; cycles: CycleProjection[]; remainders: RemaindersPayload; warnings: ...; snapshot_epoch: string; }
export interface CycleProjection { sequence: number; label: string; ... included: SimulationRow[]; excluded: SimulationExcludedRow[]; remaining: SimulationRow[]; findings: SimFinding[]; }
export interface RemaindersPayload {
  summary: { total_count: number; total_amount: number; unpaid_wages_count: number; unpaid_wages_amount: number; op_loss_count: number; op_loss_amount: number; stuck_in_flight_count: number; stuck_in_flight_amount: number; };
  items: ClassifiedRemainder[];
}
export interface ClassifiedRemainder {
  employee_id: number; employee_name: string; project_id: number; project_name: string;
  amount: number; timesheet_ids: number[];
  class: RemainderClass;
  reason: string;
  money_flow_evidence: { wallet_payment_id: number; wallet_payment_status: string; disbursed_at?: string; initiated_at?: string } | null;
}
export interface SimFinding { severity: 'blocking' | 'warning' | 'info'; code: string; message: string; in_production: boolean; }
```

### Button placement

**Desktop (`TransactionPageHeader.tsx`)** — insert after the existing "Chốt lương" block (~line 106), before "Thêm":
```tsx
<Button variant="outline" onClick={onSimulateSettlement} disabled={isSimulatingSettlement}>
  {isSimulatingSettlement ? <Loader2 className="w-4 h-4 animate-spin" /> : <ClipboardCheck className="w-4 h-4" />}
  Mô phỏng đối soát
</Button>
```
Add `onSimulateSettlement?: () => void; isSimulatingSettlement?: boolean;` to the props interface.

**Mobile (`LedgerPageHeader.tsx`)** — add to the `secondaryActions` array (~line 113–120):
```tsx
onSimulateSettlement && {
  key: 'settlement-simulation',
  label: 'Mô phỏng đối soát',
  icon: ClipboardCheck,
  onClick: onSimulateSettlement,
}
```

### Dialog structure

Mirror `OnePayFeeReportDialog.tsx` skeleton. Key sections:

1. **Header:** Title "Mô phỏng đối soát (dry-run)", description "Xem trước lần xuất sao kê này và các kỳ tiếp theo. Không thay đổi dữ liệu."
2. **Controls:** Cycle count `<Select>` (default 4, options 1–6). Optional project/employee filters (reuse pattern from existing ledger filters if practical; otherwise omit for v1).
3. **Submit button:** "Chạy mô phỏng" with `Loader2` spinner.
4. **Result area** (only after success):
   - **Verdict banner** (`SimulationVerdictBanner`): big colored Alert — green (`AN_TOAN_DE_XUAT`), amber (`CAN_KIEM_TRA`), red (`KHONG_THE_TAT_TOAN`). Shows summary stats: total eligible, included, remaining, all-settled yes/no.
   - **Reconciliation row:** `exported_total | ledger_receivable | delta` with green check or red ✗.
   - **Per-cycle table** (`SimulationCycleTable`): columns — Kỳ, From–To, Included (#/₫), Excluded (#), Remaining (#/₫), Findings (#). Each row expandable.
   - **Row drilldown** (`SimulationRowDrilldown`): Tabs `Bao gồm | Loại trừ | Còn lại`. Each tab a `<Table>` with employee, project, amount, reason (for excluded), bank account (masked).
   - **Remainders panel** (`SimulationRemainders`): the **most business-critical section**. Shows the 3-class summary as colored stat cards:
     - 🔴 `OP_LOSS` (red) — "Lỗ vốn — đã trả NV nhưng chưa thu hồi" with count + VND
     - 🟠 `UNPAID_WAGES` (amber) — "Còn nợ lương NV" with count + VND
     - 🔵 `STUCK_IN_FLIGHT` (blue) — "Đang chờ ngân hàng — không xuất lại" with count + VND
     - Below the cards: a `<Table>` of every remainder item with employee, project, amount, class badge, reason, and (for OP_LOSS / STUCK_IN_FLIGHT) a `money_flow_evidence` row citing the `wallet_payment_id` and status. Sorted `OP_LOSS` first (most urgent).
     - Top-of-panel caveat (Vietnamese): *"Các giao dịch thuộc kỳ trước vẫn chưa thanh toán sẽ KHÔNG được tự động bao phủ."*
   - **Findings** (`SimulationFindings`): list of blocking/warning/info. Each finding shows a "SẢN XUẤT KHÔNG KIỂM TRA" badge when `in_production === false`.
   - **Stale-data note:** "Dữ liệu này chốt tại `{snapshot_epoch}`. Nếu có thay đổi, chạy lại mô phỏng trước khi xuất."
5. **Footer:** "Đóng" only. No "Export now" button — the simulation must not be a one-click path to export; admin returns to the normal export flow (which will use `if_match_snapshot` under the hood — wire this optionally in a follow-up; for v1 the dialog just informs).

### Wiring in page components

**Desktop `TransactionsPage/index.tsx`:**
```tsx
const [simulationOpen, setSimulationOpen] = useState(false);
// ... in render:
<TransactionPageHeader
  // ...existing props...
  onSimulateSettlement={() => setSimulationOpen(true)}
/>
<SettlementSimulationDialog open={simulationOpen} onOpenChange={setSimulationOpen} />
```

**Mobile `LedgerEntriesPage/index.tsx`:** same pattern with `LedgerPageHeader`.

## Related Code Files

- **Create:** `frontend/src/components/ledger/SettlementSimulationDialog.tsx` + 4 sub-components in `settlement-simulation/`.
- **Create:** `frontend/src/types/api/settlement-simulation.types.ts`.
- **Modify:** `frontend/src/config/api.config.ts`, `frontend/src/services/api/ledger.service.ts`, `frontend/src/hooks/ledger/useLedgerManagement.ts`.
- **Modify:** `frontend/src/components/transaction/TransactionPageHeader.tsx`, `frontend/src/components/ledger/LedgerPageHeader.tsx`.
- **Modify:** `frontend/src/pages/admin/TransactionsPage/index.tsx`, `frontend/src/pages/mobile/admin/LedgerEntriesPage/index.tsx`.
- **Reference patterns:** `frontend/src/components/ledger/OnePayFeeReportDialog.tsx` (dialog skeleton), `frontend/src/components/ledger/DoubleEntryModal.tsx` (complex forms), `frontend/src/hooks/ledger/useLedgerManagement.ts` (mutation pattern).

## Implementation Steps

1. **Write types first** (`settlement-simulation.types.ts`) from the Phase 3 DTO. Run `pnpm type-check` to confirm.
2. **Add endpoint** to `api.config.ts` and **method** to `ledger.service.ts`.
3. **Add mutation** to `useLedgerManagement.ts` (read-only semantics — no invalidation).
4. **Build sub-components bottom-up:** `SimulationFindings` → `SimulationRowDrilldown` → `SimulationCycleTable` → `SimulationVerdictBanner`. Each is pure, fed by props.
5. **Assemble `SettlementSimulationDialog`** mirroring `OnePayFeeReportDialog` skeleton. Wire the mutation; render result sections.
6. **Add buttons** to both headers (props + JSX).
7. **Wire dialog** into both page components.
8. **Mask bank account numbers** in the UI (show last 4 digits) — these are PII; full numbers don't belong on a preview screen.
9. **Run `pnpm lint && pnpm type-check`.**

## Success Criteria

- [ ] "Mô phỏng đối soát" button visible on `/admin/ledger` (desktop + mobile) for admins.
- [ ] Clicking opens dialog; selecting cycle count + "Chạy mô phỏng" calls the API.
- [ ] Verdict banner, summary, reconciliation, per-cycle table, drilldowns, findings all render.
- [ ] Findings with `in_production=false` show the "SẢN XUẤT KHÔNG KIỂM TRA" badge.
- [ ] Dry-run caveat visible at top of dialog.
- [ ] Bank account numbers masked in UI.
- [ ] No TanStack Query invalidation fires on simulation success (read-only verified).
- [ ] `pnpm lint` clean. `pnpm type-check` clean.

## Risk Assessment

**Risk: large result payloads (many included rows) jank the UI.**
Mitigation: render drilldown tables inside `<ScrollArea>` with a max-height; paginate client-side at 100 rows per tab if needed. Cycle table stays compact (1 row per cycle).

**Risk: mobile layout crammed.**
Mitigation: on mobile, collapse the per-cycle table to cards; hide drilldown behind a per-cycle expander. Use existing `useIsMobile()` hook.

**Risk: PII (bank account) leakage in preview.**
Mitigation: always mask to last 4. Never log full account numbers to the console.

**Risk: admin clicks export immediately after sim, data changed meanwhile.**
Mitigation: dialog shows snapshot timestamp prominently. v1 does not auto-wire `if_match_snapshot` into the existing export button (out of scope); Phase 3 ships the backend guard so a future iteration can pass it. Document this as a known follow-up.
