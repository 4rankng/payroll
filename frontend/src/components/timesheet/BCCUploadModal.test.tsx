import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { PartnerImportFile } from '@/types/api/timesheet.types';
import { BCCUploadModal } from './BCCUploadModal';

const { mockUseBCCUploadModal } = vi.hoisted(() => ({
  mockUseBCCUploadModal: vi.fn(),
}));

vi.mock('@/hooks/timesheet/useBCCUploadModal', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/hooks/timesheet/useBCCUploadModal')
  >()),
  useBCCUploadModal: mockUseBCCUploadModal,
}));

const baseResult: PartnerImportFile = {
  id: 1,
  project_id: 7,
  uploaded_by: 2,
  original_name: 'bcc.xlsx',
  for_month: '2026-07',
  status: 'completed',
  total_rows: 3,
  created_count: 2,
  skipped_count: 1,
  error_count: 0,
  error_detail: null,
  created_at: '2026-07-25T00:00:00Z',
};

function mockModalState(
  result: PartnerImportFile | null,
  allowFlexibleEmployeeImport = false,
  overrides: Record<string, unknown> = {},
) {
  mockUseBCCUploadModal.mockReturnValue({
    file: null,
    result,
    selectedProjectId: '7',
    selectedMonth: '2026-07',
    isDragging: false,
    isPending: false,
    isImportActive: false,
    needsProjectSelect: false,
    hasProject: true,
	canUpload: false,
	hintText: '',
	isReady: false,
	includeFlexibleEmployees: false,
	allowFlexibleEmployeeImport,
    setSelectedProjectId: vi.fn(),
	setSelectedMonth: vi.fn(),
	handleIncludeFlexibleEmployeesChange: vi.fn(),
    handleFileChange: vi.fn(),
    handleUpload: vi.fn(),
    handleClose: vi.fn(),
    handleDragOver: vi.fn(),
    handleDragLeave: vi.fn(),
    handleDrop: vi.fn(),
    handleReset: vi.fn(),
    handleRemoveFile: vi.fn(),
    ...overrides,
  });
}

function renderModal(allowFlexibleEmployeeImport = false) {
  render(
    <BCCUploadModal
      open
      onClose={vi.fn()}
      projectId={7}
      projects={[{ id: 7, name: 'Dự án 7' }]}
		allowFlexibleEmployeeImport={allowFlexibleEmployeeImport}
    />,
  );
}

describe('BCCUploadModal result states', () => {
  beforeEach(() => {
    mockUseBCCUploadModal.mockReset();
	});

	it('shows the safe flexible-pay import option only when an admin enables it', () => {
		mockModalState(null, true);
		renderModal(true);

		const checkbox = screen.getByRole('checkbox', {
			name: /import lương linh hoạt/i,
		});
		fireEvent.click(checkbox);
		expect(mockUseBCCUploadModal.mock.results[0]?.value.handleIncludeFlexibleEmployeesChange)
			.toHaveBeenCalledWith(true);
		expect(screen.getByText('Chỉ tạo ngày chưa có.')).toBeInTheDocument();
	});

  it('uses a continuous, scrollable dialog layout for the upload form', () => {
    mockModalState(null);
    renderModal();

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveClass('gap-0', 'max-h-[calc(100dvh-2rem)]');
    expect(screen.getByText('Tệp bảng chấm công').parentElement?.parentElement).toHaveClass('overflow-y-auto');
  });

  it('keeps the light-header title visible instead of inheriting the dark-header default', () => {
    mockModalState(null);
    renderModal();

    expect(screen.getByRole('heading', { name: /Tải lên Bảng Chấm Công/ })).toHaveClass(
      'text-slate-900',
      'dark:text-slate-100',
    );
  });

  it('shows the project picker even when the page already has a project selected', () => {
    mockModalState(null, false, { needsProjectSelect: true });
    renderModal();

    expect(screen.getByText('Dự án áp dụng')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Dự án 7' })).toBeInTheDocument();
  });

  it('renders pure completion as success without an error state', () => {
    mockModalState(baseResult);
    renderModal();

    expect(screen.getByText('Nhập dữ liệu thành công')).toBeInTheDocument();
    expect(screen.getByText('Tạo mới')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('Bỏ qua')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
    expect(screen.queryByText('Nhập dữ liệu hoàn tất một phần')).not.toBeInTheDocument();
    expect(screen.queryByText('Nhập dữ liệu thất bại')).not.toBeInTheDocument();
  });

  it('renders completed jobs with errors as partial success with grouped details', () => {
    mockModalState({
      ...baseResult,
      error_count: 2,
      error_detail: JSON.stringify([
        {
          row: 2,
          employee: 'Nguyễn Văn An',
          reason: 'không thể tạo nhân viên CCCD 123: failed employee_id=91',
        },
        { row: 3, employee: 'Nguyễn Văn An', reason: 'Thiếu tên chủ tài khoản' },
      ]),
    });
    renderModal();

    expect(screen.getByText('Nhập dữ liệu hoàn tất một phần')).toBeInTheDocument();
    expect(screen.getByText('Tạo mới')).toBeInTheDocument();
    expect(screen.getByText('Bỏ qua')).toBeInTheDocument();
    expect(screen.getByText('Lỗi')).toBeInTheDocument();
    expect(screen.queryByText('Nhập dữ liệu thành công')).not.toBeInTheDocument();
    expect(screen.queryByText('Nhập dữ liệu thất bại')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Xem 2 lỗi cần xử lý' }));
    expect(screen.getByText('Nguyễn Văn An')).toBeInTheDocument();
    expect(screen.getByText('Không thể tạo hồ sơ nhân viên')).toBeInTheDocument();
    expect(screen.getByText('Thiếu dữ liệu bắt buộc')).toBeInTheDocument();
    expect(screen.getByText(/Dòng 2:/)).toBeInTheDocument();
    expect(screen.getByText(/Dòng 3:/)).toBeInTheDocument();
    expect(screen.queryByText(/failed|employee_id|CCCD 123/i)).not.toBeInTheDocument();
  });

  it('keeps total failure visually and semantically distinct', () => {
    mockModalState({
      ...baseResult,
      status: 'failed',
      created_count: 0,
      skipped_count: 0,
      error_count: 1,
      error_detail: JSON.stringify([
        { row: 2, employee: 'Trần Thị Bình', reason: 'Ngày chấm công không hợp lệ' },
      ]),
    });
    renderModal();

    expect(screen.getByRole('alert')).toHaveTextContent('Nhập dữ liệu thất bại');
    expect(screen.queryByText('Nhập dữ liệu thành công')).not.toBeInTheDocument();
    expect(screen.queryByText('Nhập dữ liệu hoàn tất một phần')).not.toBeInTheDocument();
    expect(
      screen.getByText('Phát hiện 1 lỗi khi xử lý tệp. Vui lòng kiểm tra chi tiết bên dưới.'),
    ).toBeInTheDocument();
  });
});
