import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useState } from 'react';

import { EmployeeMultiSelector } from './employee-multi-selector';

const useEmployeesInfinite = vi.fn();

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

vi.mock('@/hooks/api/useEmployees', () => ({
  useEmployeesInfinite: (filters: unknown) => useEmployeesInfinite(filters),
}));

afterEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
});

describe('EmployeeMultiSelector', () => {
  it('searches through the paginated API, loads another page, and preserves selected IDs across a new query', () => {
    vi.useFakeTimers();
    const fetchNextPage = vi.fn();
    useEmployeesInfinite.mockImplementation((filters: { search?: string }) => ({
      data: {
        pages: [{
          data: filters.search === 'Nguyễn'
            ? [{ id: 22001, fullname: 'Nguyễn Văn Khác', cccd: '001002003004' }]
            : [{ id: 11156, fullname: 'Cà Văn Hiến', cccd: '011090007288' }],
        }],
      },
      fetchNextPage,
      hasNextPage: true,
      isFetchingNextPage: false,
      isLoading: false,
      isError: false,
    }));

    function ControlledSelector() {
      const [value, setValue] = useState<number[]>([]);
      return <EmployeeMultiSelector value={value} onChange={setValue} />;
    }

    render(<ControlledSelector />);

    fireEvent.click(screen.getByRole('combobox', { name: 'Chọn nhân viên' }));
    const commandList = document.querySelector('[cmdk-list]');
    expect(commandList).not.toBeNull();
    Object.defineProperties(commandList!, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 200 },
      scrollTop: { configurable: true, value: 100 },
    });
    fireEvent.scroll(commandList!);
    expect(fetchNextPage).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByText('Cà Văn Hiến'));
    expect(screen.getByText('Đã chọn 1 nhân viên')).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('Tìm kiếm tên hoặc CCCD...'), {
      target: { value: 'Nguyễn' },
    });

    act(() => {
      vi.advanceTimersByTime(300);
    });

    expect(useEmployeesInfinite).toHaveBeenLastCalledWith({
      pageSize: 20,
      search: 'Nguyễn',
    });
    expect(screen.getByText('Nguyễn Văn Khác')).toBeInTheDocument();
    expect(screen.getByText('Đã chọn 1 nhân viên')).toBeInTheDocument();
  });
});
