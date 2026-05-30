import { Users, User, MapPin } from "lucide-react";
import { cn } from "@/lib/utils";

interface Employee {
  id: number;
  name: string;
  email: string;
  phone: string;
  position: string;
  project: string;
  status: string;
}

interface PartnerEmployeesStatsProps {
  employees: Employee[];
}

// Watermark tokens — small inline icon + large faint icon decoration.
const STATS_CONFIG = [
  { icon: Users,  iconText: 'text-sky-600',     watermark: 'text-sky-500/15' },
  { icon: User,   iconText: 'text-emerald-600', watermark: 'text-emerald-500/15' },
  { icon: MapPin, iconText: 'text-violet-600',  watermark: 'text-violet-500/15' },
];

export const PartnerEmployeesStats = ({ employees }: PartnerEmployeesStatsProps) => {
  const activeEmployees = employees.filter(emp => emp.status === 'Đang làm việc');
  const uniqueProjects = new Set(employees.map(emp => emp.project));

  const stats = [
    { label: "Tổng nhân viên", value: employees.length },
    { label: "Đang làm việc", value: activeEmployees.length },
    { label: "Dự án", value: uniqueProjects.size },
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
            <div className="relative pr-10">
              <div className="flex items-center gap-1.5">
                <Icon className={cn('h-3 w-3 shrink-0', iconText)} strokeWidth={2.2} />
                <span className="text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight truncate">
                  {stat.label}
                </span>
              </div>
              <p className="mt-1 text-[15px] font-semibold tabular-nums text-slate-800 leading-tight truncate">
                {stat.value.toLocaleString('vi-VN')}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
};
