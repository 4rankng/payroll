import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ZaloConnectionSection } from './ZaloConnectionSection';

const mocks = vi.hoisted(() => ({
  refetch: vi.fn(),
  saveCredentials: vi.fn(),
  setEnabled: vi.fn(),
  testSend: vi.fn(),
  refreshToken: vi.fn(),
  status: {
    enabled: true,
    configured: true,
    connected: true,
    app_id: 'app-123',
    template_id: '617976',
    expires_at: '2099-01-01T00:00:00.000Z',
  },
}));

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('@/hooks/api/useZaloConnection', () => ({
  useZaloStatus: () => ({
    data: { data: mocks.status },
    isLoading: false,
    isError: false,
    refetch: mocks.refetch,
  }),
  useSaveZaloCredentials: () => ({
    mutateAsync: mocks.saveCredentials,
    isPending: false,
  }),
  useSetZaloEnabled: () => ({
    mutateAsync: mocks.setEnabled,
    isPending: false,
  }),
  useTestZaloSend: () => ({
    mutateAsync: mocks.testSend,
    isPending: false,
  }),
  useRefreshZaloToken: () => ({
    mutateAsync: mocks.refreshToken,
    isPending: false,
  }),
}));

describe('ZaloConnectionSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.saveCredentials.mockResolvedValue({ data: undefined });
    mocks.setEnabled.mockResolvedValue({ data: undefined });
    mocks.refreshToken.mockResolvedValue({ data: undefined });
    mocks.testSend.mockResolvedValue({
      data: { error_code: 0, error_msg: 'Thành công', msg_id: 'msg-123' },
    });
    Object.assign(mocks.status, {
      enabled: true,
      configured: true,
      connected: true,
      app_id: 'app-123',
      template_id: '617976',
    });
  });

  it('saves the complete credential payload', async () => {
    render(<ZaloConnectionSection />);

    fireEvent.change(screen.getByLabelText('App ID'), { target: { value: 'app-123' } });
    fireEvent.change(screen.getByLabelText('Khóa bí mật (Secret Key)'), { target: { value: 'secret-123' } });
    fireEvent.change(screen.getByLabelText('Mã truy cập (Access Token)'), { target: { value: 'access-123' } });
    fireEvent.change(screen.getByLabelText('Mã làm mới (Refresh Token)'), { target: { value: 'refresh-123' } });
    fireEvent.change(screen.getByLabelText('Mã mẫu ZNS'), { target: { value: 'template-123' } });
    fireEvent.click(screen.getByRole('button', { name: 'Lưu cấu hình' }));

    await waitFor(() => {
      expect(mocks.saveCredentials).toHaveBeenCalledWith({
        app_id: 'app-123',
        secret_key: 'secret-123',
        access_token: 'access-123',
        refresh_token: 'refresh-123',
        template_id: 'template-123',
      });
    });
  });

  it('checks the stored credentials without sending a message', async () => {
    render(<ZaloConnectionSection />);

    fireEvent.click(screen.getByRole('button', { name: 'Kiểm tra kết nối' }));

    await waitFor(() => expect(mocks.refreshToken).toHaveBeenCalledOnce());
    expect(mocks.testSend).not.toHaveBeenCalled();
  });

  it('requires confirmation before disabling the employee OTP flow', async () => {
    render(<ZaloConnectionSection />);

    fireEvent.click(screen.getByRole('switch', { name: 'Bật đặt lại mật khẩu qua Zalo' }));
    expect(mocks.setEnabled).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Xác nhận tắt' }));

    await waitFor(() => expect(mocks.setEnabled).toHaveBeenCalledWith(false));
  });

  it('trims the test phone before sending', async () => {
    render(<ZaloConnectionSection />);

    fireEvent.change(screen.getByLabelText('Số điện thoại nhận thử'), {
      target: { value: ' 0357210887 ' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Gửi thử' }));

    await waitFor(() => expect(mocks.testSend).toHaveBeenCalledWith({ phone: '0357210887' }));
    expect(await screen.findByRole('status')).toHaveTextContent('Gửi thành công');
    expect(screen.getByText(/0xxxxxxxxx hoặc 84xxxxxxxxx/)).toBeInTheDocument();
  });

  it('exposes the setup state without clipping labels', () => {
    render(<ZaloConnectionSection />);

    const activationStep = screen.getByText('Kích hoạt');
    expect(activationStep).not.toHaveClass('truncate');
    expect(activationStep).toHaveTextContent('đã hoàn tất');
  });
});
