import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, vi } from 'vitest';
import { projectService } from '@/services/api/project.service';
import { useAllProjects } from './useProjects';
import type { Project } from '@/types/api/project.types';

vi.mock('@/lib/auth', () => ({
  authManager: { getUserRole: vi.fn(() => 'admin') },
}));

vi.mock('@/services/api/project.service', () => ({
  projectService: { getProjects: vi.fn() },
}));

describe('useAllProjects', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads every backend page when the project catalogue exceeds 100 rows', async () => {
    const project = (id: number): Project => ({
      id,
      name: `Dự án ${id}`,
      code: `DA${id}`,
      client_name: 'Công ty kiểm thử',
      start_date: '2026-01-01',
      end_date: null,
      employee_count: 0,
      weekly_salary_employee_count: 0,
      monthly_salary_employee_count: 0,
      status: 'active',
      created_by: 1,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    });
    const firstPage = Array.from({ length: 100 }, (_, index) => project(index + 1));
    const secondPage = Array.from({ length: 5 }, (_, index) => project(index + 101));
    vi.mocked(projectService.getProjects)
      .mockResolvedValueOnce({
        status: 'success',
        data: firstPage,
        pagination: { page: 1, pageSize: 100, totalPages: 2, totalRecords: 105 },
      })
      .mockResolvedValueOnce({
        status: 'success',
        data: secondPage,
        pagination: { page: 2, pageSize: 100, totalPages: 2, totalRecords: 105 },
      });

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useAllProjects(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const projects = result.current.data;

    expect(projects).toHaveLength(105);
    expect(projects.at(-1)?.id).toBe(105);
    expect(projectService.getProjects).toHaveBeenNthCalledWith(1, {
      page: 1,
      pageSize: 100,
      sortBy: 'name',
      sortOrder: 'asc',
    });
    expect(projectService.getProjects).toHaveBeenNthCalledWith(2, {
      page: 2,
      pageSize: 100,
      sortBy: 'name',
      sortOrder: 'asc',
    });
  });
});
