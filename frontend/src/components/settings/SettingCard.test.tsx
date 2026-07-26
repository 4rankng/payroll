import { useState } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { SettingCard } from './SettingCard';

const renderCurrencyCard = ({
  initialValue = '400000000',
  originalValue = '400000000',
  onSave = vi.fn(),
}: {
  initialValue?: string;
  originalValue?: string;
  onSave?: () => void;
} = {}) => {
  const CurrencyCard = () => {
    const [value, setValue] = useState(initialValue);
    return (
      <SettingCard
        title="Giới hạn tổng tiền mỗi file Chuyển lô"
        description="Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này."
        value={value}
        originalValue={originalValue}
        onChange={setValue}
        onSave={onSave}
        onReset={() => setValue(originalValue)}
        isDirty={value !== originalValue}
        isSaving={false}
        displayMode="currency-vnd"
        min="2"
        max="9223372036854775807"
      />
    );
  };

  render(<CurrencyCard />);
  return screen.getByLabelText('Giới hạn tổng tiền mỗi file Chuyển lô');
};

describe('SettingCard currency-vnd mode', () => {
  it('formats the full VND value without converting it to Number', () => {
    const input = renderCurrencyCard();

    expect(input).toHaveValue('400.000.000');
    expect(screen.getByText('đ')).toBeInTheDocument();
    expect(input).toHaveAttribute('inputmode', 'numeric');
    expect(input).toHaveAttribute(
      'aria-describedby',
      expect.stringContaining('-description'),
    );
  });

  it('preserves int64 precision when formatting the maximum value', () => {
    const input = renderCurrencyCard({
      initialValue: '9223372036854775807',
      originalValue: '9223372036854775807',
    });

    expect(input).toHaveValue('9.223.372.036.854.775.807');
  });

  it.each([
    ['', 'Giá trị không được để trống'],
    ['1', 'Giá trị phải từ'],
    ['9223372036854775808', 'Giá trị phải từ'],
  ])('blocks invalid canonical value %s', (value, expectedError) => {
    renderCurrencyCard({ initialValue: value });

    expect(screen.getByRole('alert')).toHaveTextContent(expectedError);
    expect(screen.getByRole('button', { name: 'Lưu' })).toBeDisabled();
  });

  it('rejects decimal and negative edits instead of silently changing their meaning', () => {
    const input = renderCurrencyCard();

    fireEvent.change(input, { target: { value: '2,5' } });
    expect(screen.getByRole('alert')).toHaveTextContent('Chỉ nhập số nguyên dương');
    expect(input).toHaveValue('400.000.000');

    fireEvent.change(input, { target: { value: '2.5' } });
    expect(screen.getByRole('alert')).toHaveTextContent('Chỉ nhập số nguyên dương');
    expect(input).toHaveValue('400.000.000');

    fireEvent.change(input, { target: { value: '-5' } });
    expect(screen.getByRole('alert')).toHaveTextContent('Chỉ nhập số nguyên dương');
    expect(screen.queryByRole('button', { name: 'Lưu' })).not.toBeInTheDocument();
  });

  it('normalizes a valid edit and exposes 44px save controls', () => {
    const onSave = vi.fn();
    const input = renderCurrencyCard({ onSave });

    fireEvent.change(input, { target: { value: '500000000' } });

    expect(input).toHaveValue('500.000.000');
    expect(screen.getByRole('button', { name: 'Hủy' })).toHaveClass('h-11');
    const saveButton = screen.getByRole('button', { name: 'Lưu' });
    expect(saveButton).toHaveClass('h-11');
    fireEvent.click(saveButton);
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it('keeps retry available when the setting cannot be loaded', () => {
    const onRetry = vi.fn();
    render(
      <SettingCard
        title="Giới hạn tổng tiền mỗi file Chuyển lô"
        value=""
        originalValue=""
        onChange={vi.fn()}
        onSave={vi.fn()}
        onReset={vi.fn()}
        isDirty={false}
        isSaving={false}
        displayMode="currency-vnd"
        min="2"
        max="9223372036854775807"
        unavailableMessage="Không thể tải cài đặt. Vui lòng thử lại."
        onRetry={onRetry}
      />,
    );

    expect(screen.getByLabelText('Giới hạn tổng tiền mỗi file Chuyển lô')).toBeDisabled();
    const retryButton = screen.getByRole('button', { name: 'Thử lại' });
    expect(retryButton).toHaveClass('h-11');
    fireEvent.click(retryButton);
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
