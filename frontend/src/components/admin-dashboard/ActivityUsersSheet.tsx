import { useState } from 'react';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { X } from 'lucide-react';
import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';
import { useActiveEmployeesBySchedule } from '@/hooks/api/useDashboard';
import { EmployeeDetailsSheet } from '@/components/employees/details/EmployeeDetailsSheet';
import { useEmployee, useUpdateEmployee, useDeleteEmployee } from '@/hooks/api/useEmployees';
import { UserAvatar } from '@/components/ui/user-avatar';
import { EmptyState } from '@/components/shared/EmptyState';
import type { Employee } from '@/types/api/employee.types';
import type { Bank } from '@/types/api/bank.types';

type Schedule = 'weekly' | 'monthly' | 'flexible';

const SCHEDULE_LABELS: Record<Schedule, string> = {
  weekly: 'Lương tuần',
  monthly: 'Lương tháng',
  flexible: 'Linh hoạt',
};

const ROLE_LABELS: Record<string, string> = {
  admin: 'Quản trị',
  partner: 'Quản lý',
  employee: 'Nhân viên',
};

const SCHEDULE_BADGE_LABELS: Record<string, string> = {
  weekly: 'Lương tuần',
  monthly: 'Lương tháng',
  flexible: 'Linh hoạt',
};

interface ActivityUsersSheetProps {
  schedule: Schedule | null;
  month?: string;
  onClose: () => void;
}

export function ActivityUsersSheet({ schedule, month, onClose }: ActivityUsersSheetProps) {
  const [selectedEmployeeId, setSelectedEmployeeId] = useState<number | null>(null);

  const { data: users, isLoading } = useActiveEmployeesBySchedule(schedule, month);
  const isMobile = useIsMobile();
  const { data: selectedEmployee } = useEmployee(selectedEmployeeId ?? 0, !!selectedEmployeeId);
  const updateEmployeeMutation = useUpdateEmployee();
  const deleteEmployeeMutation = useDeleteEmployee();

  const handleUpdate = async (employeeId: number, employeeData: Partial<Employee>, bankObject?: Bank | null) => {
    updateEmployeeMutation.mutate({ id: employeeId, data: employeeData as Partial<Employee>, bank: bankObject });
  };

  const handleDelete = (employee: Employee) => {
    deleteEmployeeMutation.mutate(employee.id, {
      onSuccess: () => setSelectedEmployeeId(null),
    });
  };

  return (
    <>
      <Sheet open={!!schedule} onOpenChange={(open) => { if (!open) onClose(); }}>
        <SheetContent
          side={isMobile ? "bottom" : "right"}
          className={cn(
            "admin-dashboard-activity-sheet flex w-full flex-col p-0 shadow-none sm:w-[480px]",
            isMobile && "max-h-[94dvh] rounded-t-2xl",
          )}
        >
          {/* Mobile drag handle */}
          {isMobile && (
            <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
              <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
            </div>
          )}
          <SheetHeader className="px-4 py-3 border-b flex-shrink-0">
            <div className="flex items-start justify-between gap-3">
              <SheetTitle className="text-sm font-semibold break-words">
                {schedule ? SCHEDULE_LABELS[schedule] : ''}{month ? ` — ${month}` : ''}
              </SheetTitle>
              <SheetClose asChild>
                <Button variant="ghost" size="icon" className="h-11 w-11 shrink-0 rounded-full text-muted-foreground">
                  <X className="h-4 w-4" />
                </Button>
              </SheetClose>
            </div>
          </SheetHeader>

          <div className="flex-1 overflow-y-auto p-4">
            {isLoading ? (
              <div className="grid grid-cols-1 gap-3 min-[420px]:grid-cols-2">
                {Array.from({ length: 6 }).map((_, i) => (
                  <div key={i} className="rounded-xl border p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-9 w-9 rounded-full flex-shrink-0" />
                      <div className="space-y-1.5 flex-1 min-w-0">
                        <Skeleton className="h-3 w-24" />
                        <Skeleton className="h-2.5 w-16" />
                      </div>
                    </div>
                    <Skeleton className="h-2.5 w-20" />
                  </div>
                ))}
              </div>
            ) : !users || users.length === 0 ? (
              <EmptyState title="Không có nhân viên nào đăng nhập" size="sm" className="py-4" />
            ) : (
              <div className="grid grid-cols-1 gap-3 min-[420px]:grid-cols-2">
                {users.map((user) => (
                  <button
                    key={user.user_id}
                    className="flex min-h-24 flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-colors hover:bg-muted/50 motion-reduce:transition-none"
                    onClick={() => setSelectedEmployeeId(user.employee_id)}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <UserAvatar
                        name={user.fullname}
                        username={user.username}
                        size="md"
                        className="flex-shrink-0"
                      />
                      <div className="min-w-0">
                        <p className="text-sm font-medium leading-tight break-words">{user.fullname}</p>
                        <p className="text-xs text-muted-foreground break-all">@{user.username}</p>
                      </div>
                    </div>
                    <div className="flex flex-wrap items-center justify-between gap-1 text-xs text-muted-foreground">
                      <span className="break-words">{ROLE_LABELS[user.role] ?? user.role}</span>
                      <span className="break-words text-right">{SCHEDULE_BADGE_LABELS[user.payment_schedule] ?? user.payment_schedule}</span>
                    </div>
                    {user.last_login && (
                      <p className="text-xs text-muted-foreground/70 leading-none">
                        {format(parseISO(user.last_login), 'dd/MM/yyyy HH:mm', { locale: vi })}
                      </p>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <EmployeeDetailsSheet
        employee={selectedEmployee}
        isOpen={!!selectedEmployeeId}
        onClose={() => setSelectedEmployeeId(null)}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
      />
    </>
  );
}
