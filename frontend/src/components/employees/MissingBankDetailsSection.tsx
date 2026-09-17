import { useState, useCallback, useMemo } from 'react';
import { AlertTriangle, ChevronDown, ChevronUp, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useMissingBankDetails } from '@/hooks/employees/useMissingBankDetails';
import { VIETNAMESE_EMPLOYEE_LABELS } from '@/types/api/employee.types';
import type { Employee, CurrentProject } from '@/types/api/employee.types';
import {
  getBankInformationWarningKind,
  getBankInformationWarningReason,
} from '@/utils/bank-information-warning';

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

  // Group rows by warning kind, invalid first: OnePay-confirmed wrong data
  // outranks incomplete data. Server order is preserved within each group.
  const { rows, invalidCount, missingCount } = useMemo(() => {
    const invalid = employees.filter(
      employee => getBankInformationWarningKind(employee) === 'invalid',
    );
    const missing = employees.filter(
      employee => getBankInformationWarningKind(employee) !== 'invalid',
    );
    return {
      rows: [...invalid, ...missing],
      invalidCount: invalid.length,
      missingCount: missing.length,
    };
  }, [employees]);

  if (isLoading || totalCount === 0) return null;

  return (
    <div className="overflow-hidden rounded-xl border border-amber-200/80 bg-amber-50/50">
      {/* Header row */}
      <div className="flex items-center justify-between gap-2 px-3 py-2 sm:gap-3 sm:px-4 sm:py-3">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1">
          <div className="flex w-full items-start gap-2">
          <AlertTriangle className="h-4 w-4 shrink-0 text-amber-600" aria-hidden="true" />
          <span className="min-w-0 text-pretty text-sm font-semibold text-amber-900">
            Thông tin ngân hàng không hợp lệ
          </span>
          </div>
          {invalidCount > 0 && (
            <Badge className="border-rose-300 bg-rose-100 text-xs font-semibold text-rose-800 hover:bg-rose-100 shrink-0">
              Sai thông tin: {invalidCount}
            </Badge>
          )}
          {missingCount > 0 && (
            <Badge className="border-amber-300 bg-amber-100 text-xs font-semibold text-amber-800 hover:bg-amber-100 shrink-0">
              Thiếu thông tin: {missingCount}
            </Badge>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={toggle}
          aria-expanded={isExpanded}
          aria-controls="missing-bank-details-table"
          className="h-11 w-11 shrink-0 gap-1.5 px-0 text-xs text-amber-700 hover:bg-amber-100 hover:text-amber-900 sm:h-9 sm:w-auto sm:px-3"
        >
          {isExpanded ? (
            <><ChevronUp className="h-3.5 w-3.5" aria-hidden="true" /><span className="sr-only sm:not-sr-only">Ẩn</span></>
          ) : (
            <><ChevronDown className="h-3.5 w-3.5" aria-hidden="true" /><span className="sr-only sm:not-sr-only">Xem danh sách</span></>
          )}
        </Button>
      </div>

      {/* Expandable table */}
      {isExpanded && rows.length > 0 && (
        <div id="missing-bank-details-table" className="border-t border-amber-200">
          <div className="max-h-80 overflow-auto overscroll-contain focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring" role="region" aria-label="Danh sách cảnh báo thông tin ngân hàng" tabIndex={0}>
            <table className="w-full min-w-[760px] text-sm">
              <caption className="sr-only">
                Danh sách nhân viên có thông tin ngân hàng không hợp lệ
              </caption>
              <thead className="sticky top-0 z-10 bg-amber-100/80 backdrop-blur-sm">
                <tr>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900 w-12">STT</th>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">Họ và tên</th>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">CCCD</th>
                  {/* Reason column: why is this employee in the list? */}
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">Lý do</th>
                  <th className="text-left px-4 py-2.5 text-xs font-semibold text-amber-900">Dự án hiện tại</th>
                </tr>
              </thead>
              <tbody className="bg-card divide-y divide-amber-100">
                {rows.map((employee, index) => {
                  const hasPending = (employee.timesheet_summary?.pending_timesheets ?? 0) > 0;
                  const warningKind = getBankInformationWarningKind(employee);
                  const isInvalid = warningKind === 'invalid';
                  const warningReason = getBankInformationWarningReason(employee);
                  return (
                    <tr
                      key={employee.id}
                      onClick={() => onEmployeeClick?.(employee)}
                      onKeyDown={(event) => {
                        if (!onEmployeeClick) return;
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault();
                          onEmployeeClick(employee);
                        }
                      }}
                      tabIndex={onEmployeeClick ? 0 : undefined}
                      aria-label={
                        onEmployeeClick
                          ? `Mở thông tin nhân viên ${employee.fullname}`
                          : undefined
                      }
                      className={
                        onEmployeeClick
                          ? 'cursor-pointer transition-colors hover:bg-amber-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-amber-600'
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
                              <Clock className="h-2.5 w-2.5" aria-hidden="true" />
                              {employee.timesheet_summary!.pending_timesheets} chờ duyệt
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{employee.cccd}</td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap items-start gap-1">
                          <span
                            className={`inline-flex shrink-0 items-center gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium leading-4 ${
                              isInvalid
                                ? 'border-rose-300 bg-rose-100 text-rose-800'
                                : 'border-amber-300 bg-amber-100 text-amber-900'
                            }`}
                          >
                            <AlertTriangle className="h-2.5 w-2.5" aria-hidden="true" />
                            {isInvalid ? 'Sai thông tin' : 'Thiếu thông tin'}
                          </span>
                          <span className={`inline-flex max-w-[240px] items-start gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium leading-4 ${
                            isInvalid
                              ? 'border-rose-300 bg-rose-100 text-rose-800'
                              : 'border-amber-300 bg-amber-100 text-amber-900'
                          }`}>
                            <span className="break-words">{warningReason}</span>
                          </span>
                        </div>
                      </td>
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
