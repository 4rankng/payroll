import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { ConsoleDonut, ConsoleGauge, toDonutSegments } from './console-charts';

describe('toDonutSegments', () => {
  it('keeps a short tail as-is', () => {
    const prepared = toDonutSegments([
      { label: 'A', value: 3, display: '3' },
      { label: 'B', value: 1, display: '1' },
    ]);
    expect(prepared.map((segment) => segment.label)).toEqual(['A', 'B']);
    expect(prepared.map((segment) => segment.percentage)).toEqual([75, 25]);
  });

  it('collapses beyond five named segments into Khác with summed values', () => {
    const prepared = toDonutSegments([
      { label: 'A', value: 5, display: '5' },
      { label: 'B', value: 4, display: '4' },
      { label: 'C', value: 3, display: '3' },
      { label: 'D', value: 2, display: '2' },
      { label: 'E', value: 1, display: '1' },
      { label: 'F', value: 2, display: '2' },
      { label: 'G', value: 3, display: '3' },
    ]);
    expect(prepared).toHaveLength(6);
    expect(prepared.at(-1)?.label).toBe('Khác');
    expect(prepared.at(-1)?.value).toBe(5);
    expect(prepared.at(-1)?.percentage).toBe(25);
  });

  it('drops zero-value segments and yields zero percentages for an empty set', () => {
    expect(toDonutSegments([{ label: 'A', value: 0, display: '0' }])).toHaveLength(0);
    expect(toDonutSegments([])).toHaveLength(0);
  });

  it('falls back to the chart token ladder by index', () => {
    const prepared = toDonutSegments([
      { label: 'A', value: 1, display: '1' },
      { label: 'B', value: 1, display: '1', color: '#123456' },
    ]);
    expect(prepared[0].color).toBe('hsl(var(--chart-1))');
    expect(prepared[1].color).toBe('#123456');
  });
});

describe('ConsoleDonut', () => {
  it('renders the centre figure and one legend row per segment', () => {
    const { container, getByText, getByRole } = render(
      <ConsoleDonut
        segments={[
          { label: 'MB', value: 3, display: '120 gd · 3 tỷ' },
          { label: 'VCB', value: 1, display: '40 gd · 1 tỷ' },
        ]}
        centreValue="4 tỷ"
        centreLabel="Tổng chi trả"
      />,
    );
    expect(getByRole('img')).toHaveAttribute('aria-label', 'Tổng chi trả 4 tỷ');
    expect(getByText('MB')).toBeInTheDocument();
    expect(getByText('VCB')).toBeInTheDocument();
    expect(getByText('120 gd · 3 tỷ')).toBeInTheDocument();
    expect(getByText('75%')).toBeInTheDocument();
    expect(getByText('25%')).toBeInTheDocument();
    // Two segment arcs + one track ring.
    expect(container.querySelectorAll('circle')).toHaveLength(3);
  });

  it('renders only the track ring when the total is zero', () => {
    const { container } = render(
      <ConsoleDonut segments={[]} centreValue="0" centreLabel="Tổng chi trả" />,
    );
    expect(container.querySelectorAll('circle')).toHaveLength(1);
  });
});

describe('ConsoleGauge', () => {
  it('renders null for a non-finite rate (denominator-zero last resort)', () => {
    const { container } = render(<ConsoleGauge value={Number.NaN} label="Hoàn tất" />);
    expect(container.firstChild).toBeNull();
  });

  it('clamps rates above 1 and defaults the figure to a whole percent', () => {
    const { getByText, getByRole } = render(<ConsoleGauge value={1.4} label="Hoàn tất" />);
    expect(getByText('100%')).toBeInTheDocument();
    expect(getByRole('img')).toHaveAttribute('aria-label', 'Hoàn tất 100%');
  });

  it('honours an explicit display figure', () => {
    const { getByText } = render(<ConsoleGauge value={0.5} display="4/8" label="Điểm danh" />);
    expect(getByText('4/8')).toBeInTheDocument();
  });
});
