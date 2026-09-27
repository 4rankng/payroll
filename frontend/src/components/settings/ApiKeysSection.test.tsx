import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { ApiKeysSection } from './ApiKeysSection';
import type { APIKey } from '@/services/api/apiKeys.service';

const mocks = vi.hoisted(() => {
  const initialKeys: APIKey[] = [
    {
      id: 1,
      name: 'TingHire',
      key_prefix: 'ttk_40WjkqjL',
      created_by: 10,
      last_used_at: null,
      revoked_at: '2026-09-26T13:33:00.000Z',
      created_at: '2026-09-26T13:33:00.000Z',
    },
    {
      id: 2,
      name: 'Zalo Chatbot Active',
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
    expect(screen.getByText('TingHire')).toBeInTheDocument();
    expect(screen.getByText('Zalo Chatbot Active')).toBeInTheDocument();
    expect(screen.getByText('ttk_40WjkqjL••••••••••••')).toBeInTheDocument();
    expect(screen.getByText('ttk_abc123xy••••••••••••')).toBeInTheDocument();

    // Active key has Revoke button, revoked key does not
    expect(screen.getByRole('button', { name: 'Thu hồi' })).toBeInTheDocument();
  });

  it('allows creating a new API key and reveals the secret key', async () => {
    mocks.createKey.mockResolvedValueOnce({
      data: {
        id: 3,
        name: 'New Chatbot',
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
    fireEvent.change(input, { target: { value: 'New Chatbot' } });

    const createButton = screen.getByRole('button', { name: 'Tạo khoá API' });
    fireEvent.click(createButton);

    await waitFor(() => {
      expect(mocks.createKey).toHaveBeenCalledWith({ name: 'New Chatbot' });
    });

    // Revealed key alert
    expect(
      screen.getByText('Khoá chỉ hiển thị một lần. Hãy sao chép ngay.'),
    ).toBeInTheDocument();
    const keyInput = screen.getByLabelText('Khoá API cho New Chatbot');
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
      'Thu hồi khoá "Zalo Chatbot Active"? Chatbot đang dùng khoá này sẽ ngừng hoạt động ngay.',
    );
    await waitFor(() => {
      expect(mocks.revokeKey).toHaveBeenCalledWith(2);
    });
  });

  it('copies prefix and target skill path to clipboard', async () => {
    render(<ApiKeysSection />);

    // Copy prefix of first key
    const copyPrefixBtn = screen.getByRole('button', {
      name: 'Sao chép tiền tố ttk_40WjkqjL',
    });
    fireEvent.click(copyPrefixBtn);
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith('ttk_40WjkqjL');
    });

    // Copy target skill path
    const copyPathBtn = screen.getByRole('button', {
      name: 'Sao chép đường dẫn',
    });
    fireEvent.click(copyPathBtn);
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        '.claude/skills/payroll-password-reset/SKILL.md',
      );
    });
  });

  it('toggles SKILL.md preview and downloads file', async () => {
    const createObjectURLMock = vi.fn().mockReturnValue('blob:mock-url');
    const revokeObjectURLMock = vi.fn();
    window.URL.createObjectURL = createObjectURLMock;
    window.URL.revokeObjectURL = revokeObjectURLMock;

    render(<ApiKeysSection />);

    // Toggle preview
    const previewBtn = screen.getByRole('button', { name: /Xem trước nội dung SKILL.md/ });
    fireEvent.click(previewBtn);

    expect(screen.getByText('Nội dung tệp: SKILL.md')).toBeInTheDocument();
    expect(screen.getByText(/Thu gọn xem trước/)).toBeInTheDocument();

    // Download button
    const downloadBtn = screen.getByRole('button', { name: 'Tải hướng dẫn API (SKILL.md)' });
    fireEvent.click(downloadBtn);

    expect(createObjectURLMock).toHaveBeenCalled();
    expect(revokeObjectURLMock).toHaveBeenCalledWith('blob:mock-url');
    expect(toast.success).toHaveBeenCalledWith('Đã tải hướng dẫn API cho agent');
  });

  it('renders empty state when there are no keys', () => {
    mocks.keys = [];
    render(<ApiKeysSection />);
    expect(screen.getByText('Chưa có khoá API nào')).toBeInTheDocument();
  });
});
