# QA — Payroll Admin UI Bugs + Overflow + Font + Aesthetic — Pending Task Spec

**Date:** 2026-05-12 (walked 2026-05-14)
**For:** Next SWE / FE pickup
**Priority:** P0 → P2 mix (one P0 page-break, rest P1/P2 polish)
**Effort:** ~1.5–2 dev-days (P0 alone: ~15 min)
**Tester:** Claude (Cowork QA pass, user `frankng`)

---

## Summary

- **Total issues:** 23  (P0: 1 · HIGH: 4 · MED: 11 · LOW: 7)
- **Pages tested:** `/admin` (dashboard), `/admin/wallet`, `/admin/ledger` (= `/admin/transactions` redirect), `/admin/timesheet`, `/admin/advance-payments`, `/admin/employees`, `/admin/projects`, `/admin/users`, `/admin/loans`, `/admin/system-health`, `/admin/cron-health`, `/admin/audit-log`, `/admin/settings`, `/admin/send-notification`, `/admin/manual-disbursement` (404)
- **Viewport tested:** browser locked to **1600×857** inner viewport — could not resize narrower in this env. Mobile/tablet patterns audited via source (responsive utilities + mobile-prefixed pages).
- **Most common pattern:** **inconsistency** — fonts, border-radius, date format, capitalization, language. The system has tokens but devs bypass them with ad-hoc `text-[11px]` / `rounded-md` / raw `d/M/yyyy` strings. App feels capable but visually noisy.
- **Critical finding:** `/admin/timesheet` is fully broken — page renders an error boundary because `ChuyenLoDialog` references an undefined identifier.

---

## Issues by Page

### /admin/timesheet — Bảng công

- [ ] **TASK-Q01** [P0 / CRITICAL]: Trang bảng công 100% blank — runtime error
  - **Observation:** Trang hiển thị "Lỗi khi tải trang quản trị / `ninePayDisabled is not defined`". Stack trace points to `ChuyenLoDialog`.
  - **Root cause:** `frontend/src/components/timesheet/ChuyenLoDialog/index.tsx:146` — `useEffect` deps array uses `ninePayDisabled`, but the variable defined on line 134 is `providerDisabled`. Typo from a recent rename.
  - **Impact:** Toàn bộ trang bảng công không sử dụng được. Admin/manager không vào được màn hình duyệt công.
  - **Fix (1 char):** Đổi `ninePayDisabled` → `providerDisabled` ở line 146.
  - **Verify:** `make api-test` không phát hiện vì là lỗi runtime FE — sau khi sửa, manual smoke test `/admin/timesheet` cùng modal "Chuyển lô".
  - **Severity:** P0 (blocks core admin flow)

### /admin/wallet — Quản lý ví

- [ ] **TASK-Q02** [HIGH]: `₫` (U+20AB) glyph rendering broken — looks like "đ_" with detached underline
  - **Observation:** KPI hero "100.000 ₫" và "51.000 ₫" render với gạch ngang/ underline rời khỏi chữ — Manrope font (`font-display`) không có glyph cho U+20AB nên fallback chia bậc.
  - **Root cause:** `tailwind.config.ts:31` — `display: ['Manrope', ...]`. `WalletPage` line 45 dùng `Intl.NumberFormat('vi-VN') + " ₫"` rồi render trong `font-display` (KpiHeroCard).
  - **Fix options:**
    - (A) Đổi `₫` sang `đ` trong `formatVND` (xấu kiểu chữ nhưng glyph có sẵn).
    - (B) Inject `@font-face` fallback hoặc thêm `Noto Sans` vào font stack TRƯỚC Manrope cho currency span.
    - (C) Wrap symbol `<span className="font-sans">₫</span>` chỉ cho mỗi ký tự currency — cleanest.
  - **Severity:** HIGH (visible on every money figure across app)

