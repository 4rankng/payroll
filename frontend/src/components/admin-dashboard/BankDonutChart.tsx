import { memo, useRef, useEffect, useCallback } from 'react';
import {
  Chart,
  DoughnutController,
  ArcElement,
  type ChartConfiguration,
} from 'chart.js';

Chart.register(DoughnutController, ArcElement);

// ── types ─────────────────────────────────────────────────────────────────────
export interface BankSlice {
  name: string;
  shortName: string;
  value: number;
  pct: number;
  color: string;
  transfers?: number;
  totalPaid?: number;
}

interface BankDonutChartProps {
  slices: BankSlice[];
  totalEmployees: number;
}

// ── constants ──────────────────────────────────────────────────────────────────
const INSIDE_THRESHOLD = 15;

// ── custom plugin: inside + outside labels with polyline connectors ────────────
function createLabelPlugin(banks: BankSlice[]) {
  return {
    id: 'sliceLabels',
    afterDraw(chart: Chart) {
      const ctx = chart.ctx;
      const meta = chart.getDatasetMeta(0);
      const { left, top, width, height } = chart.chartArea;
      const cx = left + width / 2;
      const cy = top + height / 2;

      // Inside labels for large slices
      meta.data.forEach((arc, i) => {
        const b = banks[i];
        if (!b || b.pct < INSIDE_THRESHOLD) return;
        const mid = (arc.startAngle + arc.endAngle) / 2;
        const r = (arc.outerRadius + arc.innerRadius) / 2;
        const lx = cx + Math.cos(mid) * r;
        const ly = cy + Math.sin(mid) * r;
        ctx.save();
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.shadowBlur = 4;
        ctx.shadowColor = 'rgba(0,0,0,0.35)';
        ctx.fillStyle = '#ffffff';
        ctx.font = '500 13px system-ui,sans-serif';
        ctx.fillText(b.shortName, lx, ly - 7);
        ctx.font = '500 12px system-ui,sans-serif';
        ctx.fillText(b.pct + '%', lx, ly + 7);
        ctx.restore();
      });

      // Outside label candidates
      const items: Array<{
        b: BankSlice; edgeX: number; edgeY: number;
        elbowX: number; elbowY: number; x: number; y: number;
      }> = [];
      meta.data.forEach((arc, i) => {
        const b = banks[i];
        if (!b || b.pct >= INSIDE_THRESHOLD) return;
        const mid = (arc.startAngle + arc.endAngle) / 2;
        const sliceOff = (arc.options as { offset?: number })?.offset ?? 0;
        const totalR = arc.outerRadius + sliceOff;
        const edgeX = cx + Math.cos(mid) * totalR;
        const edgeY = cy + Math.sin(mid) * totalR;
        const elbowR = totalR + 14;
        const elbowX = cx + Math.cos(mid) * elbowR;
        const elbowY = cy + Math.sin(mid) * elbowR;
        const natR = totalR + 18;
        const natX = cx + Math.cos(mid) * natR;
        const natY = cy + Math.sin(mid) * natR;
        items.push({ b, edgeX, edgeY, elbowX, elbowY, x: natX, y: natY });
      });

      // Collision resolution
      const MIN_DY = 26;
      const MAX_DX = 75;
      for (let iter = 0; iter < 40; iter++) {
        for (let a = 0; a < items.length; a++) {
          for (let bb = a + 1; bb < items.length; bb++) {
            const A = items[a], B = items[bb];
            const dx = Math.abs(B.x - A.x);
            const dy = Math.abs(B.y - A.y);
            if (dx < MAX_DX && dy < MIN_DY) {
              const push = (MIN_DY - dy) / 2 + 1;
              if (B.y >= A.y) { A.y -= push; B.y += push; }
              else { A.y += push; B.y -= push; }
            }
          }
        }
      }

      // Draw connectors and labels
      items.forEach(({ b, edgeX, edgeY, elbowX, elbowY, x, y }) => {
        const isRight = elbowX > cx;
        const textX = x + (isRight ? 6 : -6);

        ctx.save();
        ctx.beginPath();
        ctx.moveTo(edgeX, edgeY);
        ctx.lineTo(elbowX, elbowY);
        ctx.lineTo(x, y);
        ctx.strokeStyle = b.color;
        ctx.lineWidth = 1.2;
        ctx.stroke();

        ctx.beginPath();
        ctx.arc(x, y, 2.5, 0, Math.PI * 2);
        ctx.fillStyle = b.color;
        ctx.fill();

        ctx.textAlign = isRight ? 'left' : 'right';
        ctx.textBaseline = 'middle';
        ctx.fillStyle = b.color;
        ctx.font = '600 11px system-ui,sans-serif';
        ctx.fillText(b.shortName, textX, y - 7);
        ctx.font = '400 11px system-ui,sans-serif';
        ctx.fillText(b.pct + '%', textX, y + 6);
        ctx.restore();
      });
    },
  };
}

