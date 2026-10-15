import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { ApiKeysSection } from './ApiKeysSection';
import type { APIKey, APIKeyScope } from '@/services/api/apiKeys.service';

const mocks = vi.hoisted(() => {
  const initialKeys: APIKey[] = [
    {
      id: 1,
      name: 'TingHire',
      scope: 'tinghire',
      key_prefix: 'ttk_40WjkqjL',
      created_by: 10,
      last_used_at: null,
      revoked_at: '2026-09-26T13:33:00.000Z',
      created_at: '2026-09-26T13:33:00.000Z',
    },
    {
      id: 2,
      name: 'Vendor Active',
      scope: 'timesheet_read',
      key_prefix: 'ttk_abc123xy',
      created_by: 10,
      last_used_at: '2026-09-27T08:00:00.000Z',
      revoked_at: null,
      created_at: '2026-09-25T10:00:00.000Z',
    },
  ];

  return {
    refetch: vi.fn(),
    createKey: vi.fn(),
    revokeKey: vi.fn(),
    initialKeys,
    keys: [...initialKeys],
    isLoading: false,
    isError: false,
    isFetching: false,
    createPending: false,
  };
});

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('@/hooks/api/useApiKeys', () => ({
  useApiKeys: () => ({
    data: { data: mocks.keys },
    isLoading: mocks.isLoading,
    isError: mocks.isError,
    isFetching: mocks.isFetching,
    refetch: mocks.refetch,
  }),
  useCreateAPIKey: () => ({
    mutateAsync: mocks.createKey,
    isPending: mocks.createPending,
  }),
  useRevokeAPIKey: () => ({
    mutateAsync: mocks.revokeKey,
  }),
}));