- [ ] **TASK-Q03** [MED]: Toast hiển thị "Số dư khớp: 100.000 ₫ VND" — currency hai lần
  - **Observation:** `WalletPage/index.tsx:78` — `toast.success(\`Số dư khớp: ${formatVND(...)} VND\`)`. `formatVND` đã append " ₫", template lại thêm " VND".
  - **Fix:** Bỏ ` VND` ở template, hoặc đổi `formatVND` không append symbol khi gọi trong toast.
  - **Severity:** MED

- [ ] **TASK-Q04** [LOW]: Section header "Lịch sử giao dịch (2)" dùng sentence-case
  - Trên các trang khác (DashboardPage, AdvancePayments) section header dùng ALL-CAPS micro-label ("SỐ DƯ KHẢ DỤNG", "BẢNG CÔNG"). Inconsistent.
  - **Fix:** Quyết một style — đề xuất ALL-CAPS uppercase tracking-wider cho section divider; sentence-case cho headings cấp 1.

### /admin/advance-payments — Ứng lương

- [ ] **TASK-Q05** [HIGH]: KPI "Ví tiền" hiển thị error state "Không thể tải số dư" làm vỡ grid 3-column
  - **Observation:** Ba KPI card (Tổng quan / Tài chính / Ví tiền). Card Ví tiền chỉ có 1 dòng error → chiều cao khoảng 1/4 hai card kia → grid lệch xấu, mất professionalism.
  - **Fix:** Khi `walletBalance` fail, render skeleton-cao-bằng-anh-em hoặc retry button + last-known value. Tối thiểu set `min-height` bằng anchor card.
  - **Severity:** HIGH (visible on landing-after-login for ketoan flow)

- [ ] **TASK-Q06** [MED]: Hàng đầu trong bảng có text rò rỉ "100.000 đ  Hủy" — "Hủy" hiển thị inline với amount
  - **Observation:** Yêu cầu hủy → "Hủy" badge bị render cạnh số tiền không cách border, trông như phần của giá trị.
  - **Fix:** Render `Hủy` badge ở column "Trạng thái" only, hoặc thêm separator/space rõ ràng (e.g. wrap trong `<Badge>` riêng).
  - **Severity:** MED

- [ ] **TASK-Q07** [LOW]: Filter pill số đếm hiển thị "0" cho trạng thái không có (Thất bại 0)
  - Có thể ẩn pill khi count = 0, hoặc dim opacity. Hiện tại "Thất bại (0)" cùng nhấn mạnh như "Hoàn tất (10)".

### /admin/loans — Khoản vay

- [ ] **TASK-Q08** [MED]: Date format `10/6/2026` không zero-pad — không đồng nhất app
  - **Observation:** Cột "Ngày TT tới" và "Ngày giải ngân" hiển thị "10/6/2026", "12/5/2026". Phần còn lại của app dùng `dd/MM/yyyy` (13/05/2026).
  - **Root cause:** Default `Date.toLocaleDateString` được dùng thay vì `date-fns format(..., 'dd/MM/yyyy', { locale: vi })`.
  - **Fix:** Audit `frontend/src/components/loans/loan-table-config.tsx` cell renderers cho `next_payment_date` (line ~58), `created_at` (line 249 dùng `"d MMM yyyy"` — khác nữa!). Standardize → `dd/MM/yyyy`.
  - **Severity:** MED

- [ ] **TASK-Q09** [MED]: KPI strip không có card container — khác mọi page khác
  - Loans page dùng `InlineStatStrip` (4 số inline). Wallet/Advance-payments/Dashboard dùng card box. Inconsistent visual rhythm.
  - **Fix:** Bọc trong `KpiHeroCard` cluster cho parity, hoặc convert các page khác về inline-strip cho compact dashboards.

- [ ] **TASK-Q10** [LOW]: Cột "Ngày TT tới" có dòng màu cam, không có legend
  - Người dùng không biết cam = sắp đến hạn / quá hạn / khác. Cần legend hoặc tooltip.

### /admin/cron-health — Lịch công việc

