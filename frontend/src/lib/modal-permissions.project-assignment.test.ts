import { describe, expect, it, vi } from 'vitest';
import { authManager, type AppRole } from '@/lib/auth';
import { validateModalPermissionSync } from '@/lib/modal-permissions';
import { getModalMetadata as getRouterMetadata } from '@/constants/modalRegistry';
import { getModalMetadata, getModalSchema } from '@/constants/modalRegistry/index';

vi.mock('@/lib/auth', () => ({
  authManager: { getUserRole: vi.fn() },
}));

describe('add employee to project modal permissions', () => {
  it.each(['admin', 'partner'] as const)('allows the existing %s assignment workflow', (role) => {
    vi.mocked(authManager.getUserRole).mockReturnValue(role);
    expect(validateModalPermissionSync('add_employee_to_project')).toBe(true);
    expect(validateModalPermissionSync('add_employee_to_project', { projectId: '123' })).toBe(true);
  });

  it.each(['employee', 'adv_partner', 'accountant', null] as Array<AppRole | null>)(
    'denies assignment access for %s',
    (role) => {
      vi.mocked(authManager.getUserRole).mockReturnValue(role);
      expect(validateModalPermissionSync('add_employee_to_project', { projectId: '123' })).toBe(false);
    },
  );

  it('keeps router and domain metadata aligned and requires a project ID', () => {
    const metadata = getModalMetadata('add_employee_to_project');
    expect(metadata.roles).toEqual(getRouterMetadata('add_employee_to_project')?.roles);
    expect(metadata.permissions).toEqual({ action: 'update', subject: 'Project' });
    expect(getModalSchema('add_employee_to_project').safeParse({ projectId: '123' }).success).toBe(true);
    expect(getModalSchema('add_employee_to_project').safeParse({}).success).toBe(false);
  });
});
