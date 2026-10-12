import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { OpsApiKeysSection } from './OpsApiKeysSection';
import type { OpsAPIKey } from '@/services/api/opsApiKeys.service';

const mocks = vi.hoisted(() => {
  const initialKeys: OpsAPIKey[] = [
    {
      id: 1,
      name: 'hermes',
      scope: 'read_only',
      effective_scope: 'read_only',
      write_window_expires_at: null,
      key_prefix: 'pk_ops_9Kd2',
      created_by: 10,
      last_used_at: null,
      revoked_at: null,
      created_at: '2026-10-12T13:33:00.000Z',
    },
    {
      id: 2,
      name: 'claude-code-bao-cao',
      scope: 'read_write',
      effective_scope: 'read_only',
      write_window_expires_at: '2026-10-12T15:00:00.000Z',
      key_prefix: 'pk_ops_mNo3',
      created_by: 10,
      last_used_at: '2026-10-12T08:00:00.000Z',
      revoked_at: null,
      created_at: '2026-10-11T10:00:00.000Z',
    },
    {
      id: 3,
      name: 'mimo-cũ',
      scope: 'read_only',
      effective_scope: 'read_only',
      write_window_expires_at: null,
      key_prefix: 'pk_ops_zZ91',
      created_by: 10,
      last_used_at: null,
      revoked_at: '2026-10-10T10:00:00.000Z',
      created_at: '2026-10-09T10:00:00.000Z',
    },
  ];

  return {
    refetch: vi.fn(),
    createKey: vi.fn(),
    revokeKey: vi.fn(),
    setScope: vi.fn(),
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

vi.mock('@/hooks/api/useOpsApiKeys', () => ({
  useOpsApiKeys: () => ({
    data: { data: mocks.keys },
    isLoading: mocks.isLoading,
    isError: mocks.isError,
    isFetching: mocks.isFetching,
    refetch: mocks.refetch,
  }),
  useCreateOpsAPIKey: () => ({
    mutateAsync: mocks.createKey,
    isPending: mocks.createPending,
  }),
  useRevokeOpsAPIKey: () => ({
    mutateAsync: mocks.revokeKey,
  }),
  useSetOpsAPIKeyScope: () => ({
    mutateAsync: mocks.setScope,
  }),
}));

describe('OpsApiKeysSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.keys = [...mocks.initialKeys];
    mocks.isLoading = false;
    mocks.isError = false;
    mocks.createPending = false;

    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });

    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  it('renders the create form, key list, and the install guide', () => {
    render(<OpsApiKeysSection />);

    // Create card + scope options (ops vocabulary: read_only/read_write).
    expect(screen.getByText('Tạo khoá ops cho agent (payroll-ops-mcp)')).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: /Chỉ đọc \(READ ONLY\)/ })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: /Đọc \+ ghi \(READ WRITE\)/ })).toBeInTheDocument();

    // List rows: live read_only key, lapsed-window key, revoked key.
    expect(screen.getByText('hermes')).toBeInTheDocument();
    expect(screen.getByText('claude-code-bao-cao')).toBeInTheDocument();
    expect(screen.getByText('Hết cửa sổ ghi')).toBeInTheDocument();
    expect(screen.getByText('Đã thu hồi')).toBeInTheDocument();

    // Install guide: all five steps present (step titles carry nested spans,
    // so match on single-text-node fragments).
    expect(screen.getByText('Cài đặt MCP vào agent của bạn')).toBeInTheDocument();
    expect(screen.getByText(/ở khung phía trên/)).toBeInTheDocument();
    expect(screen.getByText(/Lưu khoá vào file bí mật \(chế độ 600\)/)).toBeInTheDocument();
    expect(screen.getByText('Cấu hình MCP client')).toBeInTheDocument();
    expect(screen.getByText(/Phạm vi khoá: chỉ đọc theo mặc định/)).toBeInTheDocument();
    expect(screen.getByText(/Kill switch \(tắt toàn bộ kênh khi cần\)/)).toBeInTheDocument();

    // Copyable snippets carry the real env contract.
    expect(screen.getByText(/chmod 600 ~\/\.secrets\/payroll-ops-agent\.key/)).toBeInTheDocument();
    expect(screen.getAllByText(/PAYROLL_OPS_KEY_FILE/).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText(/PAYROLL_OPS_MCP_DISABLED=1/)).toBeInTheDocument();
    expect(screen.getByText(/claude mcp add payroll-ops/)).toBeInTheDocument();
  });

  it('requires a scope selection before creating', async () => {
    render(<OpsApiKeysSection />);

    fireEvent.change(screen.getByLabelText(/Tên khoá \(tên agent\)/), { target: { value: 'agent-mới' } });
    fireEvent.click(screen.getByRole('button', { name: 'Tạo khoá ops' }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('Chọn phạm vi truy cập cho khoá ops');
    });
    expect(mocks.createKey).not.toHaveBeenCalled();
  });

  it('creates a read_only key by explicit selection and reveals the plaintext once', async () => {
    mocks.createKey.mockResolvedValueOnce({
      data: {
        id: 4,
        name: 'agent-mới',
        scope: 'read_only',
        effective_scope: 'read_only',
        write_window_expires_at: null,
        key_prefix: 'pk_ops_new12',
        key: 'pk_ops_new123_super_secret_plaintext',
        created_by: 10,
        last_used_at: null,
        revoked_at: null,
        created_at: new Date().toISOString(),
      },
    });

    render(<OpsApiKeysSection />);

    fireEvent.change(screen.getByLabelText(/Tên khoá \(tên agent\)/), { target: { value: 'agent-mới' } });
    fireEvent.click(screen.getByRole('radio', { name: /Chỉ đọc \(READ ONLY\)/ }));
    fireEvent.click(screen.getByRole('button', { name: 'Tạo khoá ops' }));

    await waitFor(() => {
      expect(mocks.createKey).toHaveBeenCalledWith({ name: 'agent-mới', scope: 'read_only' });
    });

    expect(
      await screen.findByDisplayValue('pk_ops_new123_super_secret_plaintext'),
    ).toBeInTheDocument();
    // The reveal alert (exact title — the create card carries a similar phrase).
    expect(screen.getByText('Khoá chỉ hiển thị một lần — sao chép ngay')).toBeInTheDocument();
  });

  it('sends write_window_hours when creating a read_write key', async () => {
    mocks.createKey.mockResolvedValueOnce({
      data: {
        id: 5,
        name: 'agent-ghi',
        scope: 'read_write',
        effective_scope: 'read_write',
        write_window_expires_at: new Date().toISOString(),
        key_prefix: 'pk_ops_ww123',
        key: 'pk_ops_ww123_super_secret_plaintext',
        created_by: 10,
        last_used_at: null,
        revoked_at: null,
        created_at: new Date().toISOString(),
      },
    });

    render(<OpsApiKeysSection />);

    fireEvent.change(screen.getByLabelText(/Tên khoá \(tên agent\)/), { target: { value: 'agent-ghi' } });
    fireEvent.click(screen.getByRole('radio', { name: /Đọc \+ ghi \(READ WRITE\)/ }));
    fireEvent.change(screen.getByLabelText(/Cửa sổ ghi \(giờ\)/), { target: { value: '4' } });
    fireEvent.click(screen.getByRole('button', { name: 'Tạo khoá ops' }));

    await waitFor(() => {
      expect(mocks.createKey).toHaveBeenCalledWith({
        name: 'agent-ghi',
        scope: 'read_write',
        write_window_hours: 4,
      });
    });
  });

  it('toggles a key to read_write (confirm-gated, 8h default window)', async () => {
    render(<OpsApiKeysSection />);

    // Row 1 (hermes, read_only) — button grants write.
    const row = screen.getByText('hermes').closest('li');
    expect(row).not.toBeNull();
    fireEvent.click(within(row as HTMLElement).getByRole('button', { name: /Cấp ghi \(8h\)/ }));

    await waitFor(() => {
      expect(mocks.setScope).toHaveBeenCalledWith({
        id: 1,
        payload: { scope: 'read_write' },
      });
    });
  });

  it('toggles a lapsed-window key back to read_only', async () => {
    render(<OpsApiKeysSection />);

    const row = screen.getByText('claude-code-bao-cao').closest('li');
    expect(row).not.toBeNull();
    fireEvent.click(within(row as HTMLElement).getByRole('button', { name: /Về chỉ đọc/ }));

    await waitFor(() => {
      expect(mocks.setScope).toHaveBeenCalledWith({
        id: 2,
        payload: { scope: 'read_only' },
      });
    });
  });

  it('revokes a live key after confirmation', async () => {
    render(<OpsApiKeysSection />);

    const row = screen.getByText('hermes').closest('li');
    expect(row).not.toBeNull();
    fireEvent.click(within(row as HTMLElement).getByRole('button', { name: /Thu hồi/ }));

    await waitFor(() => {
      expect(mocks.revokeKey).toHaveBeenCalledWith(1);
    });
  });

  it('copies the secrets-file snippet on demand', async () => {
    render(<OpsApiKeysSection />);

    const secretsStep = screen.getByText(/chmod 600/).closest('li')?.parentElement?.parentElement
      ? screen.getByText('Cài đặt MCP vào agent của bạn').closest('section')
      : null;
    expect(secretsStep).not.toBeNull();
    const buttons = within(secretsStep as HTMLElement).getAllByRole('button', { name: 'Sao chép' });
    fireEvent.click(buttons[0]); // first snippet = secrets-file commands

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalled();
    });
  });
});
