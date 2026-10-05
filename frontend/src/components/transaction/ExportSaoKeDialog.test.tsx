import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ExportSaoKeDialog } from './ExportSaoKeDialog';

const mocks = vi.hoisted(() => ({
  useProjects: vi.fn(),
  refetch: vi.fn(),
  getUserRole: vi.fn(() => 'partner' as string | null),
}));

vi.mock('@/hooks/api/useProjects', () => ({
  useProjects: (...args: unknown[]) => mocks.useProjects(...args),
}));

vi.mock('@/services/api/client', () => ({
  apiClient: { download: vi.fn().mockResolvedValue(undefined) },
  buildQueryString: () => '',
}));

vi.mock('@/lib/auth', () => ({
  authManager: { getUserRole: () => mocks.getUserRole() },
}));

const project = (id: number, name: string, code: string) => ({
  id,
  name,
  code,
  client_name: name,
  status: 'active',
});

const settled = (overrides: Partial<Record<string, unknown>> = {}) => ({
  data: { status: 'success', data: [project(67, 'TEV', 'CX002')] },
  isPending: false,
  isFetching: false,
  isError: false,
  refetch: mocks.refetch,
  ...overrides,
});

describe('ExportSaoKeDialog', () => {
  beforeEach(() => {
    mocks.useProjects.mockReset();
    mocks.refetch.mockReset();
    mocks.getUserRole.mockReturnValue('partner');
  });

  it('does not fetch projects while the dialog is closed', () => {
    mocks.useProjects.mockReturnValue(settled());

    render(<ExportSaoKeDialog open={false} onOpenChange={vi.fn()} />);

    expect(mocks.useProjects).toHaveBeenCalledWith(
      { status: 'active', pageSize: 100 },
      { enabled: false },
    );
  });

  it('shows a skeleton instead of a fake empty list while loading', () => {
    mocks.useProjects.mockReturnValue(settled({ isPending: true, data: undefined }));

    render(<ExportSaoKeDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText('Đang tải dự án…')).toBeInTheDocument();
    // Regression: this used to render "Tất cả dự án (0)" + "Không tìm thấy dự án"
    // during the whole fetch, which read as "TEV does not exist".
    expect(screen.queryByText(/Tất cả dự án \(/)).not.toBeInTheDocument();
    expect(screen.queryByText('Không tìm thấy dự án')).not.toBeInTheDocument();
  });

  it('lists the partner projects once loaded', () => {
    mocks.useProjects.mockReturnValue(
      settled({
        data: {
          status: 'success',
          data: [project(67, 'TEV', 'CX002'), project(77, 'Georim', 'GX001')],
        },
      }),
    );

    render(<ExportSaoKeDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByRole('button', { name: /TEV/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Georim/ })).toBeInTheDocument();
    expect(screen.getByText('Tất cả dự án (2)')).toBeInTheDocument();
  });

  it('filters by the typed project name', () => {
    mocks.useProjects.mockReturnValue(
      settled({
        data: {
          status: 'success',
          data: [project(67, 'TEV', 'CX002'), project(77, 'Georim', 'GX001')],
        },
      }),
    );

    render(<ExportSaoKeDialog open onOpenChange={vi.fn()} />);

    fireEvent.change(screen.getByPlaceholderText('Tìm dự án...'), {
      target: { value: 'TEV' },
    });

    expect(screen.getByRole('button', { name: /TEV/ })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Georim/ })).not.toBeInTheDocument();
  });

  it('offers a retry when the projects request fails', () => {
    mocks.useProjects.mockReturnValue(
      settled({ data: undefined, isError: true, isPending: false }),
    );

    render(<ExportSaoKeDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText('Không tải được danh sách dự án.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Thử lại/ }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });

  it('distinguishes an account with no granted projects from a failed search', () => {
    mocks.useProjects.mockReturnValue(
      settled({ data: { status: 'success', data: [] } }),
    );

    render(<ExportSaoKeDialog open onOpenChange={vi.fn()} />);

    expect(
      screen.getByText('Chưa có dự án nào được cấp cho tài khoản này'),
    ).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('Tìm dự án...'), {
      target: { value: 'TEV' },
    });

    expect(screen.getByText('Không tìm thấy dự án')).toBeInTheDocument();
  });

  it('explains instead of showing an empty list when the role cannot pick projects', () => {
    mocks.getUserRole.mockReturnValue('employee');
    mocks.useProjects.mockReturnValue(settled());

    render(<ExportSaoKeDialog open onOpenChange={vi.fn()} />);

    expect(mocks.useProjects).toHaveBeenCalledWith(
      { status: 'active', pageSize: 100 },
      { enabled: false },
    );
    expect(screen.getByText('Bạn không có quyền xem danh sách dự án.')).toBeInTheDocument();
  });
});