- [ ] **TASK-Q11** [MED]: Tiêu đề trang "Cron Health" — bằng tiếng Anh giữa app tiếng Việt
  - **Fix:** Đổi thành "Tình trạng tác vụ định kỳ" hoặc "Sức khỏe lịch chạy".

- [ ] **TASK-Q12** [LOW]: Cron expression hiển thị thô (`0 20 * * *`) — không thân thiện
  - **Fix:** Dùng `cronstrue/i18n/locales/vi` để render "Hàng ngày lúc 20:00".

- [ ] **TASK-Q13** [LOW]: Badge "10 jobs / 10 bật" trên top-right phối hợp font + viền không đồng bộ
  - Một badge nền xám, một viền xanh — tone không cùng family. Unify visual weight.

### /admin/system-health — Kiểm tra API

- [ ] **TASK-Q14** [LOW]: Tiêu đề "API Health" — same English issue như cron-health
  - **Fix:** "Tình trạng API" / "Sức khỏe API".

### /admin/audit-log — Nhật ký

- [ ] **TASK-Q15** [LOW]: Content area có max-width cứng → màn 1600px+ tạo viền trống hai bên rất rộng
  - **Fix:** Hoặc widen content max-w to ≥1280px, hoặc add side panel filters để dùng không gian trống.

### /admin/employees — Nhân viên

- [ ] **TASK-Q16** [HIGH]: Cột "Địa chỉ" overflow phải ngoài viewport — không truncate
  - **Observation:** Địa chỉ dài "Bắc Hưng - Tiên Lãng - Hải Phòng, Xã..." bị clip cứng tại edge, không có ellipsis hay tooltip.
  - **Fix:** Add `truncate max-w-[240px]` + Tooltip on hover hiển thị full address. Apply cho cả "Dự án" nếu name dài.
  - **Severity:** HIGH (table looks broken on standard laptop screens)

- [ ] **TASK-Q17** [LOW]: Backslash leak "Xã Cư M\'gar" trong địa chỉ
  - Backend data escape leaking. Có thể là DB sanitization issue. FE có thể `replace(/\\'/g, "'")` như guard.

### /admin/users — Người dùng

- [ ] **TASK-Q18** [LOW]: KPI card "Tổng 640" có chấm tím floating bottom-right
  - Có vẻ là notification indicator nhưng không rõ ngữ cảnh. Loại bỏ hoặc thêm tooltip "X user mới hôm nay".

### /admin/settings — Cài Đặt

- [ ] **TASK-Q19** [LOW]: Header "Cài Đặt Hệ Thống" dùng Title Case — khác phần lớn app dùng sentence case
  - Decide rule: page H1 sentence case ("Cài đặt hệ thống") — apply across all `PageHeader` calls.

- [ ] **TASK-Q20** [LOW]: Card "Tên khách hàng" hẹp hơn các card phía trên — đứng lẻ loi
  - Hoặc cho fullspan, hoặc add card thứ 2 cùng dòng để balance.

### /admin/send-notification — Gửi Thông Báo

- [ ] **TASK-Q21** [MED]: Title input và content textarea không có border rõ → empty state nhìn như chưa load
  - **Fix:** Thêm `border` cho input/textarea (hiện chỉ underline mờ).

- [ ] **TASK-Q22** [LOW]: Bell icon nổi bottom-right của textarea không rõ chức năng
  - Decorative? Submit shortcut? Cần tooltip hoặc remove.

### /admin/manual-disbursement — DELETED ROUTE

- [ ] **TASK-Q23** [LOW]: 404 page nhưng AdminLayout (sidebar) biến mất
  - **Observation:** Khi 404 trigger, layout bị unmount → full-screen 404. Khác behavior với các route hợp lệ.
  - **Fix:** 404 nên render trong `AdminLayout` outlet (giữ sidebar) để user navigate được mà không phải refresh.

### / (login)

