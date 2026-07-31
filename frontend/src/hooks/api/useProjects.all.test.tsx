import { renderHook } from '@testing-library/react';
import { useQuery } from '@tanstack/react-query';
import { beforeEach, vi } from 'vitest';
import { projectService } from '@/services/api/project.service';
import { useAllProjects } from './useProjects';

vi.mock('@tanstack/react-query', () => ({
  useMutation: vi.fn(),
  useQuery: vi.fn(),
  useQueryClient: vi.fn(),
}));

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
    const firstPage = Array.from({ length: 100 }, (_, index) => ({
      id: index + 1,
      name: `Dự án ${index + 1}`,
      code: `DA${index + 1}`,
    }));
    const secondPage = Array.from({ length: 5 }, (_, index) => ({
      id: index + 101,
      name: `Dự án ${index + 101}`,
      code: `DA${index + 101}`,
    }));
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

    renderHook(() => useAllProjects());
    const options = vi.mocked(useQuery).mock.calls[0][0] as {
      queryFn: () => Promise<Array<{ id: number }>>;
    };
    const projects = await options.queryFn();

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
