import { format } from "date-fns";
import { useState, useMemo, useCallback, useEffect } from "react";
import { ColumnDef, SortingState } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Unlink, User, Calendar } from "lucide-react";
import { Project } from "@/types/api/project.types";
import { ProjectEmployeeAssignment } from "@/types/api/project-employee.types";
import { DataTable } from "@/components/ui/data-table";
import { useProjectEmployeeManagement } from "@/hooks/business/useProjectEmployeeManagement";
import { PaymentScheduleToggle } from "./PaymentScheduleToggle";
import { CheckInToggle } from "./CheckInToggle";

interface AssignmentMeta {
  canRemove: boolean;
  canCancel: boolean;
}

interface ProjectEmployeesListProps {
  project: Project;
  onEmployeeRemoved?: () => void;
  onAddEmployees?: () => void;
  onTotalCountChange?: (count: number) => void;
}

export function ProjectEmployeesList({
  project,
  onEmployeeRemoved,
  onAddEmployees,
  onTotalCountChange,
}: ProjectEmployeesListProps) {
  // UI state
  const [removingId, setRemovingId] = useState<number | null>(null);
  const [showRemoveDialog, setShowRemoveDialog] = useState(false);
  const [selectedEmployeeToRemove, setSelectedEmployeeToRemove] =
    useState<ProjectEmployeeAssignment | null>(null);
  const [lastDate, setLastDate] = useState<string>("");
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // Business logic
  const {
    employees,
    totalCount: apiTotalCount,
    isLoading,
    isRemoving,
    error,
    errorMessage,
    removeEmployee,
    canAddEmployees,
    isEmpty,
    sorting,
    setSorting,
  } = useProjectEmployeeManagement({
    project,
    onEmployeeRemoved,
  });

  // Notify parent of total count
  useEffect(() => {
    onTotalCountChange?.(apiTotalCount);
  }, [apiTotalCount, onTotalCountChange]);

  // UI event handlers
  const handleRemoveEmployee = useCallback(
    async (assignment?: ProjectEmployeeAssignment) => {
      const employeeToRemove = assignment || selectedEmployeeToRemove;
      if (!employeeToRemove) return;

      setRemovingId(employeeToRemove.id);
      setShowRemoveDialog(false);

      const success = await removeEmployee(
        employeeToRemove,
        lastDate || undefined,
      );

      if (success) {
        setSelectedEmployeeToRemove(null);
        setLastDate("");
      }

      setRemovingId(null);
    },
    [selectedEmployeeToRemove, lastDate, removeEmployee],
  );

  const openRemoveDialog = useCallback(
    (assignment: ProjectEmployeeAssignment) => {
      setSelectedEmployeeToRemove(assignment);
      setShowRemoveDialog(true);
    },
    [],
  );

  const closeRemoveDialog = useCallback(() => {
    setShowRemoveDialog(false);
    setSelectedEmployeeToRemove(null);
    setLastDate("");
  }, []);

  // Handle row click
  const handleRowClick = useCallback(
    (employee: ProjectEmployeeAssignment & { meta: AssignmentMeta }) => {
      const { meta } = employee;

      // Only trigger removal dialog for employees that can be removed
      if (meta.canRemove) {
        openRemoveDialog(employee);
      } else if (meta.canCancel) {
        // For employees that can be cancelled (upcoming assignments), handle immediately
        handleRemoveEmployee(employee);
      }
    },
    [openRemoveDialog, handleRemoveEmployee],
  );

  // Memoized columns definition
  const columns: ColumnDef<
    ProjectEmployeeAssignment & { meta: AssignmentMeta }
  >[] = useMemo(
    () => {
      const cols: ColumnDef<ProjectEmployeeAssignment & { meta: AssignmentMeta }>[] = [
      // STT column removed for mobile card simplicity
      {
        id: "employee_info",
        header: "Thông tin nhân viên",
        enableSorting: true,
        cell: ({ row }) => (
          <div className="space-y-1">
            <div className="font-medium">
              {row.original.employee_name || row.original.employee_code || "-"}
            </div>
            <div className="font-mono typography-body-small text-muted-foreground">
              CCCD: {row.original.employee_cccd || "-"}
            </div>
          </div>
        ),
      },
      {
        id: "employee_code",
        header: "Mã NV",
        accessorKey: "employee_code",
        cell: ({ row }) => (
          <div className="font-mono typography-body-medium">
            {row.original.employee_code || "-"}
          </div>
        ),
      },
      {
        id: "dates",
        header: "Thời gian làm việc",
        enableSorting: true,
        cell: ({ row }) => (
          <div className="space-y-1">
            <div className="typography-body-medium">
              <span className="text-muted-foreground">BĐ:</span>{" "}
              {row.original.start_date
                ? format(new Date(row.original.start_date), 'dd/MM/yyyy')
                : "-"}
            </div>
            <div className="typography-body-medium">
              <span className="text-muted-foreground">KT:</span>{" "}
              {row.original.last_date
                ? format(new Date(row.original.last_date), 'dd/MM/yyyy')
                : "-"}
            </div>
          </div>
        ),
      },
      {
        id: "position",
        header: "Vị trí",
        accessorKey: "position",
        cell: ({ row }) => (
          <div className="typography-body-medium">
            {row.original.position || "-"}
          </div>
        ),
      },
      {
        id: "payment_schedule",
        header: "Chu kỳ thanh toán",
        enableSorting: true,
        cell: ({ row }) => (
          <PaymentScheduleToggle
            assignment={row.original}
            disabled={project.status !== "active"}
          />
        ),
      },
    ];

    if (project.is_flexible) {
      cols.push({
        id: "check_in_enabled",
        header: "Điểm danh",
        cell: ({ row }) => (
          <CheckInToggle
            assignment={row.original}
            disabled={project.status !== "active"}
          />
        ),
      });
    }

    return cols;
  }, [project.status, project.is_flexible]);

  // Pagination logic
  const totalRecords = apiTotalCount;
  const totalPages = Math.ceil(totalRecords / pageSize);
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedEmployees = employees.slice(startIndex, endIndex);

  // Pagination handlers
  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
  }, []);

  const handlePageSizeChange = useCallback((newPageSize: number) => {
    setPageSize(newPageSize);
    setCurrentPage(1);
  }, []);

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="typography-body-medium text-muted-foreground">
          Đang tải danh sách nhân viên...
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="typography-body-medium text-destructive">
          Lỗi khi tải danh sách nhân viên: {errorMessage}
        </div>
      </div>
    );
  }

  // Empty state
  if (isEmpty) {
    return (
      <div className="h-full flex flex-col">
        {/* Empty State */}
        <div className="flex-1 flex flex-col items-center justify-center space-y-4">
          <User className="w-12 h-12 text-muted-foreground" />
          <div className="text-center space-y-2">
            <h3 className="font-medium">Chưa có nhân viên nào được giao</h3>
            <p className="typography-body-medium text-muted-foreground">
              {project.status === "active"
                ? 'Nhấn nút "Thêm nhân viên" ở cuối trang để giao nhân viên vào dự án này'
                : "Chỉ có thể thêm nhân viên vào dự án đang hoạt động"}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* DataTable */}
      <div className="flex-1 overflow-auto p-4 pt-2">
        <DataTable
          columns={columns}
          data={paginatedEmployees}
          sorting={sorting}
          onSortingChange={setSorting}
          pagination={{
            page: currentPage,
            pageSize: pageSize,
            totalPages: totalPages,
            totalRecords: totalRecords,
          }}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
          onRowClick={handleRowClick}
          getRowClassName={(employee) => {
            const { meta } = employee;
            // Add cursor pointer for removable/cancellable employees
            return meta.canRemove || meta.canCancel
              ? "cursor-pointer hover:bg-muted/50"
              : "";
          }}
          primaryColumns={[
            "employee_info",
            "employee_code",
            "dates",
            "position",
            "payment_schedule",
          ]}
          caption="Danh sách nhân viên dự án"
        />
      </div>

      {/* Remove Employee Dialog with Date Selection */}
      <Dialog open={showRemoveDialog} onOpenChange={setShowRemoveDialog}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Unlink className="h-5 w-5 text-destructive" />
              Xác nhận gỡ nhân viên
            </DialogTitle>
            <DialogDescription>
              Gỡ{" "}
              <strong>
                {selectedEmployeeToRemove?.employee_name ||
                  selectedEmployeeToRemove?.employee_code ||
                  "nhân viên này"}
              </strong>{" "}
              khỏi dự án <strong>{project.name}</strong>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label
                htmlFor="last-date"
                className="text-sm font-medium flex items-center gap-2"
              >
                <Calendar className="h-4 w-4" />
                Ngày kết thúc
              </Label>
              <Input
                id="last-date"
                type="date"
                value={lastDate}
                onChange={(e) => setLastDate(e.target.value)}
                className="w-full"
              />
              <p className="text-xs text-muted-foreground">
                Để trống để kết thúc ngay lập tức
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={closeRemoveDialog}
              disabled={removingId === selectedEmployeeToRemove?.id}
            >
              Đóng
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => handleRemoveEmployee()}
              disabled={removingId === selectedEmployeeToRemove?.id}
            >
              {removingId === selectedEmployeeToRemove?.id
                ? "Đang gỡ..."
                : "Gỡ khỏi dự án"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
