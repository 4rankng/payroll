import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { AdvancePaymentPageHeaderMobile } from './AdvancePaymentPageHeaderMobile';

describe('AdvancePaymentPageHeaderMobile', () => {
  it('keeps primary and overflow actions at least 44px on mobile', () => {
    render(
      <AdvancePaymentPageHeaderMobile
        onImportPayroll={vi.fn()}
        onViewEmployees={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Nhập' })).toHaveClass('h-11', 'min-h-11');
    expect(screen.getByRole('button', { name: 'Tùy chọn' })).toHaveClass(
      'h-11',
      'min-h-11',
      'w-11',
      'min-w-11',
    );
  });
});
