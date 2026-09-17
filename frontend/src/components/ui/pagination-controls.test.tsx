import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PaginationControls } from './pagination-controls';

describe('pagination controls', () => {
  it('navigates without submitting a containing form and labels the page size', () => {
    const onPageChange = vi.fn();
    const onSubmit = vi.fn((event) => event.preventDefault());
    render(
      <form onSubmit={onSubmit}>
        <PaginationControls
          pagination={{ page: 4, pageSize: 20, totalPages: 10, totalRecords: 200 }}
          onPageChange={onPageChange}
          onPageSizeChange={vi.fn()}
        />
      </form>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Trang tiếp' }));
    expect(onPageChange).toHaveBeenLastCalledWith(5);
    fireEvent.click(screen.getByRole('button', { name: 'Trang 3' }));
    expect(onPageChange).toHaveBeenLastCalledWith(3);
    fireEvent.click(screen.getByRole('button', { name: 'Trang cuối' }));
    expect(onPageChange).toHaveBeenLastCalledWith(10);
    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByRole('combobox', { name: 'Số dòng mỗi trang' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Phân trang' })).toBeInTheDocument();
  });

  it('does not navigate beyond an empty result', () => {
    const onPageChange = vi.fn();
    render(<PaginationControls pagination={{ page: 1, pageSize: 20, totalPages: 0, totalRecords: 0 }} onPageChange={onPageChange} onPageSizeChange={vi.fn()} />);
    for (const label of ['Trang đầu', 'Trang trước', 'Trang tiếp', 'Trang cuối']) {
      const button = screen.getByRole('button', { name: label });
      expect(button).toBeDisabled();
      fireEvent.click(button);
    }
    expect(onPageChange).not.toHaveBeenCalled();
    expect(screen.getByText('0–0')).toBeInTheDocument();
  });
});
