import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { AdvPartnerMetricsStrip } from './AdvPartnerMetricsStrip';

const baseProps = {
  completedUnder30s: 153,
  completed30sTo2m: 0,
  completed2mTo5m: 0,
  completed5mTo15m: 0,
  completedOver15m: 0,
};

describe('AdvPartnerMetricsStrip', () => {
  it('renders only the processing-time buckets', () => {
    render(<AdvPartnerMetricsStrip {...baseProps} />);

    expect(screen.getByText('Thời gian xử lý')).toBeInTheDocument();
    expect(screen.getByText('<30 giây')).toBeInTheDocument();
    expect(screen.getByText('153')).toBeInTheDocument();
  });

  it('never shows the success rate or per-status counts (owned by the paired zone)', () => {
    render(<AdvPartnerMetricsStrip {...baseProps} />);

    expect(screen.queryByText('Hiệu suất')).not.toBeInTheDocument();
    expect(screen.queryByText('Tỷ lệ thành công')).not.toBeInTheDocument();
    expect(screen.queryByText('Tỉ lệ thành công')).not.toBeInTheDocument();
    expect(screen.queryByText(/HT ·/)).not.toBeInTheDocument();
    expect(screen.queryByText(/hủy/)).not.toBeInTheDocument();
    expect(screen.queryByText(/%/)).not.toBeInTheDocument();
  });
});
