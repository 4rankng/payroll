import { describe, expect, it } from 'vitest';

import { getConsoleKicker } from './console';

describe('getConsoleKicker', () => {
  it('defaults staff dashboards and ops routes to VẬN HÀNH', () => {
    expect(getConsoleKicker('/admin/dashboard')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/projects')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/timesheet')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/approvals')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/settings')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/system-health')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/audit-log')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/zns-tracking')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/partner/dashboard')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/partner/timesheet')).toBe('VẬN HÀNH');
  });

  it('maps people routes to NHÂN SỰ', () => {
    expect(getConsoleKicker('/admin/users')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/admin/users/42')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/admin/employees')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/admin/employees/')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/partner/employees')).toBe('NHÂN SỰ');
  });

  it('maps money routes to TÀI CHÍNH', () => {
    expect(getConsoleKicker('/admin/ledger')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/payment-history')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/loans')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/loans/lenders')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/advance-payments/check-in-settings')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/wallet')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/partner/timesheet/payment-history')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/adv-partner/advance-payments')).toBe('TÀI CHÍNH');
  });

  it('maps the accountant workspace to KẾ TOÁN', () => {
    expect(getConsoleKicker('/accountant')).toBe('KẾ TOÁN');
  });

  it('does not let a prefix capture a longer sibling segment', () => {
    expect(getConsoleKicker('/admin/users-archive')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/admin/employeesx')).toBe('VẬN HÀNH');
  });

  it('falls back to VẬN HÀNH for non-staff and unknown paths', () => {
    expect(getConsoleKicker('/login')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/')).toBe('VẬN HÀNH');
    expect(getConsoleKicker('/definitely-not-a-route')).toBe('VẬN HÀNH');
  });

  it('ignores query strings and trailing slashes', () => {
    expect(getConsoleKicker('/admin/ledger?tab=pending')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/accountant/')).toBe('KẾ TOÁN');
  });
});
