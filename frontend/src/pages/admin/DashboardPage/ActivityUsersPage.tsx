import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { format, parseISO } from "date-fns";
import { vi } from "date-fns/locale";
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
} from "@/components/shared/AdminPageFrame";
import { PageHeader } from "@/components/shared/PageHeader";
import { EmptyState } from "@/components/shared/EmptyState";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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

/** Desktop variant of the dashboard "active users" drill-down (PAYR-23):
 *  dense table for >=768px, paired with the mobile card page via ResponsivePage. */
const ActivityUsersPage = () => {
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
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader title={title} description={subtitle} />
      </AdminPageHeaderCard>

      <AdminSectionCard aria-label="Danh sách nhân viên đăng nhập">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-11 w-full" />
            ))}
          </div>
        ) : !users || users.length === 0 ? (
          <EmptyState title="Không có nhân viên nào đăng nhập" description="Thử chọn kỳ khác." size="sm" />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="whitespace-nowrap">Nhân viên</TableHead>
                <TableHead className="whitespace-nowrap">Vai trò</TableHead>
                <TableHead className="whitespace-nowrap">Lịch lương</TableHead>
                <TableHead className="whitespace-nowrap">Đăng nhập gần nhất</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user) => (
                <TableRow
                  key={user.user_id}
                  className="cursor-pointer"
                  onClick={() => setSelectedEmployeeId(user.employee_id)}
                >
                  <TableCell className="align-top">
                    <div className="flex items-center gap-2">
                      <UserAvatar name={user.fullname} username={user.username} size="sm" />
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-fg-primary">{user.fullname}</p>
                        <p className="truncate text-xs text-fg-tertiary">@{user.username}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="align-top text-sm text-fg-secondary">
                    {ROLE_LABELS[user.role] ?? user.role}
                  </TableCell>
                  <TableCell className="align-top">
                    <Badge variant="outline" className="text-xs font-medium">
                      {SCHEDULE_LABELS[user.payment_schedule as Schedule] ?? user.payment_schedule}
                    </Badge>
                  </TableCell>
                  <TableCell className="align-top text-sm tabular-nums text-fg-secondary">
                    {user.last_login
                      ? format(parseISO(user.last_login), "dd/MM/yyyy HH:mm", { locale: vi })
                      : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </AdminSectionCard>

      <EmployeeDetailsSheet
        employee={selectedEmployee}
        isOpen={!!selectedEmployeeId}
        onClose={() => setSelectedEmployeeId(null)}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
      />
    </AdminPageCanvas>
  );
};

export default ActivityUsersPage;
