import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { MimoConnectionSection } from './MimoConnectionSection';
import type { Setting } from '@/types/api/settings.types';

const mocks = vi.hoisted(() => ({
  refetch: vi.fn(),
  updateSetting: vi.fn(),
  createSetting: vi.fn(),
  testConnection: vi.fn(),
  settings: [] as (Setting | undefined)[],
  isLoading: false,
  isError: false,
  updatePending: false,
  createPending: false,
  testPending: false,
}));

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('@/hooks/api/useSettings', () => ({
  useSettings: () => ({
    data: mocks.settings,
    isLoading: mocks.isLoading,
    isError: mocks.isError,
    refetch: mocks.refetch,
  }),
  useUpdateSetting: () => ({
    mutateAsync: mocks.updateSetting,
    isPending: mocks.updatePending,
  }),
  useCreateSetting: () => ({
    mutateAsync: mocks.createSetting,
    isPending: mocks.createPending,
  }),
  useTestMimoConnection: () => ({
    mutateAsync: mocks.testConnection,
    isPending: mocks.testPending,
  }),
}));

const baseSetting = {
  value_type: 'string' as const,
  updated_at: '2026-10-11T00:00:00Z',
};

const maskedKeySetting: Setting = {
  id: 11,
  key: 'mimo.api_key',
  value: null,
  masked: true,
  configured: true,
  ...baseSetting,
};

const defaultModelSetting: Setting = {
  id: 12,
  key: 'mimo.default_model',
  value: 'mimo-v2.6-flash',
  ...baseSetting,
};

const fallbackModelSetting: Setting = {
  id: 13,
  key: 'mimo.fallback_model',
  value: 'mimo-v2.6-pro',
  ...baseSetting,
};

