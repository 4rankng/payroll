# Kanban cards — UU PRO migration wave (261004)

Task: create 15 .docx ticket cards for the approved Untitled UI PRO frontend migration in `kanban/TODO/`, replicating the nepocorp card format.

## Files created (all in `kanban/TODO/`)

| # | File | Wave |
|---|------|------|
| 1 | `20261004_1_cau-hinh-token-uu-pro.docx` | W1 token bridge + xoá 9 file ui chết |
| 2 | `20261004_2_nang-cap-phan-tinh-badge-card.docx` | W2 compat badge/card/separator/avatar/skeleton/label |
| 3 | `20261004_3_nang-cap-o-nhap-lieu.docx` | W3 compat input/textarea |
| 4 | `20261004_4_nang-cap-button.docx` | W4 compat button (255 nơi gọi) |
| 5 | `20261004_5_nang-cap-overlay-dialog.docx` | W5 dialog/alert-dialog/sheet/popover/tooltip |
| 6 | `20261004_6_nang-cap-menu-select.docx` | W6 dropdown-menu/select |
| 7 | `20261004_7_nang-cap-form-controls.docx` | W7 checkbox/radio/switch/slider/progress/tabs/accordion |
| 8 | `20261004_8_nang-cap-bang-du-lieu.docx` | W8 table/pagination/breadcrumb/filter-chip-bar |
| 9 | `20261004_9_nang-cap-toast.docx` | W9 sonner toastOptions |
| 10 | `20261004_10_chuyen-doi-trang-auth.docx` | W10 5 trang auth, UU thuần + @untitledui/icons |
| 11 | `20261004_11_chuyen-doi-cong-nhan-vien.docx` | W11 employee portal (mobile-first PWA) |
| 12 | `20261004_12_chuyen-doi-cong-doi-tac.docx` | W12 partner desktop 5 + mobile 4 trang |
| 13 | `20261004_13_chuyen-doi-quan-tri.docx` | W13 admin desktop 18 + mobile 17, 2–3 sub-wave |
| 14 | `20261004_14_chuyen-doi-adv-partner-ke-toan.docx` | W14 adv-partner + accountant |
| 15 | `20261004_15_loai-bo-di-san-va-hoan-thien.docx` | W-final gỡ daisyUI/Radix/lucide, bundle report, docs+ADR |

Every card contains the four required sections: **Phạm vi**, **File chính**, **Tiêu chí nghiệm thu** (10 common AC + card-specific), **Checklist QA thủ công** (7 common QA + card-specific), plus the nepocorp metadata block (`Mã`, `Loại`, `Ngày tạo: 04/10/2026`, `Nguồn`, `Màn hình / khu vực`, `Trạng thái: TODO — chưa bắt đầu`).

## Method

Template reuse instead of docx-js generation: unpacked `kanban/QA_TESTED/20260926_1-data-dense-toan-bo-trang-moi-thiet-bi.docx`, per card replaced `word/document.xml` (same `CardTitle`/`SectionHead` styles, bullet `numId=2`, page geometry 12240×15840/margins 1080) and stamped `docProps/core.xml` dates, then packed with the docx skill's `pack.py` (runs DOCX schema validation with auto-repair per file). Verified `styles.xml` byte-identical to the example card; content spot-checked via pandoc (cards 1, 4, 13, 15).

## Deviations / notes

1. **pack.py interpreter:** the skill venv python is 3.9.6 and cannot parse `pack.py`'s `str | None` syntax (needs 3.10+); homebrew pythons lacked `lxml`/`defusedxml`. Ran `pack.py` via `uv run --python 3.12 --with lxml --with defusedxml` (ephemeral env, no global installs). The generator script itself ran under the skill venv python as required.
2. **Admin mobile page count (card 13):** task text says "mobile (17)"; `ls frontend/src/pages/mobile/admin` shows 15 top-level page dirs (sub-routes likely account for the plan's 17). Kept the plan's numbers in the card since cards encode the approved plan.
3. **Dedup in card 15:** the common QA checklist already contains "Hard-refresh để nhận service worker mới", so the card-specific duplicate was dropped (repo copy-discipline rule).
4. File targets were verified against the live tree (e.g. `filter-chip-bar.tsx` / `pagination-controls.tsx` in `src/components/ui/`, 5 partner desktop + 4 partner mobile pages, 9 dead ui files all present).

No git commit performed. Nothing outside `kanban/TODO/` was touched except this report.
