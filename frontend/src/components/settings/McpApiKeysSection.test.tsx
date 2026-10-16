import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { McpApiKeysSection } from './McpApiKeysSection';
import type { MCPAPIKey } from '@/services/api/mcpApiKeys.service';

const mocks = vi.hoisted(() => {
  const initialKeys: MCPAPIKey[] = [
    {
      id: 1,
      name: 'Agent báo cáo',
      scope: 'read_only',
      key_prefix: 'mcp_40WjkqjL',
      created_by: 10,
      last_used_at: null,
      revoked_at: null,
      scope_updated_at: null,
      created_at: '2026-09-26T13:33:00.000Z',
    },
    {
      id: 2,
      name: 'Agent đối soát',
      scope: 'full',
      key_prefix: 'mcp_abc123xy',
      created_by: 10,
      last_used_at: '2026-09-27T08:00:00.000Z',
      revoked_at: '2026-09-25T10:00:00.000Z',
      scope_updated_at: '2026-09-26T09:00:00.000Z',
      created_at: '2026-09-25T10:00:00.000Z',
    },
    {
      id: 3,
      name: 'Agent ghi liệu',
      scope: 'full',
      key_prefix: 'mcp_wri789te',
      created_by: 10,
      last_used_at: null,
      revoked_at: null,
      scope_updated_at: '2026-09-28T06:30:00.000Z',
      created_at: '2026-09-24T10:00:00.000Z',
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

vi.mock('@/hooks/api/useMcpApiKeys', () => ({
  useMcpApiKeys: () => ({
    data: { data: mocks.keys },
    isLoading: mocks.isLoading,
    isError: mocks.isError,
    isFetching: mocks.isFetching,
    refetch: mocks.refetch,
  }),
  useCreateMCPAPIKey: () => ({
    mutateAsync: mocks.createKey,
    isPending: mocks.createPending,
  }),
  useRevokeMCPAPIKey: () => ({
    mutateAsync: mocks.revokeKey,
  }),
  useSetMCPAPIKeyScope: () => ({
    mutateAsync: mocks.setScope,
  }),
}));

describe('McpApiKeysSection', () => {
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

  it('renders create form, key list, and the integration guide', () => {
    render(<McpApiKeysSection />);

    // Create card + both scope options.
    expect(screen.getByText('Tạo khoá API MCP')).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: /Chỉ đọc \(READ ONLY\)/ })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: /Toàn quyền \(ALL\)/ })).toBeInTheDocument();

    // List: both fixture keys with scope badges.
    expect(screen.getByText('Agent báo cáo')).toBeInTheDocument();
    expect(screen.getByText('Agent đối soát')).toBeInTheDocument();
    // READ ONLY appears on the badge and in the scope guide table.
    expect(screen.getAllByText('READ ONLY').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('ALL').length).toBeGreaterThanOrEqual(1);

    // Guide: endpoint, tools, snippets, recipes, security notes.
    expect(screen.getByText('Hướng dẫn tích hợp MCP')).toBeInTheDocument();
    // The endpoint shows in its snippet, the Claude command and mcp.json.
    expect(screen.getAllByText(/\/api\/v1\/mcp/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/payroll_api/)).toBeInTheDocument();
    expect(screen.getByText(/payroll_endpoints/)).toBeInTheDocument();
    expect(screen.getByText(/claude mcp add --transport http payroll/)).toBeInTheDocument();
    expect(screen.getByText(/"mcpServers"/)).toBeInTheDocument();
    expect(screen.getAllByText(/GET \/api\/v1\/employees/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/ưu tiên dùng READ ONLY/i)).toBeInTheDocument();
  });

  it('requires a scope selection before creating', async () => {
    render(<McpApiKeysSection />);

    fireEvent.change(screen.getByLabelText(/Tên khoá/), { target: { value: 'Agent mới' } });
    fireEvent.click(screen.getByRole('button', { name: 'Tạo khoá MCP' }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('Chọn phạm vi truy cập cho khoá MCP');
    });
    expect(mocks.createKey).not.toHaveBeenCalled();
  });

  it('creates a key with the selected scope and reveals the plaintext once', async () => {
    mocks.createKey.mockResolvedValueOnce({
      data: {
        id: 3,
        name: 'Agent mới',
        scope: 'read_only',
        key_prefix: 'mcp_new12345',
        key: 'mcp_new12345_super_secret_plaintext_key',
        created_by: 10,
        last_used_at: null,
        revoked_at: null,
        scope_updated_at: null,
        created_at: new Date().toISOString(),
      },
    });

    render(<McpApiKeysSection />);

    fireEvent.change(screen.getByLabelText(/Tên khoá/), { target: { value: 'Agent mới' } });
    fireEvent.click(screen.getByRole('radio', { name: /Chỉ đọc \(READ ONLY\)/ }));
    fireEvent.click(screen.getByRole('button', { name: 'Tạo khoá MCP' }));

    await waitFor(() => {
      expect(mocks.createKey).toHaveBeenCalledWith({ name: 'Agent mới', scope: 'read_only' });
    });

    expect(
      screen.getByText('Khoá chỉ hiển thị một lần — sao chép ngay'),
    ).toBeInTheDocument();
    const keyInput = screen.getByLabelText('Khoá MCP cho Agent mới');
    expect(keyInput).toHaveValue('mcp_new12345_super_secret_plaintext_key');

    fireEvent.click(within(screen.getByRole('alert')).getByRole('button', { name: 'Sao chép' }));
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'mcp_new12345_super_secret_plaintext_key',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Đã lưu khoá (Đóng)' }));
    expect(
      screen.queryByText('Khoá chỉ hiển thị một lần — sao chép ngay'),
    ).not.toBeInTheDocument();
  });

  it('requires confirmation before revoking', async () => {
    mocks.revokeKey.mockResolvedValueOnce({ data: undefined });

    render(<McpApiKeysSection />);

    const row = screen.getByText('Agent báo cáo').closest('li');
    fireEvent.click(within(row as HTMLElement).getByRole('button', { name: 'Thu hồi' }));

    expect(window.confirm).toHaveBeenCalledWith(
      'Thu hồi khoá MCP "Agent báo cáo" (READ ONLY)? Agent đang dùng khoá này sẽ mất quyền truy cập ngay.',
    );
    await waitFor(() => {
      expect(mocks.revokeKey).toHaveBeenCalledWith(1);
    });
  });

  it('renders the scope toggle per key scope and hides it on revoked keys', () => {
    render(<McpApiKeysSection />);

    const readOnlyRow = screen.getByText('Agent báo cáo').closest('li');
    expect(readOnlyRow).not.toBeNull();
    expect(within(readOnlyRow as HTMLElement).getByRole('button', { name: /Cấp ghi \(ALL\)/ })).toBeInTheDocument();
    // READ ONLY keys never offer "Về chỉ đọc" — they are already there.
    expect(within(readOnlyRow as HTMLElement).queryByRole('button', { name: /Về chỉ đọc/ })).not.toBeInTheDocument();

    const fullRow = screen.getByText('Agent ghi liệu').closest('li');
    expect(fullRow).not.toBeNull();
    expect(within(fullRow as HTMLElement).getByRole('button', { name: /Về chỉ đọc/ })).toBeInTheDocument();
    expect(within(fullRow as HTMLElement).queryByRole('button', { name: /Cấp ghi/ })).not.toBeInTheDocument();

    // Revoked keys show neither the toggle nor the revoke button.
    const revokedRow = screen.getByText('Agent đối soát').closest('li');
    expect(revokedRow).not.toBeNull();
    expect(within(revokedRow as HTMLElement).queryByRole('button', { name: /Cấp ghi/ })).not.toBeInTheDocument();
    expect(within(revokedRow as HTMLElement).queryByRole('button', { name: /Thu hồi/ })).not.toBeInTheDocument();
  });

  it('shows the last scope change time when the API returns it', () => {
    render(<McpApiKeysSection />);

    // Keys 2 and 3 carry scope_updated_at; key 1 (never toggled) does not.
    // The rendered time uses the local timezone, so match the shape, not a
    // fixed clock reading.
    const fullRow = screen.getByText('Agent ghi liệu').closest('li');
    expect(
      within(fullRow as HTMLElement).getAllByText(/Đổi phạm vi lần cuối \d{2}\/\d{2}\/\d{4} \d{2}:\d{2}/)
        .length,
    ).toBe(1);
    const neverToggledRow = screen.getByText('Agent báo cáo').closest('li');
    expect(
      within(neverToggledRow as HTMLElement).queryByText(/Đổi phạm vi lần cuối/),
    ).not.toBeInTheDocument();
  });

  it('toggles a read_only key to full after confirmation', async () => {
    mocks.setScope.mockResolvedValueOnce({ data: mocks.keys[0] });

    render(<McpApiKeysSection />);

    fireEvent.click(screen.getByRole('button', { name: /Cấp ghi \(ALL\)/ }));

    expect(window.confirm).toHaveBeenCalledWith(
      'Chuyển khoá MCP "Agent báo cáo" sang ALL? Khoá sẽ được phép ghi dữ liệu với quyền quản trị của người tạo khoá. Yêu cầu phiên đã xác thực OTP.',
    );
    await waitFor(() => {
      expect(mocks.setScope).toHaveBeenCalledWith({
        id: 1,
        payload: { scope: 'full' },
      });
    });
  });

  it('toggles a full key back to read_only after confirmation', async () => {
    mocks.setScope.mockResolvedValueOnce({ data: mocks.keys[2] });

    render(<McpApiKeysSection />);

    const row = screen.getByText('Agent ghi liệu').closest('li');
    fireEvent.click(within(row as HTMLElement).getByRole('button', { name: /Về chỉ đọc/ }));

    expect(window.confirm).toHaveBeenCalledWith(
      'Chuyển khoá MCP "Agent ghi liệu" về chỉ đọc (READ ONLY)? Agent sẽ bị chặn mọi thao tác ghi ở lượt gọi tiếp theo.',
    );
    await waitFor(() => {
      expect(mocks.setScope).toHaveBeenCalledWith({
        id: 3,
        payload: { scope: 'read_only' },
      });
    });
  });

  it('does not toggle when the admin cancels the confirmation', () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<McpApiKeysSection />);

    fireEvent.click(screen.getByRole('button', { name: /Cấp ghi \(ALL\)/ }));

    expect(mocks.setScope).not.toHaveBeenCalled();
  });

  it('copies guide snippets to clipboard', async () => {
    render(<McpApiKeysSection />);

    const endpointCopy = screen.getAllByRole('button', { name: 'Sao chép' })[0];
    fireEvent.click(endpointCopy);
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expect.stringContaining('/api/v1/mcp'));
    });
  });

  it('renders empty state when there are no keys', () => {
    mocks.keys = [];
    render(<McpApiKeysSection />);
    expect(screen.getByText('Chưa có khoá MCP nào')).toBeInTheDocument();
  });
});
