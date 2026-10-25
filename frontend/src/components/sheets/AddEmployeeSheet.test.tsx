import type { ReactNode } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { DuplicateCheckMatch, DuplicateCheckResponse } from '@/types/api/employee.types';
import AddEmployeeSheet from './AddEmployeeSheet';

const mocks = vi.hoisted(() => ({
  createEmployee: vi.fn(),
  mobileSetting: undefined as { value: string | null } | undefined,
  duplicateResult: undefined as DuplicateCheckResponse | undefined,
}));

vi.mock('@/hooks/api/useEmployees', () => ({
  useCreateEmployee: () => ({ isPending: false, mutateAsync: mocks.createEmployee }),
  useDuplicateCheck: () => ({ data: mocks.duplicateResult }),
  useRequestEmployeeAccess: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));
vi.mock('@/hooks/api/useSettings', () => ({
  useMultipleSettings: () => ({ data: [mocks.mobileSetting] }),
}));
vi.mock('@/lib/auth', () => ({ authManager: { getUserRole: () => 'admin' } }));
vi.mock('@/hooks/useModalNavigation', () => ({ useModalNavigation: () => ({ closeModal: vi.fn() }) }));
vi.mock('@/components/ui/sonner', () => ({ toast: vi.fn() }));
vi.mock('@/components/ui/bank-selector', () => ({ BankSelector: () => <div /> }));
vi.mock('@/components/sheets/templates/SlideSheetTemplate', () => ({
  SlideSheetTemplate: ({ children, footer }: { children: ReactNode; footer: ReactNode }) => (
    <div>
      {children}
      {footer}
    </div>
  ),
}));

const mobileMatch: DuplicateCheckMatch = {
  id: 7,
  fullname: 'Trần Thị B',
  mobile_masked: '0901***456',
  current_project_names: [],
  created_by_name: 'Admin Test',
  created_at: '2026-01-01T00:00:00Z',
  matched_on: ['mobile'],
};

const emailMatch: DuplicateCheckMatch = {
  id: 8,
  fullname: 'Lê Văn C',
  email_masked: 'l***@example.com',
  current_project_names: [],
  created_by_name: 'Admin Test',
  created_at: '2026-01-02T00:00:00Z',
  matched_on: ['email'],
};

const cccdMatch: DuplicateCheckMatch = {
  id: 9,
  fullname: 'Phạm Thị D',
  cccd_masked: '0012***7890',
  current_project_names: [],
  created_by_name: 'Admin Test',
  created_at: '2026-01-03T00:00:00Z',
  matched_on: ['cccd'],
};

function renderSheet() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <AddEmployeeSheet isOpen onClose={vi.fn()} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function fillRequiredIdentity() {
  fireEvent.change(screen.getByLabelText(/Họ và tên/), {
    target: { value: 'Nguyễn Văn A' },
  });
  fireEvent.change(screen.getByLabelText(/Số CCCD/), {
    target: { value: '001234567890' },
  });
}

describe('AddEmployeeSheet mobile gating', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.mobileSetting = undefined;
    mocks.duplicateResult = undefined;
    mocks.createEmployee.mockResolvedValue({ data: undefined });
  });

  it('requires a phone number when the mobile policy is on', () => {
    mocks.mobileSetting = { value: 'true' };
    renderSheet();

    fillRequiredIdentity();
    fireEvent.click(screen.getByRole('button', { name: /Thêm nhân viên/ }));

    expect(screen.getByRole('alert')).toHaveTextContent('Số điện thoại là bắt buộc');
    expect(mocks.createEmployee).not.toHaveBeenCalled();
  });

  it('marks the phone field required and sends the phone number in the payload when the policy is on', () => {
    mocks.mobileSetting = { value: 'true' };
    renderSheet();

    fillRequiredIdentity();
    fireEvent.change(screen.getByLabelText(/Số điện thoại/), {
      target: { value: '0901234567' },
    });
    // The Field renders the required asterisk on the label.
    expect(screen.getByText('Số điện thoại').textContent).toBe('Số điện thoại*');

    fireEvent.click(screen.getByRole('button', { name: /Thêm nhân viên/ }));

    expect(mocks.createEmployee).toHaveBeenCalledWith({
      fullname: 'Nguyễn Văn A',
      cccd: '001234567890',
      mobile: '0901234567',
    });
  });

  it('keeps the phone number optional and out of the payload when the policy is off', () => {
    renderSheet();

    fillRequiredIdentity();
    expect(screen.getByText('Số điện thoại').textContent).toBe('Số điện thoại');

    fireEvent.click(screen.getByRole('button', { name: /Thêm nhân viên/ }));

    expect(mocks.createEmployee).toHaveBeenCalledWith({
      fullname: 'Nguyễn Văn A',
      cccd: '001234567890',
    });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('hard-blocks a duplicate phone number and disables save', () => {
    mocks.duplicateResult = { has_duplicates: true, data: [mobileMatch] };
    renderSheet();

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Số điện thoại này đã thuộc về nhân viên khác',
    );
    expect(screen.getByRole('button', { name: /Thêm nhân viên/ })).toBeDisabled();
  });

  it('keeps a duplicate email as a soft warning with save enabled', () => {
    mocks.duplicateResult = { has_duplicates: true, data: [emailMatch] };
    renderSheet();

    expect(screen.getByText('Thông tin liên hệ này đã được sử dụng')).toBeInTheDocument();
    expect(screen.queryByText('Số điện thoại này đã thuộc về nhân viên khác')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Thêm nhân viên/ })).toBeEnabled();
  });

  it('keeps the duplicate CCCD block unchanged', () => {
    mocks.duplicateResult = { has_duplicates: true, data: [cccdMatch] };
    renderSheet();

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Nhân viên với CCCD này đã tồn tại trong hệ thống',
    );
    expect(screen.getByRole('button', { name: /Thêm nhân viên/ })).toBeDisabled();
  });
});
