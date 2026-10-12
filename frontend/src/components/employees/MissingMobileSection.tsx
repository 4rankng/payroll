import { useState, useCallback } from 'react';
import type { KeyboardEvent } from 'react';
import { AlertTriangle, ChevronDown, ChevronUp, Phone } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useMissingMobile, useDuplicateMobiles } from '@/hooks/employees/useMissingMobile';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { VIETNAMESE_EMPLOYEE_LABELS } from '@/types/api/employee.types';
import type {
  Employee,
  CurrentProject,
  DuplicateMobileGroup,
  DuplicateMobileMember,
} from '@/types/api/employee.types';

interface MissingMobileSectionProps {
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
 * Shared activation props for desktop table rows: pointer and keyboard users
 * get the same action. Mobile cards use a real <button> instead, which brings
 * Enter/Space activation for free.
 */
function makeRowProps(
  employee: Employee,
  onEmployeeClick?: (employee: Employee) => void,
) {
  if (!onEmployeeClick) return {};
  return {
    onClick: () => onEmployeeClick(employee),
    onKeyDown: (event: KeyboardEvent) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        onEmployeeClick(employee);
      }
    },
    tabIndex: 0,
    'aria-label': `Mở thông tin nhân viên ${employee.fullname}`,
  };
}

const CLICKABLE_ROW_CLASS =
  'cursor-pointer transition-colors hover:bg-amber-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-amber-600';

/**
 * Who else holds this number. In-scope co-holders are named because the
 * partner can act on them; out-of-scope members are only ever counted —
 * inventing names for them would send the partner chasing people they
 * cannot see or edit.
 */
function CoHolders({
  group,
  member,
}: {
  group: DuplicateMobileGroup;
  member: DuplicateMobileMember;
}) {
  const others = group.employees.filter(e => e.id !== member.id);
  return (
    <div className="flex flex-wrap items-start gap-1">
      {others.length > 0 && (
        <span className="text-xs text-muted-foreground">
          Dùng chung với: {others.map(e => e.fullname).join(', ')}
        </span>
      )}
      {group.hidden_count > 0 && (
        <span className="inline-flex shrink-0 items-center rounded border border-amber-300 bg-amber-100 px-1.5 py-0.5 text-xs font-medium leading-4 text-amber-900">
          {group.hidden_count} nhân viên ngoài phạm vi
        </span>
      )}
    </div>
  );
}

/**
 * Collision-group header: the shared number is the thing the partner must
 * change, so it leads in mono/tabular digits and the group is counted above
 * its members ("dùng chung" counts in-scope + hidden — every holder matters).
 */
function DuplicateGroupHeader({ group }: { group: DuplicateMobileGroup }) {
  const holders = group.employees.length + group.hidden_count;
  return (
    <div className="flex flex-wrap items-center gap-1.5 bg-amber-50/80 px-3 py-2 sm:px-4">
      <Phone className="h-3.5 w-3.5 shrink-0 text-amber-700" aria-hidden="true" />
      <span className="font-mono text-sm font-semibold tabular-nums text-amber-900">
        {group.mobile}
      </span>
      <span className="inline-flex items-center gap-1 rounded-md bg-amber-100 px-1.5 py-0.5 text-xs font-semibold text-amber-800">
        {holders} nhân viên dùng chung
      </span>
      {group.hidden_count > 0 && (
        <span className="inline-flex items-center gap-1 rounded-md bg-rose-50 px-1.5 py-0.5 text-xs font-semibold text-rose-700">
          {group.hidden_count} nhân viên ngoài phạm vi
        </span>
      )}
    </div>
  );
}