- [ ] **TASK-Q24** [LOW]: Footer "Tiện lợi" wrap xuống dòng 2 ở viewport ~960px
  - "An toàn · Bảo mật · Tiện lợi" — "Tiện lợi" tách dòng riêng. Dùng `whitespace-nowrap` trên từng pill hoặc giảm padding.

---

## Cross-cutting Issues

- [ ] **TASK-FONT-01** [HIGH]: 462 ad-hoc `text-[Npx]` overrides spread across 150 files
  - **Pattern:** Devs viết `text-[11px]`, `text-[12px]`, `text-[13px]` thay vì dùng `text-xs` (11px), `text-sm` (12px), `text-base` (13px) đã định nghĩa trong `tailwind.config.ts:178-188`.
  - **Why bad:** Future scale change require touching 150 files; lint không thể bắt drift; visual rhythm phá vỡ.
  - **Fix:** Add eslint rule `no-restricted-syntax` blocking `text-\[\d+px\]`; codemod sweep replacing common values với token equivalents.

- [ ] **TASK-FONT-02** [HIGH]: `font-display` (Manrope) không có glyph U+20AB (₫)
  - Đã chi tiết ở **TASK-Q02**. Cross-cutting vì ₫ render khắp nơi (wallet, ledger, advance-payments, dashboard, sheets).
  - **Recommended fix:** wrapper utility `<MoneySymbol/>` chuyển sang `font-sans` chỉ cho ký tự currency.

- [ ] **TASK-OVERFLOW-01** [HIGH]: Tables không có `truncate` policy nhất quán
  - Pages có long-text columns (employees address, ledger description, audit-log message) đều thiếu `truncate + tooltip`. Khi data dài, layout vỡ.
  - **Fix:** Standardize column config — text columns >X chars → `truncate max-w-[Npx]` + Radix Tooltip on hover.

- [ ] **TASK-OVERFLOW-02** [MED]: KPI value font-size dùng `clamp(1.25rem, 2.5vw, 1.75rem)`
  - Trên 9+ digit VND values (e.g. `3.056.669.711 ₫`), với `clamp` nhỏ nhất 20px, đôi khi vỡ wrap. Cần `whitespace-nowrap` + `tabular-nums` (đã có tabular-nums) + đảm bảo card width có `min-w` đủ ~14ch.

- [ ] **TASK-DATE-01** [MED]: Date format inconsistent — `dd/MM/yyyy` vs `d/M/yyyy` vs `d MMM yyyy`
  - Audit từng file render date, standardize. Recommended: `dd/MM/yyyy` (zero-padded) cho table/list, "13 Thg 5, 2026" cho hero/detail. Document in `docs/frontend-conventions.md`.

- [ ] **TASK-COLOR-01** [HIGH]: `KpiHeroCard` color prop là dead code — 4 colors map 100% identical tokens
  - **File:** `frontend/src/components/admin-dashboard/KpiHeroCard.tsx:30-58`
  - All 4 variants (blue, emerald, amber, violet) đều dùng `bg-primary`. Người dùng API tưởng đang phân biệt visually nhưng card luôn cùng màu primary.
  - **Fix:** Either implement actual color variants, hoặc remove the prop entirely.

- [ ] **TASK-LANG-01** [MED]: Trộn lẫn tiếng Anh / Việt cho page titles
  - "Cron Health", "API Health" giữa app tiếng Việt. Convert sang Việt hoặc adopt formal English convention cho technical pages.

- [ ] **TASK-CASE-01** [MED]: Capitalization không thống nhất — "Cài Đặt Hệ Thống" (Title) vs "Quản lý ví" (sentence) vs "Gửi Thông Báo" (Title)
  - Pick one rule. Recommend sentence case cho page H1.

