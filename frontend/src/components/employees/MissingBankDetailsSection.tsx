import { useState, useCallback, useMemo } from 'react';
import { AlertTriangle, ChevronDown, ChevronUp, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useMissingBankDetails } from "@/hooks/employees/useMissingBankDetails";
import { useIsMobile } from "@/hooks/useBreakpoint";
import { cn } from "@/lib/utils";
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
            <Badge variant="secondary" className="text-xs px-1.5 py-0 h-5 font-normal">
              {SCHEDULE_LABEL[p.payment_schedule] ?? p.payment_schedule}
            </Badge>
          )}
          {p.position && (
            <span className="text-xs text-muted-foreground">• {p.position}</span>
          )}
        </div>
      ))}
    </div>
  );
}

/**
 * The two reason chips shown for an employee: the warning kind, then the
 * specific reason text. Shared by the mobile card list and the desktop table so
 * both surfaces state the problem identically.
 */
function WarningReasons({
  employee,
  reasonClassName,
}: {
  employee: Employee;
  reasonClassName?: string;
}) {
  const isInvalid = getBankInformationWarningKind(employee) === 'invalid';
  const warningReason = getBankInformationWarningReason(employee);
  const tone = isInvalid
    ? 'border-rose-300 bg-rose-100 text-rose-800'
    : 'border-amber-300 bg-amber-100 text-amber-900';

  return (
    <div className="flex flex-wrap items-start gap-1">
      <span
        className={cn(
          'inline-flex shrink-0 items-center gap-1 rounded border px-1.5 py-0.5 text-xs font-medium leading-4',
          tone,
        )}
      >
        <AlertTriangle className="h-2.5 w-2.5" aria-hidden="true" />
        {isInvalid ? 'Sai thông tin' : 'Thiếu thông tin'}
      </span>
      <span
        className={cn(
          'inline-flex items-start gap-1 rounded border px-1.5 py-0.5 text-xs font-medium leading-4',
          tone,
          reasonClassName,
        )}
      >
        <span className="break-words">{warningReason}</span>
      </span>
    </div>
  );
}

