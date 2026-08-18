import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { EmployeeSingleSelector } from './EmployeeSingleSelector';

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

describe('EmployeeSingleSelector', () => {
  it('uses the paginated server search, loads more, and returns one selected employee', () => {
    vi.useFakeTimers();
    const fetchNextPage = vi.fn();
    const onSelect = vi.fn();
    useEmployeesInfinite.mockImplementation((filters: { search?: string }) => ({
      data: {
        pages: [{
          data: filters.search === 'Nguyễn'
            ? [{ id: 21, fullname: 'Nguyễn Thị An', cccd: '001002003004' }]
            : [{ id: 11, fullname: 'Cà Văn Hiến', cccd: '011090007288' }],
        }],
      },
      fetchNextPage,
      hasNextPage: true,
      isFetchingNextPage: false,
      isLoading: false,
      isError: false,
    }));

    render(<EmployeeSingleSelector value={null} onSelect={onSelect} />);

    fireEvent.click(screen.getByRole('combobox', { name: 'Chọn nhân viên' }));
    const commandList = document.querySelector('[cmdk-list]');
    Object.defineProperties(commandList!, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 200 },
      scrollTop: { configurable: true, value: 100 },
    });
    fireEvent.scroll(commandList!);
    expect(fetchNextPage).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByPlaceholderText('Tìm theo tên hoặc CCCD...'), {
      target: { value: 'Nguyễn' },
    });
    act(() => vi.advanceTimersByTime(300));

    expect(useEmployeesInfinite).toHaveBeenLastCalledWith({ pageSize: 20, search: 'Nguyễn' });
    fireEvent.click(screen.getByText('Nguyễn Thị An'));
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 21 }));
  });
});
