import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { SettingsEmployeePanel } from './SettingsEmployeePanel';
import type { Setting } from '@/types/api/settings.types';

const mocks = vi.hoisted(() => ({
  refetch: vi.fn(),
  updateSetting: vi.fn(),
  createSetting: vi.fn(),
  settings: [] as (Setting | undefined)[],
  isLoading: false,
  isError: false,
  updatePending: false,
  createPending: false,
}));

vi.mock('@/hooks/api/useSettings', () => ({
  useMultipleSettings: () => ({
    data: mocks.settings,
    isLoading: mocks.isLoading,
    isError: mocks.isError,
    refetch: mocks.refetch,
  }),
  useUpdateSetting: () => ({
    mutate: mocks.updateSetting,
    isPending: mocks.updatePending,
  }),
  useCreateSetting: () => ({
    mutate: mocks.createSetting,
    isPending: mocks.createPending,
  }),
}));

const baseSetting = {
  value_type: 'boolean' as const,
  updated_at: '2026-10-12T00:00:00Z',
};

const enabledSetting: Setting = {
  id: 21,
  key: 'employee.mobile_required',
  value: 'true',
  ...baseSetting,
};

const disabledSetting: Setting = {
  id: 21,
  key: 'employee.mobile_required',
  value: 'false',
  ...baseSetting,
};

const toggleRow = () =>
  screen.getByRole('switch', { name: 'Bắt buộc số điện thoại khi thêm nhân viên mới' });

describe('SettingsEmployeePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.settings = [];
    mocks.isLoading = false;
    mocks.isError = false;
    mocks.updatePending = false;
    mocks.createPending = false;
  });

  it('renders off when the settings row is missing', () => {
    mocks.settings = [undefined];
    render(<SettingsEmployeePanel />);

    expect(toggleRow()).not.toBeChecked();
    expect(screen.queryByRole('button', { name: 'Lưu' })).not.toBeInTheDocument();
  });

  it('renders on when the stored value is true', () => {
    mocks.settings = [enabledSetting];
    render(<SettingsEmployeePanel />);

    expect(toggleRow()).toBeChecked();
    expect(screen.queryByRole('button', { name: 'Lưu' })).not.toBeInTheDocument();
  });

  it('exposes the save/reset pair only while dirty and resets on Hoàn tác', () => {
    mocks.settings = [disabledSetting];
    render(<SettingsEmployeePanel />);

    fireEvent.click(toggleRow());
    expect(screen.getByRole('button', { name: 'Lưu' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Hoàn tác' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Hoàn tác' }));
    expect(toggleRow()).not.toBeChecked();
    expect(screen.queryByRole('button', { name: 'Lưu' })).not.toBeInTheDocument();
  });

  it('creates the boolean row on first save when it does not exist yet', async () => {
    mocks.settings = [undefined];
    mocks.createSetting.mockImplementation((_data, options) => {
      options?.onSuccess?.();
    });
    render(<SettingsEmployeePanel />);

    fireEvent.click(toggleRow());
    fireEvent.click(screen.getByRole('button', { name: 'Lưu' }));

    expect(mocks.createSetting).toHaveBeenCalledWith(
      { key: 'employee.mobile_required', value: 'true', value_type: 'boolean' },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    expect(mocks.updateSetting).not.toHaveBeenCalled();
    expect(toggleRow()).toBeChecked();
    expect(screen.queryByRole('button', { name: 'Lưu' })).not.toBeInTheDocument();
  });

  it('updates the existing row and persists a false value', () => {
    mocks.settings = [enabledSetting];
    mocks.updateSetting.mockImplementation((_vars, options) => {
      options?.onSuccess?.();
    });
    render(<SettingsEmployeePanel />);

    fireEvent.click(toggleRow());
    fireEvent.click(screen.getByRole('button', { name: 'Lưu' }));

    expect(mocks.updateSetting).toHaveBeenCalledWith(
      { id: 21, data: { value: 'false', value_type: 'boolean' } },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    expect(mocks.createSetting).not.toHaveBeenCalled();
    expect(toggleRow()).not.toBeChecked();
    expect(screen.queryByRole('button', { name: 'Lưu' })).not.toBeInTheDocument();
  });

  it('disables the switch and actions while saving', () => {
    mocks.settings = [disabledSetting];
    const { rerender } = render(<SettingsEmployeePanel />);

    fireEvent.click(toggleRow());
    expect(screen.getByRole('button', { name: 'Lưu' })).toBeEnabled();

    mocks.updatePending = true;
    rerender(<SettingsEmployeePanel />);

    expect(toggleRow()).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Lưu' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Hoàn tác' })).toBeDisabled();
  });

  it('shows a retry state when the setting cannot load', () => {
    mocks.isError = true;
    render(<SettingsEmployeePanel />);

    expect(screen.getByRole('alert')).toHaveTextContent('Không thể tải cài đặt nhân viên');
    fireEvent.click(screen.getByRole('button', { name: 'Thử lại' }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
    expect(screen.queryByRole('switch')).not.toBeInTheDocument();
  });
});