- [ ] **TASK-RADIUS-01** [MED]: Border-radius mix `rounded-md` / `rounded-lg` / `rounded-xl` cho cùng kiểu container
  - WalletPage uses `rounded-xl` ngoài + `rounded-md` cho inner mini-cards. ReconciliationDownloadDialog uses `rounded-md`. AuditLogCard uses `rounded-xl`. PayrateEditPage all `rounded-xl`. Settings uses `rounded-xl` for icon, but cards default radius.
  - **Fix:** Define token: card outer = `rounded-xl` (12px), pills/badges = `rounded-full`, inputs/inner boxes = `rounded-lg` (8px). Add `eslint-plugin-tailwindcss/no-arbitrary` rules.

---

## Visual Quality Assessment

### Per-page grade (S/A/B/C/D scale)

| Page | Grade | Strengths | Weaknesses |
|------|-------|-----------|------------|
| `/admin` (Dashboard) | **B** | Masonry-like KPI cards, monthly table, top-paid section đẹp | KPI colors all-primary nên dashboard thiếu visual encoding; "Đang tải dữ liệu..." spinner placeholder cho Phân Bổ Lương thô |
| `/admin/wallet` | **B–** | Layout sạch, KPI hero rõ ràng, transaction list dễ scan | ₫ glyph vỡ, double currency trong toast, section header lower-case |
| `/admin/transactions` (`ledger`) | **B** | Status-color vertical bar nice, capital contribution cards độc đáo | File names dài full-width không truncate, legend dots nhỏ khó đọc |
| `/admin/timesheet` | **F** | — | Page crash hoàn toàn |
| `/admin/advance-payments` | **C+** | Filter pills với count badge, table dense | KPI grid vỡ vì wallet error state, "Hủy" badge inline xấu |
| `/admin/employees` | **B–** | Avatar + CCCD monospace pleasant, bank info clear | Address overflow, comma spacing, escape char leak |
| `/admin/projects` | **B+** | Sạch, monospace code, count badges tròn xinh | Half table mostly empty Lương tháng — sparse feel |
| `/admin/users` | **B** | Avatar consistent, role badges OK | Phantom purple dot trên KPI Tổng |
| `/admin/loans` | **C** | Functional table | Date format khác app, KPI strip layout khác, không legend cho cam-coded dates |
| `/admin/system-health` | **A–** | **Best looking page** — endpoint cards với HTTP method tags, P95/Avg pleasant typography, severity pill nice | Tiêu đề "API Health" bằng Anh ngữ; tiny labels |
| `/admin/cron-health` | **C** | Clean rows, status dots clear | English title, raw cron expressions, header badges inconsistent |
| `/admin/audit-log` | **B+** | Card-per-entry approach pleasant, action-type color coding | Page max-w cứng tạo nhiều whitespace 2 bên |
| `/admin/settings` | **C+** | Form rất sạch, icon trong rounded square | Title casing inconsistent, last card lonely on right |
| `/admin/send-notification` | **C–** | Quick recipient chips clean | Input không border rõ, bell icon mystery, "Gửi" button disabled state ambiguous |
| `/admin/manual-disbursement` (404) | n/a | 404 illustration OK | Layout breaks — sidebar mất |

### Overall aesthetic verdict

**Currently feels:** "*Capable but uneven — like a senior dev's first solo design system*". Có ambition (`Industrial Luxury Typography` comment trong tailwind.config 😄), có tokens, có hooks, có animation library. Nhưng cách tokens được sử dụng thiếu kỷ luật.

**Most jarring inconsistencies (top 3):**

1. **Same-but-different.** KpiHeroCard accepts `color: blue|emerald|amber|violet` nhưng cả 4 đều render giống nhau (primary). User thấy 4 KPI giống màu mà code chú thích trang trí khác — cognitive dissonance.
2. **Token bypass.** 462 instances of `text-[Npx]` ad-hoc. Type scale rõ ràng (xs=11/sm=12/base=13/lg=14/xl=16) nhưng devs hardcode. Border-radius cùng pattern.
3. **Currency glyph.** ₫ render xấu trên Manrope khiến mọi số tiền — thứ quan trọng nhất trong app payroll — trông thiếu chăm chút.

**Quick aesthetic wins (<1 hour each):**

