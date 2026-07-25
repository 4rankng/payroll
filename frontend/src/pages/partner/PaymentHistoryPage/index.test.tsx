import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import PaymentHistoryPage from './index';

vi.mock('@/components/payroll/BankTransferHistoryPageContent', () => ({
  BankTransferHistoryPageContent: () => (
    <div data-testid="bank-transfer-history">Bút toán ngân hàng</div>
  ),
}));

describe('PartnerPaymentHistoryPage', () => {
  it('uses the shared bank-transfer history contract', () => {
    render(<PaymentHistoryPage />);

    expect(screen.getByTestId('bank-transfer-history')).toBeInTheDocument();
  });
});
