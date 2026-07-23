# Design QA: Employee receiving-account card

final result: passed

## Source

- Reference: `/var/folders/8j/qs8k8y3n1hlfbl4q20k0hgjh0000gn/T/codex-clipboard-c351f598-aca5-48a8-ae4a-7df77f7923f7.png`
- Reference state: production employee dashboard, populated receiving-account card
- Reference dimensions: 1840 x 1060 pixels

## Implementation

- Route: `http://localhost:3000/employee`
- State: authenticated employee with populated bank information
- Desktop viewport: 1440 x 1000 CSS pixels at 2x device scale
- Focused card bounds: 468.17 x 252 CSS pixels
- Focused screenshot: `/Users/dev/.codex/visualizations/2026/07/23/019f8d8c-78fc-71f2-948e-1fa96ea63bea/employee-bank-card-restored.png`
- Desktop full-view screenshot: `/Users/dev/.codex/visualizations/2026/07/23/019f8d8c-78fc-71f2-948e-1fa96ea63bea/employee-page-restored-desktop.png`
- Mobile viewport: 390 x 844 CSS pixels at 2x device scale
- Mobile card bounds: 358 x 230 CSS pixels
- Mobile full-view screenshot: `/Users/dev/.codex/visualizations/2026/07/23/019f8d8c-78fc-71f2-948e-1fa96ea63bea/employee-page-restored-mobile.png`
- Side-by-side comparison: `/Users/dev/.codex/visualizations/2026/07/23/019f8d8c-78fc-71f2-948e-1fa96ea63bea/employee-bank-comparison.png`

## Verification history

1. Compared the production reference against the restored component in one side-by-side image.
2. Confirmed the cream security-paper texture, Đông Sơn drum placement, rounded border, heading hierarchy, account-number typography, divider, and two-column bank/owner layout.
3. Confirmed the implementation uses live employee bank values rather than the example values from the reference.
4. Confirmed no horizontal overflow at desktop or mobile widths.
5. Rechecked the focused card after restoration; no visual mismatch requiring another correction pass was found.
