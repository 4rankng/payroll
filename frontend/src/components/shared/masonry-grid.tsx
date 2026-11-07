/**
 * Masonry gallery, ported from toto.sg `src/components/product/masonry-grid.tsx`.
 *
 * A plain 12-column grid sizes every row to its tallest cell, so a short panel
 * beside a tall one leaves page background underneath it. This grid spans each
 * panel over as many 1px rows as it needs and packs the next panel into the
 * first gap its column span fits. A fill pass grows the panel above each hole
 * until it meets the next one, so no bare page background is left between
 * cards. Below `lg` the panels stack and the measured spans are cleared.
 *
 * Staff console only — the employee portal and auth never use this.
 */
import { type ReactNode, useLayoutEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

/** Matches the gallery's `gap-3`: 0.75rem at a 16px root. */
const GAP_PX = 12;

/** The packing only applies once the gallery has its twelve columns. */
const COLUMNS_QUERY = '(min-width: 64rem)';

/** Ignore rounding noise when deciding whether a column really is short. */
const MIN_STRETCH_PX = 2.5;

interface MasonryGridProps {
  className?: string;
  children: ReactNode;
}

export function MasonryGrid({ className, children }: MasonryGridProps) {
  const ref = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const grid = ref.current;
    if (!grid) return;
    // jsdom and older embedded browsers: no layout engine or no observers.
    if (
      typeof window === 'undefined' ||
      typeof window.matchMedia !== 'function' ||
      typeof ResizeObserver === 'undefined'
    ) {
      return;
    }

    const query = window.matchMedia(COLUMNS_QUERY);
    let disposed = false;
    let frame = 0;
    let packedState = '';

    const panels = () =>
      Array.from(grid.children).filter(
        (child): child is HTMLElement => child instanceof HTMLElement,
      );

    /** The container width plus every panel's rendered box. */
    const layoutState = () => {
      const parts = [String(Math.round(grid.getBoundingClientRect().width))];
      for (const item of panels()) {
        const r = item.getBoundingClientRect();
        parts.push(`${Math.round(r.width)}x${Math.round(r.height)}`);
      }
      return parts.join('|');
    };

    /** Drop any height the fill pass added, so the natural height is measured. */
    const unstretch = (items: HTMLElement[]) => {
      for (const item of items) item.style.minHeight = '';
    };

    const setSpans = (items: HTMLElement[]) => {
      const heights = items.map((item) => item.getBoundingClientRect().height);
      grid.style.gridAutoRows = '1px';
      grid.style.gridAutoFlow = 'row dense';
      grid.style.rowGap = '0px';
      items.forEach((item, index) => {
        item.style.gridRowEnd = `span ${Math.ceil(heights[index]) + GAP_PX}`;
      });
      packedState = layoutState();
    };

    /**
     * Find every column hole and give its height to the panel above it. Runs
     * on one settled layout, so stretching a short panel cannot move its
     * neighbours; bottom panels fill to the section foot.
     */
    const fillHoles = (items: HTMLElement[]) => {
      const gridRect = grid.getBoundingClientRect();
      const colWidth = (gridRect.width - GAP_PX * 11) / 12;
      if (colWidth <= 0) return false;

      const boxes = items.map((item) => {
        const r = item.getBoundingClientRect();
        return {
          item,
          x: Math.round((r.left - gridRect.left) / (colWidth + GAP_PX)),
          span: Math.max(1, Math.round((r.width + GAP_PX) / (colWidth + GAP_PX))),
          top: r.top - gridRect.top,
          bottom: r.bottom - gridRect.top,
          height: r.height,
        };
      });

      const gridBottom = Math.max(...boxes.map((box) => box.bottom));
      let changed = false;

      for (const box of boxes) {
        let room = Number.POSITIVE_INFINITY;
        for (let c = box.x; c < box.x + box.span; c += 1) {
          let nextTop = Number.POSITIVE_INFINITY;
          for (const other of boxes) {
            if (other === box || c < other.x || c >= other.x + other.span) continue;
            if (other.top >= box.bottom - 1 && other.top < nextTop) nextTop = other.top;
          }
          const ceiling = Number.isFinite(nextTop) ? nextTop - GAP_PX : gridBottom;
          room = Math.min(room, ceiling - box.bottom);
        }

        if (room <= MIN_STRETCH_PX) continue;
        box.item.style.minHeight = `${Math.ceil(box.height + room)}px`;
        changed = true;
      }

      return changed;
    };

    /**
     * Pack, fill and re-span in one synchronous pass. Reading offsetHeight
     * between writes forces the intermediate layouts, so every measurement
     * sees a settled grid.
     */
    const run = () => {
      frame = 0;
      if (disposed) return;
      const items = panels();

      if (!query.matches) {
        unstretch(items);
        grid.style.gridAutoRows = '';
        grid.style.gridAutoFlow = '';
        grid.style.rowGap = '';
        for (const item of items) item.style.gridRowEnd = '';
        packedState = layoutState();
        return;
      }

      unstretch(items);
      void grid.offsetHeight;
      setSpans(items);
      void grid.offsetHeight;
      if (fillHoles(items)) {
        setSpans(items);
        void grid.offsetHeight;
      }
      packedState = layoutState();
    };

    const request = () => {
      if (disposed || frame !== 0) return;
      if (layoutState() === packedState) return;
      frame = window.requestAnimationFrame(run);
    };

    run();

    const observer = new ResizeObserver(() => {
      if (disposed) return;
      request();
    });
    observer.observe(grid);
    for (const item of panels()) observer.observe(item);

    const onQueryChange = () => request();
    query.addEventListener('change', onQueryChange);
    document.fonts?.ready
      .then(() => {
        if (!disposed) request();
      })
      .catch(() => {});

    return () => {
      disposed = true;
      observer.disconnect();
      query.removeEventListener('change', onQueryChange);
      if (frame !== 0) cancelAnimationFrame(frame);
    };
  });

  return (
    <div ref={ref} data-slot="masonry-grid" className={cn('grid items-start gap-3 lg:grid-cols-12', className)}>
      {children}
    </div>
  );
}
