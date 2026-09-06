---
phase: 4
title: "Admin settings UI"
status: pending
priority: P1
effort: "1d"
dependencies: [1]
---

# Phase 4: Admin settings UI

## Overview
"Quảng cáo" tab in Settings: campaign list with time-aware status chips and click
counts, composer with duration presets and a 375px live preview reusing the employee
sheet component.

## Requirements
- Functional: list (title, project names or "Tất cả dự án", chip `Còn N ngày` /
  `Hẹn lịch` / `Hết hạn` / `Tạm dừng`, per-CTA click counts, expired greyed/grouped);
  composer (title, body, bullets ≤6, CTAs ≤3, project multi-select via existing
  `project-multi-selector.tsx` all-option, priority, active toggle); lifetime presets
  7/14/30/60 ngày (default 30) + "Tùy chọn" custom picker with live
  "Kết thúc: dd/MM/yyyy"; "Gia hạn" clones a fresh window instead of editing a dead
  campaign.
- Non-functional: parity between desktop and mobile admin SettingsPage (enforced by
  `settings-page-parity.test.tsx`); NOT wired into `useSettingsForm`/settings.service
  (own hooks); multi-select closes on click-outside (native Radix).

## Related Code Files
- Create: `frontend/src/components/settings/AdBannerSection.tsx`,
  `AdBannerComposer.tsx`, `frontend/src/hooks/api/useAdBanners.ts`
- Modify: `frontend/src/components/settings/SettingsTabList.tsx` (`ads` tab,
  `Megaphone`, rebalance spans to 6 × `col-span-2`),
  `frontend/src/pages/admin/SettingsPage/index.tsx` (TabsContent ~:117-131),
  `frontend/src/pages/mobile/admin/SettingsPage/index.tsx` (TabsContent ~:102-122),
  `frontend/src/pages/admin/SettingsPage/settings-page-parity.test.tsx`
  (`vi.mock` AdBannerSection)

## Implementation Steps
1. Hook + query keys.
2. Composer with preview (imports the SAME `EmployeeAdSheet`).
3. Section list + status chips + Gia hạn clone.
4. Tab wiring ×2 pages + span rebalance + parity test mock.

## Success Criteria
- [ ] `pnpm vitest run` green incl. `settings-page-parity.test.tsx`
- [ ] `npx tsc -p tsconfig.app.json --noEmit` + `pnpm lint` clean
- [ ] Composer validation mirrors domain rules; preview is the real component

## Risk Assessment
Forgetting the mobile page or the parity-test mock breaks the existing suite — both
are explicit checklist items above. Span rebalance changes an existing surface: keep
all six tabs legible at 375px (text-sm, px-2) and verify on the mobile viewport.
