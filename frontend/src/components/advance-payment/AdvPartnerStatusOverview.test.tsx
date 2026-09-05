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
  it('shows the success-rate chip by default (standalone views rely on it)', () => {
    render(<AdvPartnerStatusOverview {...baseProps} />);

    expect(screen.getByText('100%')).toBeInTheDocument();
  });

  it('hides the success-rate chip when the adjacent ring gauge owns the number', () => {
    render(<AdvPartnerStatusOverview {...baseProps} showSuccessBadge={false} />);

    expect(screen.queryByText('100%')).not.toBeInTheDocument();
    // Status cells stay — they are this zone's own data.
    expect(screen.getByText('Hoàn tất')).toBeInTheDocument();
    expect(screen.getByText('Đã hủy')).toBeInTheDocument();
    expect(screen.getByText('158')).toBeInTheDocument();
  });
});
