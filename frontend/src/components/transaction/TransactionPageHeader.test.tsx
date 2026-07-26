import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { TransactionPageHeader } from './TransactionPageHeader';

describe('TransactionPageHeader', () => {
  it('keeps settlement simulation directly accessible at normal desktop widths', () => {
    const onSimulateSettlement = vi.fn();

    render(
      <TransactionPageHeader
        onAddTransaction={vi.fn()}
        onSendSaoKePayroll={vi.fn()}
        onSendSaoKeAdvance={vi.fn()}
        onViewSaoKeHistory={vi.fn()}
        onExportSaoKePayroll={vi.fn()}
        onExportSaoKeAdvance={vi.fn()}
        onImportOnePayFeeReport={vi.fn()}
        onSimulateSettlement={onSimulateSettlement}
      />,
    );

    const simulationButton = screen.getByTestId('compact-settlement-simulation-button');

    expect(simulationButton).toHaveClass('hidden', 'lg:inline-flex');
    fireEvent.click(simulationButton);
    expect(onSimulateSettlement).toHaveBeenCalledOnce();
  });
});
