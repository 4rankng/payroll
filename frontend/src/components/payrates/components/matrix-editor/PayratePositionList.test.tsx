import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PayratePositionList } from './PayratePositionList';

const rates = {
  'Lương 500': {
    'ngày thường': { HC: 62500, TCN: 93750 },
    'ngày nghỉ': { HC: 125000, TCN: 125000 },
    'ngày lễ': { HC: 187500, TCN: 187500 },
  },
};

describe('PayratePositionList', () => {
  it('renders a position-first editor without squeezing data into a table', () => {
    const { container } = render(
      <PayratePositionList
        rates={rates}
        positions={['Lương 500']}
        hourTypes={['HC', 'TCN']}
        editingHourType={null}
        editingHourTypeValue=""
        onEditHourType={vi.fn()}
        onSaveHourType={vi.fn()}
        onRemoveHourType={vi.fn()}
        onCancelEditHourType={vi.fn()}
        setEditingHourTypeValue={vi.fn()}
        editingPosition={null}
        editingPositionValue=""
        onEditPosition={vi.fn()}
        onSavePosition={vi.fn()}
        onRemovePosition={vi.fn()}
        onCopyRates={vi.fn()}
        setEditingPositionValue={vi.fn()}
        setEditingPosition={vi.fn()}
        onRateChange={vi.fn()}
      />,
    );

    expect(screen.getByRole('region', { name: 'Mức lương cho Lương 500' })).toBeInTheDocument();
    expect(screen.getByText('Đơn vị: ₫/giờ')).toBeInTheDocument();
    expect(screen.getByText('Ngày thường')).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: 'Mức lương Lương 500, ngày thường, HC' })).toHaveValue('62.500');
    expect(screen.getByRole('button', { name: 'Tùy chọn cho vị trí Lương 500' })).toBeInTheDocument();
    expect(container.querySelector('table')).not.toBeInTheDocument();
  });
});
