import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { AdvPartnerMetricsStrip } from './AdvPartnerMetricsStrip';

const baseProps = {
  totalPaid: 157,
  totalRequests: 158,
  totalCancelled: 1,
  completedUnder30s: 153,
  completed30sTo2m: 0,
  completed2mTo5m: 0,
  completed5mTo15m: 0,
  completedOver15m: 0,
  successRate: 100,
};

describe('AdvPartnerMetricsStrip', () => {
  it('shows the success rate exactly once, inside the ring gauge', () => {
    render(<AdvPartnerMetricsStrip {...baseProps} />);

    expect(
      screen.getByRole('img', { name: 'Tỷ lệ thành công 100.0%' }),
    ).toBeInTheDocument();
    // The label next to the ring must stay text-only (no second number).
    expect(screen.getByText('Tỷ lệ thành công')).toBeInTheDocument();
  });

  it('never re-displays per-status counts owned by AdvPartnerStatusOverview', () => {
    render(<AdvPartnerMetricsStrip {...baseProps} />);

    // Old caption duplicated "157/157 HT · 1 hủy" against the status cells.
    expect(screen.queryByText(/HT ·/)).not.toBeInTheDocument();
    expect(screen.queryByText(/hủy/)).not.toBeInTheDocument();
  });

  it('renders the processing-time buckets', () => {
    render(<AdvPartnerMetricsStrip {...baseProps} />);

    expect(screen.getByText('<30 giây')).toBeInTheDocument();
    expect(screen.getByText('153')).toBeInTheDocument();
  });
});
