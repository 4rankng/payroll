# Project Roadmap

Direction and priorities inferred from recent commit history and codebase signals. Sections are marked as **inferred** or **stated** based on evidence.

> Last analyzed: 2026-07-19. This is a snapshot, not a commitment. The repo has no formal roadmap file.

## Active Development Areas (Inferred from Recent Work)

### 1. Attendance & Check-In/Out (highest velocity -- 26 commits since June)

Most actively developed feature. Recent work includes:

- Geofence validation for both check-in and check-out (accuracy gate, distinct gates per action)
- Admin review workflow for device-GPS-failed check-ins
- Auto-rejection when checkout window closes without check-out
- Checkout window extension (K+1h to K+3h)
- Check-in health dashboard with drill-down (successful/failed attempt breakdown)
- Location map component with satellite layer
- Concurrency-safe check-in handling
- Mobile GPS acquisition window widening

**Signal:** This area dominates recent commits and has the most test coverage additions.

### 2. Payment Disbursement & Provider Integration (23 commits)

- OnePay fee report import functionality
- Disbursement fee charged only at transfer execution (not preflight)
- Tiered fee structure support
- Bulk transfer improvements (9Pay batch execution, transaction creation)
- IPN processing improvements
- OnePay IPN webhook IP whitelist

### 3. Security Hardening (significant effort)

- SQL injection fixes across ~20 list repositories (ORDER BY sanitization)
- Path traversal fixes
- Adv_partner authorization gap closure
- Production server kernel update (.23 to .37)
- fail2ban + sshd hardening
- UFW firewall (22/80/443 only)
- MySQL and Redis restart policy configuration

### 4. Mobile UI Refresh (ongoing)

- Mobile admin pages refined
- Desktop advance payment filter labels updated
- Employee self-service mobile shell unified across regular, flexible, and check-in-enabled variants
- Employee profile failure/loading states now use non-interactive mobile chrome with guarded retry and session-safe cache clearing
- Mobile-first attendance components now reserve measured toolbar height for safe areas and large-text layouts
- Employee advance-request cancellation and attendance recovery flows hardened against rapid repeat actions
- Responsive component variants (desktop/mobile table columns)

### 5. Notifications & Communication

- Payroll report (sao ke) email delivery with idempotency
- Web push notification infrastructure (PWA, VAPID)
- Auth session sync improvements

## Stabilizing / Maintenance Areas

### Testing

- Attendance service tests expanded significantly (auto-reject, admin review, concurrency, geofence)
- Integration test suite exists at `backend/tests/`
- Frontend unit coverage is still light, but employee mobile Playwright coverage now exists with synthetic fixtures and unexpected-request failure gates

### Code Quality

- Very low TODO/FIXME density (2 in backend, 5 in frontend across entire codebase)
- CI/CD pipeline added (GitHub Actions)
- Golangci-lint configuration maintained

## Infrastructure & DevOps

- Docker buildx local cache optimization (frontend)
- Dual-lockfile drift mitigation (yarn.lock + pnpm-lock.yaml coexistence)
- Demo server (1GB droplet) with 2GB swap requirement documented
- Adminer over SSH tunnel (no public exposure)
- OneDrive backup/restore pipeline

## Areas with Less Recent Activity (Possible Future Focus)

| Area | Notes |
|------|-------|
| Timesheet bulk operations | Core work appears stable; 7 commits since June, mostly fixes |
| Loan/Lender management | Existing functionality, no recent major changes |
| Partner features | Moderate activity (3 commits); scoping improvements |
| Payrate configuration | Minor activity (2 commits) |
| Dashboard | Minor activity (2 commits); recent health dashboard addition |
| PWA | Infrastructure in place (2 commits); no recent enhancement |

## Notable Technical Debt Signals

- **Dual lockfile**: Frontend has both `yarn.lock` (Docker builds) and `pnpm-lock.yaml` (local dev). Requires manual sync.
- **Migration application**: `schema_migrations` table reported empty on some deployments; manual DDL application documented as workaround.
- **Demo server resources**: 1GB droplet requires 2GB swap; OOM risk documented.
- **Frontend test coverage**: Unit/integration coverage on the frontend is still thin, even though a synthetic employee mobile Playwright suite is now checked in.