export const MissingMobileSection = ({
  onEmployeeClick,
}: MissingMobileSectionProps) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const isMobile = useIsMobile();
  const {
    employees: missingEmployees,
    isLoading: missingLoading,
  } = useMissingMobile();
  const {
    groups,
    totalEmployees: duplicateTotal,
    isLoading: duplicateLoading,
  } = useDuplicateMobiles();

  const toggle = useCallback(() => setIsExpanded(v => !v), []);

  // "No dead card" rule (same as PartnerApprovalRequestsCard): while loading
  // or when both lists are clean the section renders nothing at all — a
  // partner with a healthy phone book never sees an empty warning shell.
  if (missingLoading || duplicateLoading) return null;
  if (missingEmployees.length === 0 && groups.length === 0) return null;

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
            Cảnh báo số điện thoại
          </p>
          {/* Why it matters, stated once: without a unique mobile an employee
              cannot log in by phone number nor receive the Zalo OTP. */}
          <p className="mt-1 text-pretty text-xs leading-4 text-muted-foreground">
            Thiếu hoặc trùng số điện thoại khiến nhân viên không thể đăng nhập
            bằng SĐT và không nhận được OTP Zalo.
          </p>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            {missingEmployees.length > 0 && (
              <span className="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-xs font-semibold text-amber-700">
                Thiếu số điện thoại
                <span className="tabular-nums">{missingEmployees.length}</span>
              </span>
            )}
            {groups.length > 0 && (
              /* Conflicts outrank gaps: a duplicate number breaks accounts
                  that already work, so its chip carries the rose tone. */
              <span className="inline-flex items-center gap-1 rounded-md bg-rose-50 px-1.5 py-0.5 text-xs font-semibold text-rose-700">
                Số điện thoại trùng
                <span className="tabular-nums">{duplicateTotal}</span>
              </span>
            )}
          </div>
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={toggle}
          aria-expanded={isExpanded}
          aria-controls="missing-mobile-list"
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

      {/* Expanded list. Phones get card lists — a 760px-wide table cannot be
          read at 390px, it just scrolls sideways with cut columns. */}
      {isExpanded && (
        <div id="missing-mobile-list" className="border-t border-amber-200">
          {missingEmployees.length > 0 && (
            <div>
              <h3 className="px-3 pt-2.5 text-xs font-semibold uppercase tracking-wide text-amber-900 sm:px-4">
                Thiếu số điện thoại
              </h3>
              {/* Phones get a card list; the table is a desktop-density view.
                  Chosen in JS rather than by CSS so only one variant exists in
                  the DOM. */}
              {isMobile ? (
                <ul
                  className="divide-y divide-amber-100"
                  aria-label="Danh sách nhân viên thiếu số điện thoại"
                >
                  {missingEmployees.map((employee, index) => {
                    const content = (
                      <>
                        <span className="min-w-0 break-words text-sm font-semibold text-foreground">
                          {index + 1}. {employee.fullname}
                        </span>
                        <span className="font-mono text-xs tabular-nums text-muted-foreground">
                          {employee.cccd}
                        </span>
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
                <div
                  className="max-h-80 overflow-auto overscroll-contain focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                  role="region"
                  aria-label="Danh sách nhân viên thiếu số điện thoại"
                  tabIndex={0}
                >
                  <table className="w-full min-w-[560px] text-sm">
                    <caption className="sr-only">
                      Danh sách nhân viên thiếu số điện thoại
                    </caption>
                    <thead className="sticky top-0 z-10 bg-amber-100/80 backdrop-blur-sm">
                      <tr>
                        <th className="w-12 px-4 py-2.5 text-left text-xs font-semibold text-amber-900">STT</th>
                        <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Họ và tên</th>
                        <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">CCCD</th>
                        <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Dự án hiện tại</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-amber-100 bg-card">
                      {missingEmployees.map((employee, index) => (
                        <tr
                          key={employee.id}
                          {...makeRowProps(employee, onEmployeeClick)}
                          className={onEmployeeClick ? CLICKABLE_ROW_CLASS : ''}
                        >
                          <td className="px-4 py-3 text-xs tabular-nums text-muted-foreground">{index + 1}</td>
                          <td className="px-4 py-3">
                            <span className="text-sm font-medium text-foreground">{employee.fullname}</span>
                          </td>
                          <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{employee.cccd}</td>
                          <td className="px-4 py-3">
                            <ProjectCell projects={employee.current_projects} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {groups.length > 0 && (
            <div>
              <h3 className="px-3 pt-2.5 text-xs font-semibold uppercase tracking-wide text-rose-900 sm:px-4">
                Số điện thoại trùng
              </h3>
              <div role="region" aria-label="Danh sách số điện thoại trùng">
                {/* One block per collision group: the shared number leads, its
                    members sit beneath — the partner must see WHICH number to
                    change before they see who carries it. */}
                {groups.map(group => (
                  <div key={group.mobile} className="border-t border-amber-100 first:border-t-0">
                    <DuplicateGroupHeader group={group} />
                    {isMobile ? (
                      <ul className="divide-y divide-amber-100">
                        {group.employees.map((member, index) => {
                            // Duplicate rows carry only id + fullname; click
                            // handlers key off id, so empty out the required
                            // Employee fields.
                            const employee: Employee = {
                              id: member.id,
                              fullname: member.fullname,
                              cccd: '',
                              current_projects: [],
                            };
                          const content = (
                            <>
                              <span className="min-w-0 break-words text-sm font-semibold text-foreground">
                                {index + 1}. {member.fullname}
                              </span>
                              <span className="font-mono text-xs tabular-nums text-muted-foreground">
                                {group.mobile}
                              </span>
                              <CoHolders group={group} member={member} />
                            </>
                          );

                          return (
                            <li key={member.id}>
                              {onEmployeeClick ? (
                                <button
                                  type="button"
                                  onClick={() => onEmployeeClick(employee)}
                                  aria-label={`Mở thông tin nhân viên ${member.fullname}`}
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
                      <table className="w-full min-w-[560px] text-sm">
                        <caption className="sr-only">
                          Nhân viên dùng chung số {group.mobile}
                        </caption>
                        <thead className="bg-amber-100/60">
                          <tr>
                            <th className="w-12 px-4 py-2.5 text-left text-xs font-semibold text-amber-900">STT</th>
                            <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Họ và tên</th>
                            <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Số điện thoại</th>
                            <th className="px-4 py-2.5 text-left text-xs font-semibold text-amber-900">Dùng chung với</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-amber-100 bg-card">
                          {group.employees.map((member, index) => {
                            const employee: Employee = {
                              id: member.id,
                              fullname: member.fullname,
                              cccd: '',
                              current_projects: [],
                            };
                            return (
                              <tr
                                key={member.id}
                                {...makeRowProps(employee, onEmployeeClick)}
                                className={onEmployeeClick ? CLICKABLE_ROW_CLASS : ''}
                              >
                                <td className="px-4 py-3 text-xs tabular-nums text-muted-foreground">{index + 1}</td>
                                <td className="px-4 py-3">
                                  <span className="text-sm font-medium text-foreground">{member.fullname}</span>
                                </td>
                                <td className="px-4 py-3 font-mono text-xs tabular-nums text-muted-foreground">
                                  {group.mobile}
                                </td>
                                <td className="px-4 py-3">
                                  <CoHolders group={group} member={member} />
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
