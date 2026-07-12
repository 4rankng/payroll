import { useState, useCallback, useMemo } from 'react';
import { AlertTriangle, ChevronDown, ChevronUp, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useMissingBankDetails } from '@/hooks/employees/useMissingBankDetails';
import { VIETNAMESE_EMPLOYEE_LABELS } from '@/types/api/employee.types';
import type { Employee, CurrentProject } from '@/types/api/employee.types';

interface MissingBankDetailsSectionProps {
  onEmployeeClick?: (employee: Employee) => void;
}

const SCHEDULE_LABEL: Record<string, string> = VIETNAMESE_EMPLOYEE_LABELS.schedule;

function ProjectCell({ projects }: { projects: CurrentProject[] }) {
  if (!projects?.length) {
    return <span className="text-xs text-muted-foreground italic">Chưa có dự án</span>;
  }
  return (
    <div className="flex flex-col gap-1">
      {projects.map((p, i) => (
        <div key={i} className="flex items-center gap-1.5 flex-wrap">
          <span className="font-mono text-xs font-medium text-foreground">{p.code}</span>
          <span className="text-xs text-muted-foreground">{p.name}</span>
          {p.payment_schedule && (
            <Badge variant="secondary" className="text-[11px] px-1.5 py-0 h-4 font-normal">
              {SCHEDULE_LABEL[p.payment_schedule] ?? p.payment_schedule}
            </Badge>
          )}
          {p.position && (
            <span className="text-[11px] text-muted-foreground">• {p.position}</span>
          )}
        </div>
      ))}
    </div>
  );
}

export const MissingBankDetailsSection = ({
  onEmployeeClick,
}: MissingBankDetailsSectionProps) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const { employees, totalCount, isLoading } = useMissingBankDetails();

  const toggle = useCallback(() => setIsExpanded(v => !v), []);

  const rows = useMemo(() => employees, [employees]);

  if (isLoading || totalCount === 0) return null;

  return (
    <div className="rounded-xl border border-amber-200/80 bg-amber-50/50 overflow-hidden" style={{ boxShadow: '0 1px 4px rgba(217,119,6,0.08)' }}>
      {/* Header row */}
      <div className="flex items-center justify-between px-4 py-3 gap-3">
        <div className="flex items-center gap-2.5 min-w-0">
          <AlertTriangle className="h-4 w-4 text-amber-600 shrink-0" />
          <span className="text-sm font-semibold text-amber-900 truncate">
            Nhân viên thiếu thông tin ngân hàng
          </span>
          <Badge className="bg-amber-100 text-amber-800 border-amber-300 hover:bg-amber-100 text-xs font-semibold shrink-0">
            {totalCount}
          </Badge>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={toggle}
          className="h-7 gap-1.5 text-amber-700 hover:text-amber-900 hover:bg-amber-100 text-xs shrink-0"
        >
          {isExpanded ? (
            <><ChevronUp className="h-3.5 w-3.5" />Ẩn</>
          ) : (
            <><ChevronDown className="h-3.5 w-3.5" />Xem danh sách</>
          )}
        </Button>
      </div>

      {/* Expandable table */}
      {isExpanded && rows.length > 0 && (
        <div className="border-t border-amber-200">
          <div className="max-h-80 overflow-y-auto">
            <table className="w-full text-sm">
              <thead className="sticky top-0 z-10 bg-amber-100/80 backdrop-blur-sm">
                <tr>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900 w-12">STT</th>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">Họ và tên</th>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">CCCD</th>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">Dự án hiện tại</th>
                </tr>
              </thead>
              <tbody className="bg-card divide-y divide-amber-100">
                {rows.map((employee, index) => {
                  const hasPending = (employee.timesheet_summary?.pending_timesheets ?? 0) > 0;
                  return (
                    <tr
                      key={employee.id}
                      onClick={() => onEmployeeClick?.(employee)}
                      className={
                        onEmployeeClick
                          ? 'hover:bg-amber-50 cursor-pointer transition-colors'
                          : ''
                      }
                    >
                      <td className="px-4 py-3 text-xs text-muted-foreground tabular-nums">{index + 1}</td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-foreground text-sm">{employee.fullname}</span>
                          {hasPending && (
                            <span
                              title={`${employee.timesheet_summary!.pending_timesheets} bảng công chờ duyệt`}
                              className="inline-flex items-center gap-1 text-[11px] font-medium text-orange-700 bg-orange-100 border border-orange-200 rounded px-1.5 py-0.5"
                            >
                              <Clock className="h-2.5 w-2.5" />
                              {employee.timesheet_summary!.pending_timesheets} chờ duyệt
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{employee.cccd}</td>
                      <td className="px-4 py-3">
                        <ProjectCell projects={employee.current_projects} />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