describe('ApiKeysSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.keys = [...mocks.initialKeys];
    mocks.isLoading = false;
    mocks.isError = false;
    mocks.createPending = false;

    // Mock clipboard
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });

    // Mock confirm
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  it('renders header, metrics ribbon, and existing keys correctly', () => {
    render(<ApiKeysSection />);

    // Header & Badge
    expect(screen.getByText('Khoá API & Tích hợp Machine-to-Machine')).toBeInTheDocument();
    expect(screen.getByText('Header: X-API-Key')).toBeInTheDocument();

    // Metric Rail
    expect(screen.getByText('Tổng số khoá')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument(); // total keys
    expect(screen.getAllByText('Đang hoạt động').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Đã thu hồi').length).toBeGreaterThanOrEqual(1);

    // Keys rendered
    expect(screen.getAllByText('TingHire').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Vendor Active')).toBeInTheDocument();
    expect(screen.getByText('ttk_40WjkqjL••••••••••••')).toBeInTheDocument();
    expect(screen.getByText('ttk_abc123xy••••••••••••')).toBeInTheDocument();

    // TingHire carries the TingHire badge, Vendor Active carries the Vendor badge.
    expect(screen.getAllByText('TingHire').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('Vendor — bảng chấm công').length).toBe(2);

    // Active key has Revoke button, revoked key does not
    expect(screen.getByRole('button', { name: 'Thu hồi' })).toBeInTheDocument();
  });

  it('allows creating a new API key and reveals the secret key', async () => {
    mocks.createKey.mockResolvedValueOnce({
      data: {
        id: 3,
        name: 'Đối tác A',
        scope: 'timesheet_read',
        key_prefix: 'ttk_new12345',
        key: 'ttk_new12345_super_secret_plaintext_key',
        created_by: 10,
        last_used_at: null,
        revoked_at: null,
        created_at: new Date().toISOString(),
      },
    });

    render(<ApiKeysSection />);

    const input = screen.getByLabelText(/Tên khoá/);
    fireEvent.change(input, { target: { value: 'Đối tác A' } });

    // Select Vendor - bảng chấm công scope radio
    const vendorRadio = screen.getByLabelText(/Vendor — bảng chấm công/);
    fireEvent.click(vendorRadio);

    const createButton = screen.getByRole('button', { name: 'Tạo khoá API' });
    fireEvent.click(createButton);

    await waitFor(() => {
      expect(mocks.createKey).toHaveBeenCalledWith({ name: 'Đối tác A', scope: 'timesheet_read' });
    });

    // Revealed key alert
    expect(
      screen.getByText('Khoá chỉ hiển thị một lần. Hãy sao chép ngay.'),
    ).toBeInTheDocument();
    const keyInput = screen.getByLabelText('Khoá API cho Đối tác A');
    expect(keyInput).toHaveValue('ttk_new12345_super_secret_plaintext_key');

    // Copy revealed key
    const copyButton = screen.getByRole('button', { name: 'Sao chép' });
    fireEvent.click(copyButton);

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'ttk_new12345_super_secret_plaintext_key',
      );
      expect(toast.success).toHaveBeenCalledWith('Đã sao chép khoá API vào clipboard');
    });

    // Dismiss banner
    const dismissButton = screen.getByRole('button', { name: 'Đã lưu khoá (Đóng)' });
    fireEvent.click(dismissButton);
    expect(
      screen.queryByText('Khoá chỉ hiển thị một lần. Hãy sao chép ngay.'),
    ).not.toBeInTheDocument();
  });

  it('requires confirmation before revoking an API key', async () => {
    mocks.revokeKey.mockResolvedValueOnce({ data: undefined });

    render(<ApiKeysSection />);

    const revokeButton = screen.getByRole('button', { name: 'Thu hồi' });
    fireEvent.click(revokeButton);

    expect(window.confirm).toHaveBeenCalledWith(
      'Thu hồi khoá "Vendor Active" (Vendor — bảng chấm công)? Đối tác đang dùng khoá này sẽ mất quyền xem bảng chấm công ngay.',
    );
    await waitFor(() => {
      expect(mocks.revokeKey).toHaveBeenCalledWith(2);
    });
  });

  it('copies the key prefix to clipboard', async () => {
    render(<ApiKeysSection />);

    const copyPrefixBtn = screen.getByRole('button', {
      name: 'Sao chép tiền tố ttk_40WjkqjL',
    });
    fireEvent.click(copyPrefixBtn);
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith('ttk_40WjkqjL');
    });
  });

  it('renders empty state when there are no keys', () => {
    mocks.keys = [];
    render(<ApiKeysSection />);
    expect(screen.getByText('Chưa có khoá API nào')).toBeInTheDocument();
  });

  describe('API key scope', () => {
    it('labels every key row with its scope badge', () => {
      render(<ApiKeysSection />);

      expect(screen.getAllByText('TingHire').length).toBeGreaterThanOrEqual(2);
      expect(screen.getAllByText('Vendor — bảng chấm công').length).toBe(2);
    });

    it('renders chatbot badge for legacy chatbot scope', () => {
      mocks.keys = [
        {
          id: 7,
          name: 'Legacy Chatbot',
          scope: 'chatbot',
          key_prefix: 'ttk_legacy02',
          created_by: 10,
          last_used_at: null,
          revoked_at: null,
          created_at: '2026-09-25T10:00:00.000Z',
        } as unknown as APIKey,
      ];

      render(<ApiKeysSection />);

      expect(screen.getByText('Chatbot (Cũ)')).toBeInTheDocument();
    });

    it('renders an unsupported badge for an unknown scope', () => {
      mocks.keys = [
        {
          id: 8,
          name: 'Unknown Key',
          scope: 'unknown_scope' as unknown as APIKeyScope,
          key_prefix: 'ttk_unknown01',
          created_by: 10,
          last_used_at: null,
          revoked_at: null,
          created_at: '2026-09-25T10:00:00.000Z',
        } as APIKey,
      ];

      render(<ApiKeysSection />);

      expect(screen.getByText('Không còn hỗ trợ')).toBeInTheDocument();
    });

    it('treats a legacy key with no scope as the vendor read scope', () => {
      // scopeOf() falls back to 'timesheet_read' when the server predates the
      // scope column, matching the backend's fail-closed comparison.
      mocks.keys = [
        {
          id: 6,
          name: 'Legacy Key',
          key_prefix: 'ttk_legacy01',
          created_by: 10,
          last_used_at: null,
          revoked_at: null,
          created_at: '2026-09-25T10:00:00.000Z',
        } as APIKey,
      ];

      render(<ApiKeysSection />);

      expect(screen.getAllByText('Vendor — bảng chấm công').length).toBe(2);
    });
  });
});