// ── detect card background ─────────────────────────────────────────────────────
function getCardBg(): string {
  const el = document.querySelector('[data-card-bg]')?.parentElement
    ?? document.querySelector('.shadow-card');
  if (el) {
    const bg = getComputedStyle(el).backgroundColor;
    if (bg && bg !== 'rgba(0, 0, 0, 0)') return bg;
  }
  return matchMedia('(prefers-color-scheme: dark)').matches ? '#1e1e1e' : '#ffffff';
}

// ── component ──────────────────────────────────────────────────────────────────
export const BankDonutChart = memo(function BankDonutChart({
  slices,
  totalEmployees,
}: BankDonutChartProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const chartRef = useRef<Chart | null>(null);

  const buildChart = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    if (chartRef.current) {
      chartRef.current.destroy();
      chartRef.current = null;
    }

    if (!slices.length) return;

    const bgColor = getCardBg();
    const labelPlugin = createLabelPlugin(slices);

    const config: ChartConfiguration<'doughnut'> = {
      type: 'doughnut',
      plugins: [labelPlugin],
      data: {
        labels: slices.map((b) => b.shortName),
        datasets: [{
          data: slices.map((b) => b.pct),
          backgroundColor: slices.map((b) => b.color),
          borderColor: bgColor,
          borderWidth: 3,
          hoverOffset: 4,
          offset: slices.map((b) => b.pct < INSIDE_THRESHOLD ? 14 : 0),
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        cutout: '60%',
        layout: { padding: { top: 55, bottom: 30, left: 70, right: 30 } },
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: (ctx) => ` ${slices[ctx.dataIndex].shortName} — ${slices[ctx.dataIndex].pct}%`,
            },
          },
        },
      },
    };

    chartRef.current = new Chart(canvas, config);
  }, [slices]);

  useEffect(() => {
    buildChart();
    return () => {
      chartRef.current?.destroy();
      chartRef.current = null;
    };
  }, [buildChart]);

  if (!slices.length) return null;

  return (
    <div>
      <div className="flex justify-center">
        <div className="relative" style={{ width: 320, height: 320 }}>
          <canvas
            ref={canvasRef}
            role="img"
            aria-label={`Donut chart: ${slices.map((b) => `${b.shortName} ${b.pct}%`).join(', ')}`}
          />
          <div
            className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 text-center pointer-events-none"
            style={{ marginTop: -5 }}
          >
            <div className="text-[28px] font-medium text-foreground leading-tight tabular-nums">
              {totalEmployees.toLocaleString('vi-VN')}
            </div>
            <div className="text-xs text-muted-foreground mt-[3px]">nhân viên</div>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-[11px] mt-4">
        {slices.map((b, i) => (
          <LegendItem key={b.shortName} slice={b} isTop={i === 0} />
        ))}
      </div>
    </div>
  );
});

// ── helpers ──────────────────────────────────────────────────────────────────
function formatVND(value: number): string {
  const abs = Math.abs(value);
  if (abs >= 1e9) return `${(abs / 1e9).toFixed(1)}B đ`;
  if (abs >= 1e6) return `${(abs / 1e6).toFixed(1)}M đ`;
  if (abs >= 1e3) return `${(abs / 1e3).toFixed(0)}K đ`;
  return `${abs.toLocaleString('vi-VN')} đ`;
}

// ── Legend item ────────────────────────────────────────────────────────────────
const LegendItem = memo(function LegendItem({ slice, isTop }: { slice: BankSlice; isTop: boolean }) {
  return (
    <div
      className="flex items-center justify-between gap-2 py-1.5 border-b border-border/30 last:border-0"
    >
      <div className="flex items-center gap-2 min-w-0">
        <span
          className="w-2.5 h-2.5 rounded-full shrink-0"
          style={{ backgroundColor: slice.color }}
        />
        <span className="text-xs text-foreground truncate">{slice.name}</span>
        {isTop && (slice.transfers ?? 0) > 0 && (
          <span className="text-[10px] text-muted-foreground shrink-0 tabular-nums">
            {slice.transfers!.toLocaleString('vi-VN')} lần
          </span>
        )}
        {isTop && (slice.totalPaid ?? 0) > 0 && (
          <span className="text-[10px] text-muted-foreground shrink-0 tabular-nums">
            / {formatVND(slice.totalPaid!)}
          </span>
        )}
      </div>
      <span className="text-xs font-semibold tabular-nums shrink-0" style={{ color: slice.color }}>
        {slice.pct}%
      </span>
    </div>
  );
});
