<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# docs

## Purpose
Project documentation including Architecture Decision Records (ADRs), payment provider integration specs, workflow definitions, UI/UX review notes, and LGD (Lương Giờ Dự án / Project Hourly Wage) shift mapping files. Serves as the knowledge base for design decisions and external integration references.

## Key Files
| File | Description |
|------|-------------|
| `workflow_flexible_checkin_advance_payment.md` | Flexible employee check-in/out → earn quota → advance payment workflow spec |
| `adr/CONTEXT.md` | ADR index and context log |
| `adr/0001-advance-payment-partner-role.md` | ADR: Advance payment partner role design |
| `adr/flexi-checkin-spec.md` | Flexible check-in/check-out specification with shift validation rules |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `adr/` | Architecture Decision Records — numbered decisions with context |
| `LGD Files/` | LGD shift mapping Excel files from partner (TingTing project) |
| `onepay/` | OnePay integration docs, signature guide, test scripts |
| `reviews/` | UI/UX review notes and pending task tracking |
| `reviews/pending-tasks/` | Outstanding items from reviews |
| `superpowers/` | Implementation specs from superpowers planning |
| `timesheets-excel/` | BCC (Bảng Chấm Công) timesheet Excel templates and upload plan |

## For AI Agents

### Working In This Directory
- ADRs follow the Michael Nygard format (Context, Decision, Status, Consequences)
- LGD Excel files are reference data from the partner — read-only, do not modify
- OnePay docs include REST API signature guide and test scripts (Go + Python + curl)
- Workflow specs define the source of truth for feature behavior

### Testing Requirements
- No automated tests for docs — verify by reading the spec against implementation
- OnePay test scripts in `onepay/` can be run manually against sandbox

### Common Patterns
- Workflow docs use Vietnamese + English hybrid (Vietnamese for domain terms, English for technical terms)
- ADR numbering: `NNNN-title.md` starting from 0001
- LGD files named with date and project name

## Dependencies

### Internal
- `backend/internal/infra/disbursement/onepay/` — OnePay adapter implementation
- `backend/internal/app/services/advance_payment/` — FlexPay service (matches workflow spec)
- `frontend/src/components/payrates/` — Payrate editor (matches LGD shift structure)

### External
- OnePay Payout API documentation (PDF in `onepay/`)
- Partner-provided LGD shift files (Excel)
