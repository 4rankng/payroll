/**
 * Masonry packing for console galleries.
 *
 * A plain 12-column grid sizes every row to its tallest cell, so a short
 * panel beside a tall one leaves a hole of page background underneath it.
 * Payroll dashboards pair deliberately uneven panels (a wide table next to a
 * donut, a tall leaderboard next to a bar chart), so those holes are the rule.
 *
 * This grid keeps the same 12 tracks but packs panels masonry-style: every
 * panel spans as many 1px rows as its content needs (plus the 12px gap) and
 * dense auto-placement drops the next panel into the first gap its span fits.
 * Below `lg` the panels stack in one column and the spans are cleared.
 *
 * Ported from the latest toto.sg insights gallery. Guards keep jsdom tests
 * (no ResizeObserver/matchMedia) on the plain grid fallback.
 */
import { type ReactNode, useLayoutEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

/** Matches the gallery's `gap-3`: 0.75rem at a 16px root. */
const GAP_PX = 12;

/** The packing only applies once the gallery has its twelve columns. */
const COLUMNS_QUERY = '(min-width: 64rem)';

interface MasonryGridProps {
  className?: string;
  children: ReactNode;
}

export function MasonryGrid({ className, children }: MasonryGridProps) {
  const ref = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const grid = ref.current;
    if (!grid) return;
    if (typeof window === 'undefined' || typeof ResizeObserver === 'undefined') return;

    const query = window.matchMedia?.(COLUMNS_QUERY);
    if (!query) return;

    let frame = 0;

    const panels = () =>
      Array.from(grid.children).filter((child): child is HTMLElement => child instanceof HTMLElement);

    const pack = () => {
      frame = 0;
      const items = panels();

      if (!query.matches) {
        grid.style.gridAutoRows = '';
        grid.style.gridAutoFlow = '';
        grid.style.rowGap = '';
        for (const item of items) item.style.gridRowEnd = '';
        return;
      }

      // Measure before reshaping: every panel still reports the height its
      // content needs at the width its column span gives it. Widths do not
      // change when the spans are applied, so those heights stay valid.
      const heights = items.map((item) => item.getBoundingClientRect().height);

      grid.style.gridAutoRows = '1px';
      grid.style.gridAutoFlow = 'row dense';
      grid.style.rowGap = '0px';

      // The span includes the gap, so the next panel in the same column starts
      // exactly 12px below this one.
      items.forEach((item, index) => {
        item.style.gridRowEnd = `span ${Math.ceil(heights[index]) + GAP_PX}`;
      });
    };

    const schedule = () => {
      if (frame !== 0) return;
      frame = window.requestAnimationFrame(pack);
    };

    pack();

    // Watch the container (a width change reflows every chart) and every
    // panel (a chart can re-render when data or hover state changes).
    const observer = new ResizeObserver(schedule);
    observer.observe(grid);
    for (const item of panels()) observer.observe(item);

    // Web fonts land after first paint and can nudge the measured heights.
    if (document.fonts) {
      document.fonts.ready.then(schedule).catch(() => {});
    }
    query.addEventListener('change', schedule);

    return () => {
      observer.disconnect();
      query.removeEventListener('change', schedule);
      if (frame !== 0) cancelAnimationFrame(frame);
    };
  });

  return (
    <div
      ref={ref}
      data-slot="masonry-grid"
      className={cn('grid items-start gap-3 lg:grid-cols-12', className)}
    >
      {children}
    </div>
  );
}
