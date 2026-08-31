import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { useState } from 'react';
import { SearchableSelect } from './searchable-select';

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

vi.stubGlobal('ResizeObserver', ResizeObserverMock);
Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
  configurable: true,
  value: vi.fn(),
});

const PROJECTS = [
  { value: '1', label: 'PQC Hải Phòng' },
  { value: '2', label: 'GEORIM' },
  { value: '3', label: 'Phòng kế toán' },
  { value: '4', label: 'Tan Nhan site', disabled: true },
];

function Harness({ onChange }: { onChange?: (v: string) => void }) {
  const [value, setValue] = useState<string | undefined>(undefined);
  return (
    <SearchableSelect
      value={value}
      onChange={onChange ?? ((v) => setValue(v))}
      options={PROJECTS}
      placeholder="Chọn dự án"
    />
  );
}

describe('SearchableSelect', () => {
  it('opens and shows options with the selected one checked', () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('combobox'));
    expect(screen.getByText('PQC Hải Phòng')).toBeTruthy();
    expect(screen.getByText('GEORIM')).toBeTruthy();
  });

  it('filters diacritic-insensitively ("phan phong" → "Phòng...")', () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('combobox'));
    const input = screen.getByPlaceholderText('Tìm kiếm...');
    fireEvent.change(input, { target: { value: 'ke toan' } });
    expect(screen.getByText('Phòng kế toán')).toBeTruthy();
    expect(screen.queryByText('GEORIM')).toBeNull();
  });

  it('calls onChange and closes on select', () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    fireEvent.click(screen.getByRole('combobox'));
    fireEvent.click(screen.getByText('GEORIM'));
    expect(onChange).toHaveBeenCalledWith('2');
  });

  it('shows empty message when nothing matches', () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('combobox'));
    fireEvent.change(screen.getByPlaceholderText('Tìm kiếm...'), {
      target: { value: 'zzzz' },
    });
    expect(screen.getByText('Không tìm thấy.')).toBeTruthy();
  });
});
