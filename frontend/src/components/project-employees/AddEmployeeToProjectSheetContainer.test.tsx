import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AddEmployeeToProjectSheetContainer } from './AddEmployeeToProjectSheetContainer';
const mocks = vi.hoisted(() => ({ mobile: false, query: {} as Record<string, unknown>, refetch: vi.fn(), useProject: vi.fn() }));
vi.mock('@/hooks/useBreakpoint', () => ({ useIsMobile: () => mocks.mobile }));
vi.mock('@/hooks/api/useProjects', () => ({ useProject: (...args: unknown[]) => { mocks.useProject(...args); return mocks.query; } }));
vi.mock('./AddEmployeesToProject', () => ({ AddEmployeesToProject: ({ project }: { project: { id: number } }) => <p>Ready project {project.id}</p> }));
beforeEach(() => { vi.clearAllMocks(); mocks.query = { data: undefined, isLoading: true, isError: false, refetch: mocks.refetch }; });
describe.each([false, true])('add project employee mobile=%s', mobile => {
  it('waits for a cold project fetch before mounting dependent controls', () => {
    mocks.mobile = mobile;
    const { rerender } = render(<AddEmployeeToProjectSheetContainer isOpen onClose={vi.fn()} projectId="12" />);
    expect(screen.getByRole('status')).toHaveTextContent('Đang tải');
    expect(screen.queryByText('Ready project 12')).not.toBeInTheDocument();
    mocks.query = { ...mocks.query, isLoading: false, data: { id: 12 } };
    rerender(<AddEmployeeToProjectSheetContainer isOpen onClose={vi.fn()} projectId="12" />);
    expect(screen.getByText('Ready project 12')).toBeInTheDocument();
  });
  it('shows failed project reads and retries without opening assignment controls', () => {
    mocks.mobile = mobile;
    mocks.query = { ...mocks.query, isLoading: false, isError: true };
    render(<AddEmployeeToProjectSheetContainer isOpen onClose={vi.fn()} projectId="12" />);
    expect(screen.getByRole('alert')).toHaveTextContent('Không thể tải thông tin dự án');
    fireEvent.click(screen.getByRole('button', { name: 'Thử lại' }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });
  it.each([undefined, 'bad', '12bad', '0', '-1'])('rejects invalid deep link project %s without a project request', projectId => {
    mocks.mobile = mobile;
    render(<AddEmployeeToProjectSheetContainer isOpen onClose={vi.fn()} projectId={projectId} />);
    expect(mocks.useProject).toHaveBeenCalledWith(0, false);
    expect(screen.getByRole('alert')).toHaveTextContent('Liên kết thiếu mã dự án hợp lệ');
    expect(screen.queryByRole('button', { name: 'Thử lại' })).not.toBeInTheDocument();
  });
});