describe('MimoConnectionSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.settings = [];
    mocks.isLoading = false;
    mocks.isError = false;
    mocks.updatePending = false;
    mocks.createPending = false;
    mocks.testPending = false;
  });

  it('renders the designed missing-key state when no key is configured', () => {
    render(<MimoConnectionSection />);

    expect(screen.getByText('Trợ lý AI MiMo')).toBeInTheDocument();
    expect(screen.getByText('Chưa cấu hình')).toBeInTheDocument();

    // Designed missing-key state: visible before any test runs, and the test
    // button stays disabled until a key is saved.
    expect(screen.getByRole('alert')).toHaveTextContent('Chưa có Khóa bí mật');
    expect(
      screen.getByText('Nhập Khóa bí mật và bấm "Lưu cấu hình" để bật trợ lý AI.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Kiểm tra kết nối/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /Lưu cấu hình/ })).toBeEnabled();
  });

  it('saves a new key via create when the row does not exist yet', async () => {
    mocks.createSetting.mockResolvedValue({ data: undefined });
    mocks.updateSetting.mockResolvedValue({ data: undefined });
    render(<MimoConnectionSection />);

    fireEvent.change(screen.getByLabelText('Khóa bí mật'), {
      target: { value: 'mimo-sk-brand-new' },
    });

    fireEvent.click(screen.getByRole('button', { name: /Lưu cấu hình/ }));

    await waitFor(() => {
      expect(mocks.createSetting).toHaveBeenCalledWith({
        key: 'mimo.api_key',
        value: 'mimo-sk-brand-new',
        value_type: 'string',
      });
    });
    // Model routing rows are upserted alongside the key.
    await waitFor(() => {
      expect(mocks.createSetting).toHaveBeenCalledWith({
        key: 'mimo.default_model',
        value: 'mimo-v2.6-flash',
        value_type: 'string',
      });
    });
    expect(toast.success).toHaveBeenCalledWith('Đã lưu cấu hình MiMo');
  });

  it('updates the existing masked row when a key is replaced', async () => {
    mocks.settings = [maskedKeySetting, defaultModelSetting, fallbackModelSetting];
    mocks.updateSetting.mockResolvedValue({ data: undefined });
    render(<MimoConnectionSection />);

    // Masked secrets never render their value; the field is empty by design.
    expect(screen.getByText('Đã cấu hình')).toBeInTheDocument();
    expect(screen.getByLabelText('Khóa bí mật')).toHaveValue('');

    fireEvent.change(screen.getByLabelText('Khóa bí mật'), {
      target: { value: 'mimo-sk-replacement' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Lưu cấu hình/ }));

    await waitFor(() => {
      expect(mocks.updateSetting).toHaveBeenCalledWith({
        id: maskedKeySetting.id,
        data: { value: 'mimo-sk-replacement' },
      });
    });
    expect(mocks.createSetting).not.toHaveBeenCalled();
  });

  it('disables the test button while the key has unsaved changes', () => {
    mocks.settings = [maskedKeySetting, defaultModelSetting, fallbackModelSetting];
    render(<MimoConnectionSection />);

    expect(screen.getByRole('button', { name: /Kiểm tra kết nối/ })).toBeEnabled();
    fireEvent.change(screen.getByLabelText('Khóa bí mật'), {
      target: { value: 'not-saved-yet' },
    });
    expect(screen.getByRole('button', { name: /Kiểm tra kết nối/ })).toBeDisabled();
    expect(screen.getByText('Lưu thay đổi trước khi kiểm tra.')).toBeInTheDocument();
  });

  it('shows the success state with latency after a good connection test', async () => {
    mocks.settings = [maskedKeySetting, defaultModelSetting, fallbackModelSetting];
    mocks.testConnection.mockResolvedValue({
      success: true,
      code: 'ok',
      message: 'Kết nối MiMo thành công — độ trễ 234 ms, 2 mô hình khả dụng',
      latency_ms: 234,
      model_count: 2,
      default_model_present: true,
    });
    render(<MimoConnectionSection />);

    fireEvent.click(screen.getByRole('button', { name: /Kiểm tra kết nối/ }));

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent('Kết nối MiMo thành công — độ trễ 234 ms');
    });
    expect(toast.success).toHaveBeenCalledWith('Kết nối MiMo hợp lệ');
  });

  it.each([
    ['invalid_key', 'Khóa API không hợp lệ', 'Khóa đã bị thu hồi hoặc sai — dán khóa mới từ MiMo Platform rồi lưu lại.'],
    ['timeout', 'MiMo không phản hồi kịp', 'Kết nối đã hết thời gian chờ — thử lại sau ít phút.'],
    ['rate_limited', 'Đã vượt giới hạn tần suất', 'Thử lại sau ít phút.'],
    ['unavailable', 'Dịch vụ MiMo đang gián đoạn', 'Thử lại sau ít phút. Các nghiệp vụ khác không bị ảnh hưởng.'],
    ['busy', 'Hệ thống đang xử lý nhiều yêu cầu AI', 'Thử lại sau ít phút.'],
  ] as const)('designs the %s error state with both buttons staying enabled', async (code, title, hint) => {
    mocks.settings = [maskedKeySetting, defaultModelSetting, fallbackModelSetting];
    mocks.testConnection.mockResolvedValue({
      success: false,
      code,
      message: `Mô phỏng lỗi ${code}`,
    });
    render(<MimoConnectionSection />);

    fireEvent.click(screen.getByRole('button', { name: /Kiểm tra kết nối/ }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(title);
    });
    expect(screen.getByRole('alert')).toHaveTextContent(`Mô phỏng lỗi ${code}`);
    expect(screen.getByText(hint)).toBeInTheDocument();
    // Designed state: retry stays possible — both buttons remain enabled.
    expect(screen.getByRole('button', { name: /Kiểm tra kết nối/ })).toBeEnabled();
    expect(screen.getByRole('button', { name: /Lưu cấu hình/ })).toBeEnabled();
  });

  it('falls back to the generic error state when the request throws', async () => {
    mocks.settings = [maskedKeySetting, defaultModelSetting, fallbackModelSetting];
    mocks.testConnection.mockRejectedValue(new Error('network down'));
    render(<MimoConnectionSection />);

    fireEvent.click(screen.getByRole('button', { name: /Kiểm tra kết nối/ }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Không thể kiểm tra kết nối');
    });
    expect(
      screen.getByText('Không thể kiểm tra kết nối — kiểm tra kết nối mạng và thử lại.'),
    ).toBeInTheDocument();
  });

  it('renders the load-error state with a retry action', () => {
    mocks.isError = true;
    render(<MimoConnectionSection />);

    expect(screen.getByText('Không thể tải cấu hình MiMo.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Tải lại/ }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });

  it('lets admins pick both models from the allowlist', async () => {
    mocks.settings = [maskedKeySetting, defaultModelSetting, fallbackModelSetting];
    mocks.updateSetting.mockResolvedValue({ data: undefined });
    render(<MimoConnectionSection />);

    const defaultSelect = screen.getByLabelText('Mô hình mặc định');
    expect(defaultSelect).toHaveValue('mimo-v2.6-flash');
    const fallbackSelect = screen.getByLabelText('Mô hình dự phòng');
    expect(fallbackSelect).toHaveValue('mimo-v2.6-pro');

    fireEvent.change(defaultSelect, { target: { value: 'mimo-v2.6-pro' } });
    fireEvent.click(screen.getByRole('button', { name: /Lưu cấu hình/ }));

    await waitFor(() => {
      expect(mocks.updateSetting).toHaveBeenCalledWith({
        id: defaultModelSetting.id,
        data: { value: 'mimo-v2.6-pro' },
      });
    });
  });
});
