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
    expect(screen.getByText('₫')).toBeInTheDocument();
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

  it('accepts 200000000 typed through the formatted display without error', () => {
    const input = renderCurrencyCard();

    // Typing appends one keystroke to the previously rendered formatted
    // value. Once the formatter inserts separators ("2.000"), the next
    // keystroke arrives as "2.0000" — every one of these states must parse.
    const typedDomValues = [
      '2',
      '20',
      '200',
      '2000',
      '2.0000',
      '20.0000',
      '200.0000',
      '2.000.0000',
      '20.000.0000',
    ];
    for (const domValue of typedDomValues) {
      fireEvent.change(input, { target: { value: domValue } });
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    }

    expect(input).toHaveValue('200.000.000');
    expect(screen.getByRole('button', { name: 'Lưu' })).toBeEnabled();
  });

  it('accepts backspacing through the formatted display', () => {
    const input = renderCurrencyCard({ initialValue: '2000' });

    // Backspace shortens the formatted display "2.000" to "2.00" — the dots
    // are separators, not decimal points, so the edit must keep "200".
    fireEvent.change(input, { target: { value: '2.00' } });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(input).toHaveValue('200');

    fireEvent.change(input, { target: { value: '20' } });
    expect(input).toHaveValue('20');
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

describe('SettingCard currency-slider mode', () => {
  const renderSliderCard = ({
    initialValue = '400000000',
    originalValue = '400000000',
    onSave = vi.fn(),
  }: {
    initialValue?: string;
    originalValue?: string;
    onSave?: () => void;
  } = {}) => {
    const SliderCard = () => {
      const [value, setValue] = useState(initialValue);
      return (
        <SettingCard
          title="Giới hạn tổng tiền mỗi file Chuyển lô"
          value={value}
          originalValue={originalValue}
          onChange={setValue}
          onSave={onSave}
          onReset={() => setValue(originalValue)}
          isDirty={value !== originalValue}
          isSaving={false}
          displayMode="currency-slider"
          min="100000000"
          max="500000000"
        />
      );
    };

    render(<SliderCard />);
    return screen.getByRole('slider', { name: 'Giới hạn tổng tiền mỗi file Chuyển lô' });
  };

  it('renders the formatted value and the slider bounds', () => {
    const slider = renderSliderCard();

    expect(slider).toHaveAttribute('aria-valuenow', '400000000');
    expect(slider).toHaveAttribute('aria-valuemin', '100000000');
    expect(slider).toHaveAttribute('aria-valuemax', '500000000');
    expect(screen.getByText('400.000.000')).toBeInTheDocument();
    expect(screen.getByText('100.000.000')).toBeInTheDocument();
    expect(screen.getByText('500.000.000')).toBeInTheDocument();
  });

  it('moves in 10M steps and saves', () => {
    const onSave = vi.fn();
    const slider = renderSliderCard({ onSave });

    fireEvent.keyDown(slider, { key: 'ArrowRight' });

    expect(slider).toHaveAttribute('aria-valuenow', '410000000');
    expect(screen.getByText('410.000.000')).toBeInTheDocument();

    const saveButton = screen.getByRole('button', { name: 'Lưu' });
    expect(saveButton).toBeEnabled();
    fireEvent.click(saveButton);
    expect(onSave).toHaveBeenCalledTimes(1);
  });
});
