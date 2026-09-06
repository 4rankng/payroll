import { describe, expect, it } from 'vitest';

import { ADMIN_MENU_ITEMS } from '@/components/AdminSidebar';
import { ADMIN_MORE_ITEMS } from './AdminLayout';

describe('Admin financial navigation', () => {
  it.each([
    ['desktop and drawer', ADMIN_MENU_ITEMS],
    ['mobile more menu', ADMIN_MORE_ITEMS],
  ])('keeps the ledger discoverable in %s navigation', (_name, items) => {
    expect(items.map((item) => item.path)).toEqual(
      expect.arrayContaining(['/admin/ledger']),
    );
  });

  it.each([
    ['desktop and drawer', ADMIN_MENU_ITEMS],
    ['mobile more menu', ADMIN_MORE_ITEMS],
  ])('retires the legacy transactions entry in %s navigation', (_name, items) => {
    expect(items.map((item) => item.path)).not.toContain('/admin/transactions');
  });
});
