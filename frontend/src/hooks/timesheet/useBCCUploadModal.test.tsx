import type { ChangeEvent, ReactNode } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { QueryKeys } from '@/lib/queryKeys';
import { timesheetService } from '@/services/api/timesheet.service';
import type { PartnerImportFile } from '@/types/api/timesheet.types';
import {
  buildFailureSummary,
  groupErrorsByEmployee,
  useBCCUploadModal,
} from './useBCCUploadModal';

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
}

function createImport(
  status: PartnerImportFile['status'],
): PartnerImportFile {
  return {
    id: 91,
    project_id: 12,
    uploaded_by: 7,
    original_name: 'bang-cham-cong.xlsx',
    for_month: '2026-07',
    status,
    total_rows: 2,
    created_count: status === 'completed' ? 1 : 0,
    skipped_count: 0,
    error_count: 1,
    error_detail: null,
    processed_at: status === 'pending' ? null : '2026-07-25T06:00:00Z',
    created_at: '2026-07-25T05:59:00Z',
  };
}

describe('useBCCUploadModal', () => {
  beforeEach(() => {
    vi.stubGlobal('crypto', {
      randomUUID: vi.fn(() => 'bcc-upload-test-key'),
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

	it.each(['completed', 'failed'] as const)(
    'refreshes the invalid-bank-information list when an import becomes %s',
    async (terminalStatus) => {
      const queryClient = new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      });
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      vi.spyOn(timesheetService, 'uploadBCCFile').mockResolvedValue(
        createImport('pending'),
	);

      vi.spyOn(timesheetService, 'getPartnerImport').mockResolvedValue(
        createImport(terminalStatus),
      );

      const { result } = renderHook(
        () =>
          useBCCUploadModal({
            projectId: 12,
            onClose: vi.fn(),
          }),
        { wrapper: createWrapper(queryClient) },
      );

      const file = new File(['test'], 'bang-cham-cong.xlsx', {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      act(() => {
        result.current.handleFileChange({
          target: { files: [file] },
        } as unknown as ChangeEvent<HTMLInputElement>);
      });
      act(() => {
        result.current.handleUpload();
      });

      await waitFor(() => {
        expect(result.current.result?.status).toBe(terminalStatus);
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: QueryKeys.employees.missingBankDetails(),
      });
    },
  );

	it.each(['completed', 'failed'] as const)(
    'refreshes the invalid-bank-information list when upload returns %s directly',
    async (terminalStatus) => {
      const queryClient = new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      });
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      vi.spyOn(timesheetService, 'uploadBCCFile').mockResolvedValue(
        createImport(terminalStatus),
      );

      const { result } = renderHook(
        () =>
          useBCCUploadModal({
            projectId: 12,
            onClose: vi.fn(),
          }),
        { wrapper: createWrapper(queryClient) },
      );

      const file = new File(['test'], 'bang-cham-cong.xlsx', {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      act(() => {
        result.current.handleFileChange({
          target: { files: [file] },
        } as unknown as ChangeEvent<HTMLInputElement>);
      });
      act(() => {
        result.current.handleUpload();
      });

      await waitFor(() => {
        expect(result.current.result?.status).toBe(terminalStatus);
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: QueryKeys.employees.missingBankDetails(),
      });
    },
  );

  it.each(['completed', 'failed'] as const)(
    'refreshes the invalid-bank-information list when upload returns %s directly',
    async (terminalStatus) => {
      const queryClient = new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      });
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      vi.spyOn(timesheetService, 'uploadBCCFile').mockResolvedValue(
        createImport(terminalStatus),
      );

      const { result } = renderHook(
        () =>
          useBCCUploadModal({
            projectId: 12,
            onClose: vi.fn(),
          }),
        { wrapper: createWrapper(queryClient) },
      );

      const file = new File(['test'], 'bang-cham-cong.xlsx', {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      act(() => {
        result.current.handleFileChange({
          target: { files: [file] },
        } as unknown as ChangeEvent<HTMLInputElement>);
      });
      act(() => {
        result.current.handleUpload();
      });

      await waitFor(() => {
        expect(result.current.result?.status).toBe(terminalStatus);
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: QueryKeys.employees.missingBankDetails(),
      });
    },
	);

	it('sends the admin flexible-pay import choice', async () => {
		const queryClient = new QueryClient({
			defaultOptions: {
				queries: { retry: false },
				mutations: { retry: false },
			},
		});
		const uploadSpy = vi.spyOn(timesheetService, 'uploadBCCFile').mockResolvedValue(
			createImport('completed'),
		);
		const { result } = renderHook(
			() => useBCCUploadModal({ projectId: 12, allowFlexibleEmployeeImport: true, onClose: vi.fn() }),
			{ wrapper: createWrapper(queryClient) },
		);
		const file = new File(['test'], 'bang-cham-cong.xlsx', {
			type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
		});
		act(() => {
			result.current.handleFileChange({ target: { files: [file] } } as unknown as ChangeEvent<HTMLInputElement>);
		});
		act(() => {
			result.current.handleIncludeFlexibleEmployeesChange(true);
		});
		act(() => {
			result.current.handleUpload();
		});

		await waitFor(() => {
			expect(uploadSpy).toHaveBeenCalledWith(file, 12, expect.any(String), true, 'bcc-upload-test-key');
		});
	});

  it('explains when a payroll file is uploaded to BCC', () => {
    const errors = [{
      row: 0,
      employee: '',
      reason: 'Tệp này là bảng lương, không phải bảng chấm công',
    }];

    expect(buildFailureSummary(errors)).toBe(
      'Tệp này là bảng lương. Hãy dùng “Nhập bảng lương” thay vì tải lên BCC.',
    );
    expect(Array.from(groupErrorsByEmployee(errors).keys())).toEqual(['Tệp đã tải lên']);
  });

  it('explains a legacy BCC format error without an employee label', () => {
    const errors = [{
      row: 0,
      employee: '',
      reason: 'Tệp không đúng mẫu bảng chấm công',
    }];

    expect(buildFailureSummary(errors)).toBe(
      'Tệp chưa đúng mẫu BCC. Hãy chọn tệp có ngày và ca làm việc.',
    );
    expect(Array.from(groupErrorsByEmployee(errors).keys())).toEqual(['Tệp đã tải lên']);
  });
});