export const MissingBankDetailsSection = ({
  onEmployeeClick,
}: MissingBankDetailsSectionProps) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const isMobile = useIsMobile();
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
    <div className="overflow-hidden rounded-xl border border-amber-200 bg-card">
      {/* Designed as the app's card + icon-chip component rather than a tinted
          block: amber icon chip, foreground title, and counts as compact
          colour-coded chips that read as data, not as controls. */}
      <div className="flex items-start gap-2.5 px-3 py-2.5 sm:px-4 sm:py-3">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-100 text-amber-700">
          <AlertTriangle className="h-4 w-4" aria-hidden="true" />
        </div>

        <div className="min-w-0 flex-1">
          <p className="text-pretty text-sm font-semibold leading-tight text-foreground">
            Thông tin ngân hàng không hợp lệ
          </p>
          {(invalidCount > 0 || missingCount > 0) && (
            <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
              {invalidCount > 0 && (
                <span className="inline-flex items-center gap-1 rounded-md bg-rose-50 px-1.5 py-0.5 text-xs font-semibold text-rose-700">
                  Sai thông tin
                  <span className="tabular-nums">{invalidCount}</span>
                </span>
              )}
              {missingCount > 0 && (
                <span className="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-xs font-semibold text-amber-700">
                  Thiếu thông tin
                  <span className="tabular-nums">{missingCount}</span>
                </span>
              )}
            </div>
          )}
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={toggle}
          aria-expanded={isExpanded}
          aria-controls="missing-bank-details-table"
          className="-mr-1 -my-1.5 h-11 w-11 shrink-0 px-0 text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          {isExpanded ? (
            <ChevronUp className="h-4 w-4" aria-hidden="true" />
          ) : (
            <ChevronDown className="h-4 w-4" aria-hidden="true" />
          )}
          <span className="sr-only">
            {isExpanded ? "Ẩn danh sách" : "Xem danh sách"}
          </span>
        </Button>
      </div>

      {/* Expanded list. Phones get one card per employee — a 760px-wide table
          cannot be read at 390px, it just scrolls sideways with cut columns. */}
      {isExpanded && rows.length > 0 && (
        <div
          id="missing-bank-details-table"
          className="border-t border-amber-200"
        >
          {/* Phones get a card list; the table is a desktop-density view. Chosen
              in JS rather than by CSS so only one variant exists in the DOM. */}
          {isMobile ? (
            <ul
              className="divide-y divide-amber-100"
              aria-label="Danh sách cảnh báo thông tin ngân hàng"
            >
            {rows.map((employee, index) => {
              const hasPending =
                (employee.timesheet_summary?.pending_timesheets ?? 0) > 0;
              const content = (
                <>
                  <div className="flex min-w-0 items-start justify-between gap-2">
                    <span className="min-w-0 break-words text-sm font-semibold text-foreground">
                      {index + 1}. {employee.fullname}
                    </span>
                    {hasPending && (
                      <span className="inline-flex shrink-0 items-center gap-1 rounded border border-orange-200 bg-orange-100 px-1.5 py-0.5 text-xs font-medium text-orange-700">
                        <Clock className="h-2.5 w-2.5" aria-hidden="true" />
                        {employee.timesheet_summary!.pending_timesheets} chờ duyệt
                      </span>
                    )}
                  </div>
                  <span className="font-mono text-xs tabular-nums text-muted-foreground">
                    {employee.cccd}
                  </span>
                  <WarningReasons employee={employee} />
                  <ProjectCell projects={employee.current_projects} />
                </>
              );

              return (
                <li key={employee.id}>
                  {onEmployeeClick ? (
                    <button
                      type="button"
                      onClick={() => onEmployeeClick(employee)}
                      aria-label={`Mở thông tin nhân viên ${employee.fullname}`}
                      className="flex w-full flex-col items-start gap-1.5 px-3 py-3 text-left transition-colors active:bg-amber-50"
                    >
                      {content}
                    </button>
                  ) : (
                    <div className="flex flex-col items-start gap-1.5 px-3 py-3">
                      {content}
                    </div>
                  )}
                </li>
              );
            })}
            </ul>
          ) : (
            <div>
              <div
                className="max-h-80 overflow-auto overscroll-contain focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                role="region"
              aria-label="Danh sách cảnh báo thông tin ngân hàng"
              tabIndex={0}
            >
              <table className="w-full min-w-[760px] text-sm">
                <caption className="sr-only">
                  Danh sách nhân viên có thông tin ngân hàng không hợp lệ
                </caption>
                <thead className="sticky top-0 z-10 bg-amber-100/80 backdrop-blur-sm">
                  <tr>
                    <th className="w-12 px-4 py-2.5 text-left text-xs font-semibold text-amber-900">STT</th>
                    <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Họ và tên</th>
                    <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">CCCD</th>
                    {/* Reason column: why is this employee in the list? */}
                    <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Lý do</th>
                    <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Dự án hiện tại</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-amber-100 bg-card">
                  {rows.map((employee, index) => {
                    const hasPending = (employee.timesheet_summary?.pending_timesheets ?? 0) > 0;
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
                        <td className="px-4 py-3 text-xs tabular-nums text-muted-foreground">{index + 1}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-foreground">{employee.fullname}</span>
                            {hasPending && (
                              <span
                                title={`${employee.timesheet_summary!.pending_timesheets} bảng công chờ duyệt`}
                                className="inline-flex items-center gap-1 rounded border border-orange-200 bg-orange-100 px-1.5 py-0.5 text-xs font-medium text-orange-700"
                              >
                                <Clock className="h-2.5 w-2.5" aria-hidden="true" />
                                {employee.timesheet_summary!.pending_timesheets} chờ duyệt
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{employee.cccd}</td>
                        <td className="px-4 py-3">
                          <WarningReasons employee={employee} reasonClassName="max-w-[240px]" />
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
      )}
    </div>
  );
};