- (A) **Fix ₫ glyph** — wrap currency symbol trong `font-sans`. 1 utility component, ~30 min including grep + sweep.
- (B) **Standardize section labels** ALL-CAPS uppercase tracking-wider. Replace lower-case section headers (Wallet "Lịch sử giao dịch") với pattern dùng trên Dashboard/Advance-payments.
- (C) **Hide zero-count filter pills** hoặc giảm opacity-60. Hiện "Thất bại (0)" nhấn ngang "Hoàn tất (10)".
- (D) **Date format sweep** — find/replace `d/M/yyyy` → `dd/MM/yyyy` trong loans/projects (3 files).
- (E) **Remove the dead KpiHeroCard color prop** OR implement real variants (4 distinct accent colors).
- (F) **Localize "Cron Health" / "API Health"** page titles.

**Larger polish efforts (S effort, half-day to day each):**

- (G) **Truncate + tooltip policy across tables** — write a `<TruncatedCell>` component, apply consistently to employees address, ledger descriptions, loan lender names.
- (H) **Codemod `text-[Npx]` → tokens** — write a node script using `recast` or `jscodeshift`. Plus an eslint rule.
- (I) **404 inside AdminLayout** — refactor router để fallback route render trong layout outlet thay vì sibling.
- (J) **Empty state language** — currently mix of "Không thể tải số dư", "Chưa chạy", "—", "Đang tải dữ liệu...". Standardize: skeleton on first-load, illustrated empty state nếu no-data, retry button nếu error.

---

## Aesthetic TASK entries (per user rubric)

- [ ] **TASK-AESTH-01** [MED]: **Spacing rhythm** — replace ad-hoc `gap-[Npx]` / `px-[Npx]` với 4-pt scale tokens (`gap-1/2/3/4/6`, `p-1/2/3/4/6`). Sweep `text-[Npx]` cùng lúc.
- [ ] **TASK-AESTH-02** [MED]: **Border radius unification** — `rounded-xl` cho cards outer, `rounded-lg` cho inner/inputs, `rounded-full` cho pills. Eslint rule chống arbitrary radius.
- [ ] **TASK-AESTH-03** [MED]: **Color palette discipline** — KpiHeroCard color prop hoặc dùng đúng (4 distinct accent colors), hoặc xóa. Sidebar currently navy gradient, accent currently `--primary` (cùng navy) → dashboard thiếu hierarchy.
- [ ] **TASK-AESTH-04** [LOW]: **Typography pairing** — Body `text-base = 13px` (config) là quá nhỏ cho admin desktop, đặc biệt nội dung tiếng Việt với dấu phụ. Cân nhắc bump `base` lên 14px, `sm` lên 13px.
- [ ] **TASK-AESTH-05** [LOW]: **Empty state polish** — current empty states là plain text. Add illustrated states (small SVG + tagline) for: no transactions, no loans, no notifications.
- [ ] **TASK-AESTH-06** [LOW]: **Loading state consistency** — sometimes skeleton, sometimes "Đang tải dữ liệu...", sometimes spinner. Pick skeleton everywhere for layout-preserving feel.
- [ ] **TASK-AESTH-07** [LOW]: **Iconography weight audit** — Lucide icons mix `h-3/h-3.5/h-4/h-5` randomly. Standardize: section icon = 16px (`h-4`), inline icon = 14px (`h-3.5`), KPI accent = 16px in 32px tile.
- [ ] **TASK-AESTH-08** [LOW]: **Number formatting helper** — single `formatVND(value, { withSymbol: true|false })` used everywhere. Eliminate manual `value + ' ₫'` concatenations.
- [ ] **TASK-AESTH-09** [MED]: **Vietnamese diacritics line-height** — `text-base` defined `lineHeight: 1.5` but custom `text-[13px]` blocks don't inherit — letters with stacked diacritics (ầ, ễ, ợ) clip on tight rows.
- [ ] **TASK-AESTH-10** [LOW]: **Mobile feel audit** — desktop-only QA in this env; có separate `mobile/*` page tree implies mobile is a parallel app. Worth verifying mobile/desktop visual parity (icons, colors, spacing tokens shared).

