import { describe, expect, it, vi } from 'vitest';
import { authManager } from '@/lib/auth';
import { canAccess, validateModalPermissionSync } from '@/lib/modal-permissions';
import { getModalMetadata as getRouterMetadata } from '@/constants/modalRegistry';
import { getModalMetadata } from '@/constants/modalRegistry/index';

vi.mock('@/lib/auth', () => ({ authManager: { getUserRole: vi.fn() } }));

describe('self-account modal permissions', () => {
  it.each(['admin', 'partner', 'adv_partner', 'accountant', 'employee'] as const)(
    'allows %s to open their own password form', (role) => {
      vi.mocked(authManager.getUserRole).mockReturnValue(role);
      expect(validateModalPermissionSync('change_password')).toBe(true);
      expect(getRouterMetadata('change_password')?.roles).toContain(role);
      expect(getModalMetadata('change_password').roles).toContain(role);
    },
  );

  it.each(['admin', 'partner'] as const)('allows the existing %s self-profile modal', (role) => {
    vi.mocked(authManager.getUserRole).mockReturnValue(role);
    expect(validateModalPermissionSync('user_profile')).toBe(true);
  });

  it.each(['adv_partner', 'accountant', 'employee'] as const)(
    'does not grant %s user administration or new profile routes', (role) => {
      vi.mocked(authManager.getUserRole).mockReturnValue(role);
      expect(canAccess('update', 'User')).toBe(false);
      expect(canAccess('read', 'User')).toBe(false);
      for (const modal of ['add_user', 'edit_user', 'reset_password', 'user_details', 'user_profile']) {
        expect(validateModalPermissionSync(modal)).toBe(false);
      }
    },
  );

  it('requires authentication and keeps both registries aligned', () => {
    vi.mocked(authManager.getUserRole).mockReturnValue(null);
    expect(validateModalPermissionSync('change_password')).toBe(false);
    for (const modal of ['change_password', 'user_profile'] as const) {
      const metadata = getModalMetadata(modal);
      expect(metadata.roles).toEqual(getRouterMetadata(modal)?.roles);
      expect(metadata.permissions).toEqual({ action: modal === 'change_password' ? 'update' : 'read', subject: 'SelfAccount' });
    }
  });
});
