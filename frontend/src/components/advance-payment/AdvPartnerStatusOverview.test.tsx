import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { AdvPartnerStatusOverview } from './AdvPartnerStatusOverview';

const baseProps = {
  totalPaid: 157,
  totalPending: 0,
  totalFailed: 0,
  totalCancelled: 1,
  totalRequests: 158,
  totalPaidAmount: 610920990,
  totalCancelledAmount: 3000000,
  successRate: 100,
};

describe('AdvPartnerStatusOverview', () => {
  it('shows the success rate exactly once — the header chip', () => {
    render(<AdvPartnerStatusOverview {...baseProps} />);

    expect(screen.getAllByText('100%')).toHaveLength(1);
  });

  it('renders the status cells — this zone owns the per-status counts', () => {
    render(<AdvPartnerStatusOverview {...baseProps} />);

    expect(screen.getByText('Hoàn tất')).toBeInTheDocument();
    expect(screen.getByText('Đã hủy')).toBeInTheDocument();
    expect(screen.getByText('158')).toBeInTheDocument();
  });
});
