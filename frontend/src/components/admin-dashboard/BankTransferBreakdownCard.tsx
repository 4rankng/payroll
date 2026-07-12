import { memo, useState, useMemo } from 'react';
import { Building2, ChevronDown } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useBankUsageAllProjects } from '@/hooks/api/useDashboard';
import type { BankUsageItem, ProjectBankUsageItem } from '@/types/api/dashboard.types';
import { BankDonutChart, type BankSlice } from '@/components/admin-dashboard/BankDonutChart';

// ── palette ───────────────────────────────────────────────────────────────────
// Matches the mockup: dominant blue, then lighter blues, greens, ambers, reds, greys
const PALETTE = [
  '#1d4ed8', // deep blue  (rank 1)
  '#60a5fa', // sky blue   (rank 2)
  '#34d399', // emerald    (rank 3)
  '#fbbf24', // amber      (rank 4)
  '#f87171', // rose       (rank 5)
  '#94a3b8', // slate      (others)
  '#a78bfa', // violet
  '#fb923c', // orange
  '#4ade80', // green
  '#38bdf8', // light sky
];

const OTHER_COLOR = '#94a3b8';
const MAX_SLICES = 5; // top 5 named, rest → "Khác"

// ── helpers ───────────────────────────────────────────────────────────────────
function extractShortName(bankName: string): string {
  const match = bankName.match(/\(([^)]+)\)/);
  return match ? match[1] : bankName.slice(0, 3).toUpperCase();
}

function buildSlices(banks: BankUsageItem[], totalEmployees: number): BankSlice[] {
  if (!banks.length) return [];

  const sorted = [...banks].sort((a, b) => b.employee_count - a.employee_count);
  const top = sorted.slice(0, MAX_SLICES);
  const rest = sorted.slice(MAX_SLICES);

  const slices: BankSlice[] = top.map((b, i) => ({
    name: b.bank_name,
    shortName: extractShortName(b.bank_name),
    value: b.employee_count,
    pct: totalEmployees > 0 ? Math.round((b.employee_count / totalEmployees) * 100) : 0,
    color: PALETTE[i] ?? PALETTE[PALETTE.length - 1],
    transfers: b.transfer_count,
    totalPaid: b.total_paid_vnd,
  }));

  if (rest.length > 0) {
    const otherCount = rest.reduce((s, b) => s + b.employee_count, 0);
    const otherTransfers = rest.reduce((s, b) => s + b.transfer_count, 0);
    const otherPaid = rest.reduce((s, b) => s + b.total_paid_vnd, 0);
    slices.push({
      name: 'Ngân hàng khác',
      shortName: 'Khác',
      value: otherCount,
      pct: totalEmployees > 0 ? Math.round((otherCount / totalEmployees) * 100) : 0,
      color: OTHER_COLOR,
      transfers: otherTransfers,
      totalPaid: otherPaid,
    });
  }

  return slices;
}

// ── Project selector ──────────────────────────────────────────────────────────
interface ProjectSelectorProps {
  projects: ProjectBankUsageItem[];
  selectedId: number | null;
  onSelect: (id: number | null) => void;
}

export const ProjectSelector = memo(function ProjectSelector({
  projects,
  selectedId,
  onSelect,
}: ProjectSelectorProps) {
  const [open, setOpen] = useState(false);
  const selected = projects.find((p) => p.project_id === selectedId);

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 text-xs border border-border/60 rounded-lg px-2.5 py-1.5 bg-card hover:bg-muted/40 transition-colors"
      >
        <span className="max-w-[140px] truncate text-foreground">
          {selected ? selected.project_name : 'Tất cả dự án'}
        </span>
        <ChevronDown className={`w-3.5 h-3.5 text-muted-foreground transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 bg-card border border-border rounded-xl shadow-lg min-w-[180px] max-h-60 overflow-y-auto py-1">
          <button
            onClick={() => { onSelect(null); setOpen(false); }}
            className={`w-full text-left px-3 py-2 text-xs hover:bg-muted/50 transition-colors ${selectedId === null ? 'font-semibold text-primary' : 'text-foreground'}`}
          >
            Tất cả dự án
          </button>
          {projects.map((p) => (
            <button
              key={p.project_id}
              onClick={() => { onSelect(p.project_id); setOpen(false); }}
              className={`w-full text-left px-3 py-2 text-xs hover:bg-muted/50 transition-colors ${selectedId === p.project_id ? 'font-semibold text-primary' : 'text-foreground'}`}
            >
              <span className="truncate block">{p.project_name}</span>
              <span className="text-[11px] text-muted-foreground">{p.total_employees} NV</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
});

// ── Main card ─────────────────────────────────────────────────────────────────
export const BankTransferBreakdownCard = memo(function BankTransferBreakdownCard({
  selectedProjectId = null,
}: {
  selectedProjectId?: number | null;
}) {
  const { data: allData, isLoading: allLoading } = useBankUsageAllProjects();

  const isLoading = allLoading;

  // Determine which data to show
  const displayData = useMemo(() => {
    if (selectedProjectId !== null && allData) {
      const proj = allData.projects.find((p) => p.project_id === selectedProjectId);
      if (proj) {
        return {
          slices: buildSlices(proj.banks, proj.total_employees),
          totalEmployees: proj.total_employees,
        };
      }
    }
    // Overall
    const src = allData?.overall;
    if (!src) return null;
    return {
      slices: buildSlices(src.banks, src.total_employees),
      totalEmployees: src.total_employees,
    };
  }, [selectedProjectId, allData]);

  return (
    <Card className="shadow-card">
      <CardContent className="pt-0">
        {isLoading ? (
          <div>
            <div className="flex justify-center">
              <Skeleton className="w-[280px] h-[280px] rounded-full" />
            </div>
            <div className="mt-4 space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center justify-between py-1">
                  <div className="flex items-center gap-2">
                    <Skeleton className="w-2.5 h-2.5 rounded-full" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                  <Skeleton className="h-3 w-8" />
                </div>
              ))}
            </div>
          </div>
        ) : displayData ? (
          <BankDonutChart
            slices={displayData.slices}
            totalEmployees={displayData.totalEmployees}
          />
        ) : (
          <div className="flex flex-col items-center justify-center py-10 text-center">
            <Building2 className="w-8 h-8 text-muted-foreground/30 mb-2" />
            <p className="text-xs text-muted-foreground">Chưa có dữ liệu ngân hàng</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
});
