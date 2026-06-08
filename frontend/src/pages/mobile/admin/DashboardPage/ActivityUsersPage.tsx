import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ArrowLeft, User, Activity } from "lucide-react";
import { format, parseISO } from "date-fns";
import { vi } from "date-fns/locale";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { UserAvatar } from "@/components/ui/user-avatar";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { EmptyState } from "@/components/shared/EmptyState";
import { EmployeeDetailsSheet } from "@/components/employees/details/EmployeeDetailsSheet";
import { useActiveEmployeesBySchedule } from "@/hooks/api/useDashboard";
import { useEmployee, useUpdateEmployee, useDeleteEmployee } from "@/hooks/api/useEmployees";
import type { Employee } from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";

type Schedule = "weekly" | "monthly" | "flexible";

const SCHEDULE_LABELS: Record<Schedule, string> = {
  weekly: "Lương tuần",
  monthly: "Lương tháng",
  flexible: "Linh hoạt",
};

const ROLE_LABELS: Record<string, string> = {
  admin: "Quản trị",
  partner: "Quản lý",
  employee: "Nhân viên",
};

const SCHEDULE_BADGE_LABELS: Record<string, string> = {
  weekly: "Lương tuần",
  monthly: "Lương tháng",
  flexible: "Linh hoạt",
};

const ActivityUsersPage = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [selectedEmployeeId, setSelectedEmployeeId] = useState<number | null>(null);

  const schedule = searchParams.get("schedule") as Schedule | null;
  const month = searchParams.get("month") ?? undefined;

  const { data: users, isLoading } = useActiveEmployeesBySchedule(schedule, month);
  const { data: selectedEmployee } = useEmployee(
    selectedEmployeeId ?? 0,
    !!selectedEmployeeId,
  );
  const updateEmployeeMutation = useUpdateEmployee();
  const deleteEmployeeMutation = useDeleteEmployee();

  const handleUpdate = async (
    employeeId: number,
    employeeData: Partial<Employee>,
    bankObject?: Bank | null,
  ) => {
    updateEmployeeMutation.mutate({
      id: employeeId,
      data: employeeData as Partial<Employee>,
      bank: bankObject,
    });
  };

  const handleDelete = (employee: Employee) => {
    deleteEmployeeMutation.mutate(employee.id, {
      onSuccess: () => setSelectedEmployeeId(null),
    });
  };

  const title = schedule ? SCHEDULE_LABELS[schedule] : "Nhân viên đăng nhập";
  const subtitle = month
    ? `Tháng ${month.split("-").reverse().join("/")}`
    : "Tất cả thời gian";

  return (
    <div className="flex flex-col min-h-full pb-20">
      <MobilePageHeader
        icon={Activity}
        title={title}
        subtitle={subtitle}
        sticky={false}
        bordered={false}
        actions={
          <Button variant="ghost" size="icon" className="h-9 w-9" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
        }
      />

      {/* Content */}
      <div className="flex-1 px-4 py-3">
        {isLoading ? (
          <div className="grid grid-cols-2 gap-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="rounded-xl border p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <Skeleton className="h-9 w-9 rounded-full shrink-0" />
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
          <EmptyState
            icon={User}
            title="Không có nhân viên nào đăng nhập"
          />
        ) : (
          <div className="grid grid-cols-2 gap-3">
            {users.map((user) => (
              <button
                key={user.user_id}
                className="rounded-xl border bg-card p-3 text-left hover:bg-muted/50 active:bg-muted/70 transition-colors flex flex-col gap-2 touch-manipulation"
                onClick={() => setSelectedEmployeeId(user.employee_id)}
              >
                <div className="flex items-center gap-2 min-w-0">
                  <UserAvatar
                    name={user.fullname}
                    username={user.username}
                    size="md"
                    className="shrink-0"
                  />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate leading-tight">
                      {user.fullname}
                    </p>
                    <p className="text-xs text-muted-foreground truncate">
                      @{user.username}
                    </p>
                  </div>
                </div>
                <div className="flex items-center justify-between gap-1 text-xs text-muted-foreground">
                  <span>{ROLE_LABELS[user.role] ?? user.role}</span>
                  <span>
                    {SCHEDULE_BADGE_LABELS[user.payment_schedule] ??
                      user.payment_schedule}
                  </span>
                </div>
                {user.last_login && (
                  <p className="text-xs text-muted-foreground/70 leading-none">
                    {format(parseISO(user.last_login), "dd/MM/yyyy HH:mm", {
                      locale: vi,
                    })}
                  </p>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      <EmployeeDetailsSheet
        employee={selectedEmployee}
        isOpen={!!selectedEmployeeId}
        onClose={() => setSelectedEmployeeId(null)}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
      />
    </div>
  );
};

export default ActivityUsersPage;
