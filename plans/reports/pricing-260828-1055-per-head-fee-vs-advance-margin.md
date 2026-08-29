# Per-Head Service Fee vs Current Advance-Fee Margin — Pricing Analysis

**Date:** 2026-08-28 · **Source:** prod DB (tingting.vip), read-only SELECTs, project LGD (id 58) — the only project with advance history · **Context:** Hải Anh (Vfic) asked Frank to compute the per-worker service fee that replaces the current % fee on salary advances, net of cost of funds ("lãi suất huy động vốn"), Hilex-proposal style.

## 1. What the current model actually earns (Aug 2026 actuals)

| Metric | Value |
|---|---|
| Advance requests (COMPLETED) | 147 |
| Workers who took an advance | 103 of 280 enrolled (37% adoption) |
| Gross advanced | 611,565,000 ₫ |
| Fee collected | 8,838,120 ₫ |
| **Blended fee rate** | **1.445%** (not 2% — schedule since 17/05: 2% ≤ 3.5M, 1.3% > 3.5M, min 10k; avg request 4.16M) |
| Provider transfer cost | ~566k ₫ (147 × 3,850 ₫/transfer) |

Monthly trend (fee / enrolled / users): Apr 1.29M/9/9 → May 1.51M/21/15 → Jun 1.82M/27/19 → Jul 2.10M/70/27 → **Aug 8.84M/280/103**. Scale jumped ~4× in Aug (bulk onboarding); per-head fee stabilized: Jul 29.9k, Aug 31.6k.

## 2. Cost of funds — CORRECTED 28/08: real loan interest is 18.46M ₫/month, not usage-based

The first pass modeled funding cost as advance-usage vnd-days × 8%/yr (~1.4M/month). **Wrong** — the company carries standing term loans (table `loans` / `loan_repayment_schedules`, loans admin page):

| Loan | Principal | Rate | Starts | Interest/month |
|---|---|---|---|---|
| LOAN-2025-001 | 500M | 7.5% | 02/2026 | 3,125,000 |
| LOAN-2026-001 | 500M | 8% | 27/08/2026 | 3,333,333 |
| LOAN-2026-002 | 800M | 8% | 01/09/2026 | 5,333,333 |
| LOAN-2026-003 | 1,000M | 8% | 03/09/2026 | 6,666,667 |
| **Total** | **2.8B ₫** | ~7.9% avg | | **18,458,333 ₫/month** (bullet principal due 07–08/2027) |

Historical interest actually paid: 3.13M/month (02–07) → 6.46M (08) → 18.46M run-rate from 09/2026. The loans fund the whole operation (LGD advances + weekly payroll float ~1.74B/month), per user confirmation.

**Consequence:** current fee revenue (8.84M) covers only 48% of interest → the business is ~**−10.2M ₫/month underwater** before ops costs. Break-even revenue = 18.46M interest + 0.57M transfers = **19.02M ₫/month** = **28,956 ₫/head on 657**.

## 3. The per-head fee ("charge theo đầu công nhân"), Aug run-rate

**Corrected denominator — the whole workforce the platform is actively paying now** (per user, 28/08): not just the LGD FlexPay list.

| Segment | Workers |
|---|---|
| Weekly timesheet projects, active last 5 weeks (21 projects) | 377 |
| — biggest: EVA 105, LGDISPLAY 61, Thái Bình Dương 28, BUMHAN 26, PQC 21, Supra Masan 18, Lear 18, EPE 17, Hilex 16, CBS 14 … | |
| FlexPay enrolled (LGD, Aug list) | 280 |
| **Total (zero overlap between the two sets)** | **657** |

| Scenario | Monthly equiv. | / all 657 heads | / LGD-only 280 | / 103 active users |
|---|---|---|---|---|
| A. Replace full fee revenue (D1) | 8.84M | **13.5k ₫** | 31.6k ₫ | 85.8k ₫ |
| B. Fee − real loan interest (D2) | **−9.62M (LOSS)** | — | — | — |
| C. Fee − interest − transfers (D3) | **−10.19M (LOSS)** | — | — | — |
| **D4. BREAK-EVEN (interest + transfers)** | **19.02M** | **29.0k ₫** | 67.9k ₫ | 184.7k ₫ |

**Revised recommendation (post-correction):**
- **All-in flat: 35,000 ₫/head/month on 657** (= 23.0M, break-even +21%) — quote anchor.
- **Split (preferred): LGD 45,000 ₫/head + weekly 20,000 ₫/head** = 20.14M (break-even +6%): LGD 45k ≈ replacement 31.6k + cost share; weekly 20k ≈ 0.43% of its 1.74B payroll flow and reflects that the loans mostly fund the weekly float.
- **Floor: 29,000 ₫/head** ≈ exact break-even. Below this the service loses money every month.
- Old numbers (15k all-in / LGD 30k + weekly 10k) are **below water** — superseded.

## 4. Negotiation / legal framing

- Legal driver (Hải Anh's point): % fee charged to workers on advanced salary looks like unlicensed money lending (Decree 144 / Civil Code caps). A flat per-head B2B service fee invoiced to Vfic = clean service contract, no lending exposure, workers pay nothing.
- Frank's "2% profit" anchor is nominal; actual collected is 1.445% blended (tiering) and it no longer even covers loan interest — anchor talks on the **19.0M break-even**, not on 2%.
- Per-head fee shifts volume risk to the service side: at 37% FlexPay adoption 15k/all-heads covers it; if adoption → 100% with same avg advance (4.16M), the equivalent revenue doubles. Mitigate with either a volume band, a small per-request component, or a quarterly rate review.
- Spreading over all 657 heads (not just 280 FlexPay) also prices the weekly payroll service itself — timesheets, bulk transfers, sao kê reconciliation for 21 projects — which today carries no separate fee in the system.
- Hilex template: same structure for the next customer — per-head monthly fee on the enrolled list.

## 5. Unresolved questions

1. ~~Actual funding rate~~ RESOLVED 28/08: real loans = 2.8B ₫ principal, ~7.9% avg, 18.46M ₫/month interest (see §2).
2. Whether the service-fee switch also changes who funds the float (affects whether interest shrinks in future).
3. Real OnePay per-transfer tariff on the current contract (3,850 ₫ assumed from settings).
4. Aug is partial (through day 28); day 29–31 volume may nudge Aug slightly higher.
5. Bullet principal 2.8B ₫ due 07–08/2027 needs a refinancing/repayment plan — the per-head fee at break-even covers interest only, not principal amortization.
