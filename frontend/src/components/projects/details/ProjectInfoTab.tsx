import { useMemo } from "react";
import { Project } from "@/types/api/project.types";
import { Info, Clock, Wallet, CalendarOff } from "lucide-react";
import { useUsersByIds } from "@/hooks/api/useUsers";
import { getUserFullName } from "@/utils/userHelpers";
import { authManager } from "@/lib/auth";
import { isOffDay } from "@/components/projects/OffDaysPicker";
import { formatDate } from "@/utils/formatters";

const DAY_LABELS = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];

// Small inline icon + label heading for each info section.
function SectionLabel({ icon: Icon, children }: { icon: React.ComponentType<{ className?: string }>; children: React.ReactNode }) {
  return (
    <p className="inline-flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1.5">
      <Icon className="h-3 w-3 opacity-70" />
      {children}
    </p>
  );
}

interface ProjectInfoTabProps {
  project: Project;
}

const formatDateForDisplay = (dateString: string | null | undefined): string => {
  if (!dateString) return "-";
  return formatDate(dateString);
};

function Field({ label, value, mono }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-2 border-b border-border/40 last:border-0">
      <span className="text-xs text-muted-foreground shrink-0">{label}</span>
      <span className={`text-xs font-medium text-right ${mono ? 'font-mono' : ''}`}>{value}</span>
    </div>
  );
}

export function ProjectInfoTab({ project }: ProjectInfoTabProps) {
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';

  const userIds = useMemo(() => {
    return project?.created_by ? [project.created_by] : [];
  }, [project?.created_by]);

  const { data: userMap, isLoading: isLoadingCreator } = useUsersByIds(userIds);
  const creatorName = getUserFullName(project?.created_by, userMap);

  const offDays = project.off_days ?? 0;

  return (
    <div className="p-4 space-y-4">
      {/* Basic Info */}
      <section>
        <SectionLabel icon={Info}>Thông tin cơ bản</SectionLabel>
        <div className="bg-muted/30 rounded-xl px-3 py-1">
          <Field label="Tên dự án" value={project.name || '-'} />
          <Field label="Mã dự án" value={project.code || '-'} mono />
          <Field label="Khách hàng" value={project.client_name || '-'} />
          <Field
            label="Người tạo"
            value={isLoadingCreator ? <span className="text-muted-foreground">...</span> : creatorName}
          />
        </div>
      </section>

      {/* Time */}
      <section>
        <SectionLabel icon={Clock}>Thời gian</SectionLabel>
        <div className="bg-muted/30 rounded-xl px-3 py-1">
          <Field label="Ngày bắt đầu" value={formatDateForDisplay(project.start_date)} />
          <Field label="Ngày kết thúc" value={formatDateForDisplay(project.end_date)} />
          {project.description && (
            <div className="py-1.5 border-t border-border/50">
              <p className="text-xs text-muted-foreground mb-0.5">Mô tả</p>
              <p className="text-xs leading-relaxed">{project.description}</p>
            </div>
          )}
        </div>
      </section>

      {/* Salary Period + Off Days side by side on sm+, stacked on mobile */}
      <section className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <SectionLabel icon={Wallet}>Kỳ lương tháng</SectionLabel>
          <div className="bg-muted/30 rounded-xl px-3 py-1">
            <Field
              label="Bắt đầu"
              value={project.salary_period_from ? `Ngày ${project.salary_period_from}` : 'Ngày 01'}
            />
            <Field
              label="Kết thúc"
              value={project.salary_period_to ? `Ngày ${project.salary_period_to}` : 'Cuối tháng'}
            />
          </div>
        </div>

        <div>
          <SectionLabel icon={CalendarOff}>Ngày nghỉ tuần</SectionLabel>
          <div className="bg-muted/30 rounded-xl px-3 py-2">
            {offDays === 0 ? (
              <p className="text-xs text-muted-foreground">Không có</p>
            ) : (
              <div className="space-y-1.5">
                <div className="flex flex-wrap gap-1">
                  {DAY_LABELS.map((label, bit) => (
                    <span
                      key={bit}
                      className={`px-1.5 py-0.5 rounded text-xs font-semibold border ${
                        isOffDay(offDays, bit)
                          ? 'bg-red-100 text-red-600 border-red-400'
                          : 'bg-emerald-50 text-emerald-700 border-emerald-300'
                      }`}
                    >
                      {label}
                    </span>
                  ))}
                </div>
                <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <span className="inline-block w-2.5 h-2.5 rounded-sm bg-emerald-100 border border-emerald-300" />
                    Ngày thường
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="inline-block w-2.5 h-2.5 rounded-sm bg-red-100 border border-red-400" />
                    Ngày nghỉ
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}
