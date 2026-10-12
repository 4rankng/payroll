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

    fireEvent.click(screen.getByRole('button', { name: 'Thu hồi' }));

    expect(window.confirm).toHaveBeenCalledWith(
      'Thu hồi khoá MCP "Agent báo cáo" (READ ONLY)? Agent đang dùng khoá này sẽ mất quyền truy cập ngay.',
    );
    await waitFor(() => {
      expect(mocks.revokeKey).toHaveBeenCalledWith(1);
    });
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
