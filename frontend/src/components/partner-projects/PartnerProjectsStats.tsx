import { FolderOpen, Users, TrendingUp } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ProjectSummary } from '@/types/api/project.types';

interface PartnerProjectsStatsProps {
  summary: ProjectSummary;
}

// Watermark tokens — small inline icon + large faint icon decoration.
const STATS_CONFIG = [
  { icon: FolderOpen,  iconText: 'text-sky-600',     watermark: 'text-sky-500/15' },
  { icon: Users,       iconText: 'text-emerald-600', watermark: 'text-emerald-500/15' },
  { icon: TrendingUp,  iconText: 'text-violet-600',  watermark: 'text-violet-500/15' },
];

export const PartnerProjectsStats = ({ summary }: PartnerProjectsStatsProps) => {
  const stats = [
    { label: "Dự án hoạt động", value: summary.total_active_projects },
    { label: "Tổng nhân viên", value: "-" },
    { label: "Hiệu suất", value: "98%" },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
      {stats.map((stat, index) => {
        const { icon: Icon, iconText, watermark } = STATS_CONFIG[index];
        return (
          <div
            key={index}
            className="group relative rounded-xl border border-border bg-card/70 px-3 py-2.5 overflow-hidden transition-colors hover:bg-muted/40"
            style={{ backdropFilter: 'blur(8px)', boxShadow: '0 1px 4px rgba(2,132,199,0.07)' }}
          >
            <Icon
              className={cn(
                'absolute right-2 top-1/2 -translate-y-1/2 h-10 w-10 pointer-events-none',
                'transition-transform duration-300 group-hover:scale-105',
                watermark,
              )}
              strokeWidth={1.5}
            />
            <div className="relative min-w-0 pr-10">
              <div className="flex items-center gap-1.5">
                <Icon className={cn('h-3 w-3 shrink-0', iconText)} strokeWidth={2.2} />
                <span className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight line-clamp-2">
                  {stat.label}
                </span>
              </div>
              <p className="mt-1 break-words text-[15px] font-semibold tabular-nums text-slate-800 leading-tight">
                {stat.value}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
};
