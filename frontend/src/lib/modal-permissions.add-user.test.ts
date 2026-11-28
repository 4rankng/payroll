import { describe, expect, it } from 'vitest';
import { getModalMetadata as getRouterMetadata, MODAL_REGISTRY } from '@/constants/modalRegistry';
import { getModalMetadata } from '@/constants/modalRegistry/index';
import { MODAL_PERMISSIONS } from '@/lib/modal-permissions';
import { modalConfig as addUserSheetConfig } from '@/components/sheets/AddUserSheet';

/**
 * Only admins manage users (canManageUsers). The URL-driven ModalRouter gates
 * ?modal=add_user on registry roles alone (no CASL pass), so every registry
 * copy must agree on the admin-only role list or a non-admin can still deep
 * link straight into the add-user sheet.
 */
describe('add_user is strictly admin-gated across every registry copy', () => {
  const ADMIN_ONLY = ['admin'];

  it('locks both modal registries to admin', () => {
    expect(MODAL_REGISTRY.add_user.roles).toEqual(ADMIN_ONLY);
    expect(getRouterMetadata('add_user')?.roles).toEqual(ADMIN_ONLY);
    expect(getModalMetadata('add_user').roles).toEqual(ADMIN_ONLY);
  });

  it('locks the manual permission registry to admin', () => {
    expect(MODAL_PERMISSIONS.add_user.roles).toEqual(ADMIN_ONLY);
  });

  it('locks the sheet modal config to admin', () => {
    expect(addUserSheetConfig.permissions.roles).toEqual(ADMIN_ONLY);
  });
});