---

## Acceptance Criteria

- [ ] `/admin/timesheet` loads with no JavaScript console error
- [ ] All money figures across app render `₫` cleanly (no detached underline)
- [ ] All admin tables with long text use `truncate` + tooltip
- [ ] KPI cards have uniform heights even when one card is in error/empty state
- [ ] Date format = `dd/MM/yyyy` consistently in lists/tables
- [ ] Zero new `text-[Npx]` or `rounded-md/lg/xl` arbitrary occurrences after sweep
- [ ] Section header capitalization rule documented + applied
- [ ] No English-only page titles in admin tree
- [ ] Lighthouse a11y ≥ 90 on `/admin/wallet`, `/admin`, `/admin/advance-payments`

---

## Files Likely Changed

**P0 fix:**
- `frontend/src/components/timesheet/ChuyenLoDialog/index.tsx` — line 146

**Currency glyph (TASK-Q02 / TASK-FONT-02):**
- New: `frontend/src/components/shared/MoneySymbol.tsx`
- Sweep: `frontend/src/utils/vietnamese.ts`, `frontend/src/pages/admin/WalletPage/index.tsx:44-46`, all `formatVND` callsites

**KpiHeroCard color rebuild (TASK-COLOR-01):**
- `frontend/src/components/admin-dashboard/KpiHeroCard.tsx` — COLOR_MAP rebuild

**Date format (TASK-Q08 / TASK-DATE-01):**
- `frontend/src/components/loans/loan-table-config.tsx`
- `frontend/src/components/projects/details/ProjectHeader.tsx`

**Token sweeps (TASK-FONT-01 / TASK-RADIUS-01):**
- Many files; do as a separate PR using codemod

**Toast double-currency (TASK-Q03):**
- `frontend/src/pages/admin/WalletPage/index.tsx:78`

**Employees address overflow (TASK-Q16):**
- `frontend/src/config/employee-table-desktop.tsx`

**Manual-disbursement 404 layout (TASK-Q23):**
- `frontend/src/App.tsx` or `frontend/src/routes/*` — wrap NotFound in AdminLayout outlet

---

## Notes / Non-Issues

- **/admin/manual-disbursement** returns 404 — but folder `frontend/src/pages/admin/ManualDisbursementPage/` exists with multiple components. Route likely was removed but page code left behind. Worth a confirmation: cleanup dead code, or restore route?
- **Cron schedule raw expressions** flagged LOW because audience is technical admins; could keep raw with mouseover human-readable.
- **Long file names trong ledger** ("Trả lương cho VFIC Manpower, file: ...xlsx") — could break into 2 lines instead of truncating since they encode useful info.
- **Sidebar navigation** itself is solid — `XIN CHÀO Frank Ng` pinned bottom, version `v1.10.0` discreet, grouped sections (Quản lý / Tài chính / Hệ thống).
- **Login page Cute design** with money/coins illustration is OK aesthetic — feels branded.
- **AdminSidebar.tsx has 9 ad-hoc text sizes** — concentrated source of token bypass; refactor priority for unified look.

---

## How to verify after fixes

```bash
# Backend (per CLAUDE.md "make api-test before any feature merge")
cd /Users/dev/Documents/projects/payroll
make api-test

# Frontend smoke:
# 1. Login as frankng / Admin123
# 2. Visit /admin/timesheet — must render
# 3. Open Chuyển lô dialog — must not crash
# 4. /admin/wallet — verify ₫ renders cleanly in DPR=1 and DPR=2
# 5. /admin/advance-payments — verify KPI grid height symmetry
# 6. /admin/loans — verify all dates dd/MM/yyyy
# 7. Console.error grep — no ReferenceError, no missing-key warnings on tested pages
```

**No git operations during pickup per workflow rule. Manual review and commit by user.**
