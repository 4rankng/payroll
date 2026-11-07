/**
 * Console kickers — the mono domain label shown in every staff page header
 * (toto.sg mission-control port). Pure path→domain mapping; no router
 * dependency so it is testable and usable from any layout.
 */

export type ConsoleKicker = 'VẬN HÀNH' | 'NHÂN SỰ' | 'TÀI CHÍNH' | 'KẾ TOÁN';

const DEFAULT_KICKER: ConsoleKicker = 'VẬN HÀNH';

/** Longest-prefix routes first is not required — prefixes are disjoint, but
 *  matching is segment-exact (`/admin/users` never captures
 *  `/admin/users-archive`). */
const ROUTE_KICKERS: ReadonlyArray<{ prefix: string; kicker: ConsoleKicker }> = [
  { prefix: '/accountant', kicker: 'KẾ TOÁN' },

  // NHÂN SỰ — people records and accounts.
  { prefix: '/admin/users', kicker: 'NHÂN SỰ' },
  { prefix: '/admin/employees', kicker: 'NHÂN SỰ' },

  // TÀI CHÍNH — money movement.
  { prefix: '/admin/ledger', kicker: 'TÀI CHÍNH' },
  { prefix: '/admin/payment-history', kicker: 'TÀI CHÍNH' },
  { prefix: '/admin/loans', kicker: 'TÀI CHÍNH' },
  { prefix: '/admin/advance-payments', kicker: 'TÀI CHÍNH' },
  { prefix: '/admin/wallet', kicker: 'TÀI CHÍNH' },
  { prefix: '/partner/timesheet/payment-history', kicker: 'TÀI CHÍNH' },
  { prefix: '/adv-partner', kicker: 'TÀI CHÍNH' },
];

export const getConsoleKicker = (pathname: string): ConsoleKicker => {
  const path = pathname.split('?')[0].replace(/\/+$/, '') || '/';
  const match = ROUTE_KICKERS.find(
    (route) => path === route.prefix || path.startsWith(`${route.prefix}/`),
  );
  return match?.kicker ?? DEFAULT_KICKER;
};
