import { useState, useEffect, useMemo } from "react";
import { ColumnDef, SortingState } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/ui/data-table";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from "@/components/ui/sheet";
import { Label } from "@/components/ui/label";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Users, UserPlus, Building2, Link, Plus, CheckCircle2, X, Calendar, Clock } from "lucide-react";
import { Project } from "@/types/api/project.types";
import { useEmployees } from "@/hooks/api/useEmployees";
import {
  useAssignEmployeeToProject,
} from "@/hooks/api/useProjectEmployees";
import { toast } from "@/components/ui/sonner";
import { Employee } from "@/types/api/employee.types";
import { projectEmployeeService } from "@/services/api/project-employee.service";
import { useCurrentPayRate } from "@/hooks/api/usePayRates";
import { extractPositionsFromPayrates, DEFAULT_POSITION } from "@/types/api/payrate.types";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { cn } from "@/lib/utils";

interface AddEmployeesToProjectProps {
  project: Project;
  onEmployeesAdded?: () => void;
  isOpen?: boolean;
  onClose?: () => void;
}

export function AddEmployeesToProject({
  project,
  onEmployeesAdded,
  isOpen,
  onClose
}: AddEmployeesToProjectProps) {
  const isMobile = useIsMobile();
  const [assigningId, setAssigningId] = useState<number | null>(null);
  const [assignedEmployeeIds, setAssignedEmployeeIds] = useState<Set<number>>(new Set());
  const [isLoadingAssignments, setIsLoadingAssignments] = useState(true);
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const [selectedPosition, setSelectedPosition] = useState<string>(DEFAULT_POSITION);
  const [startDate, setStartDate] = useState<string>(() => {
    // Default to today's date
    return new Date().toISOString().split('T')[0];
  });
  const [paymentSchedule, setPaymentSchedule] = useState<'weekly' | 'monthly'>('weekly');
  const [recentlyAddedIds, setRecentlyAddedIds] = useState<Set<number>>(new Set());
  const [employeePositions, setEmployeePositions] = useState<Map<number, string>>(new Map());
  const [sorting, setSorting] = useState<SortingState>([]);

  const { sortBy, sortOrder } = useMemo(() => {
    if (sorting.length === 0) return { sortBy: undefined, sortOrder: undefined as 'desc' | 'asc' | undefined };
    const col = sorting[0];
    return {
      sortBy: col.id,
      sortOrder: col.desc ? ('desc' as const) : ('asc' as const),
    };
  }, [sorting]);

  const {
    data: employeesData,
    isLoading: isLoadingEmployees
  } = useEmployees({
    pageSize: 50,
    sortBy,
    sortOrder,
  });

  const { mutateAsync: assignSingle, isPending: isAssigning } = useAssignEmployeeToProject();

  // Fetch current payrate for the project to get available positions
  const { data: payrateData } = useCurrentPayRate(project.id);

  // Modal navigation
  const { openProjectDetails } = useProjectModals();

  // Extract available positions from payrate data
  const availablePositions = useMemo(() => {
    if (payrateData?.rates) {
      return extractPositionsFromPayrates(payrateData);
    }
    return [];
  }, [payrateData]);

  // Check if payrates exist for the selected project
  const hasPayrates = availablePositions.length > 0;

  // Fetch employees already assigned to this specific project
  useEffect(() => {
    const fetchAssignedEmployees = async () => {
      setIsLoadingAssignments(true);
      try {
        // Get active assignments for this specific project only
        const response = await projectEmployeeService.getProjectEmployees(project.id, {
          status: 'active',
          pageSize: 1000 // Get a large number to capture most assignments
        });

        // Extract employee IDs that are actively assigned to this specific project
        const projectEmployeeIds = new Set<number>();
        response.data.forEach(assignment => {
          projectEmployeeIds.add(assignment.employee_id);
        });

        setAssignedEmployeeIds(projectEmployeeIds);
      } catch (error) {
        console.error('Failed to fetch project assignments:', error);
        toast({
          title: "Lỗi",
          description: "Không thể tải danh sách phân công của dự án.",
          variant: "destructive"
        });
      } finally {
        setIsLoadingAssignments(false);
      }
    };

    fetchAssignedEmployees();
  }, [project.id]);

  // Reset position when payrates are loaded
  useEffect(() => {
    if (hasPayrates && availablePositions.length > 0) {
      setSelectedPosition(availablePositions[0]);
    } else {
      setSelectedPosition(DEFAULT_POSITION);
    }
  }, [hasPayrates, availablePositions]);

  // Filter available employees (not already assigned to this specific project)
  const availableEmployees = useMemo(() => {
    if (!employeesData?.data) return [];

    return employeesData.data.filter((employee: Employee) => {
      // Only show employees who are not currently assigned to this specific project
      return !assignedEmployeeIds.has(employee.id);
    });
  }, [employeesData, assignedEmployeeIds]);

  const handleEmployeeClick = (employee: Employee) => {
    setSelectedEmployee(employee);
    setShowConfirmDialog(true);
  };

  const handleRowClick = (employee: Employee) => {
    // Don't process if employee is already added or currently being assigned
    if (recentlyAddedIds.has(employee.id) || assigningId === employee.id) {
      return;
    }

    // Show confirmation dialog to allow position and date selection
    handleEmployeeClick(employee);
  };

  const handleDirectAssign = async (employee: Employee, position: string) => {
    if (!position?.trim()) return;

    try {
      setAssigningId(employee.id);

      const today = new Date();
      const startDate = today.toISOString().split('T')[0];

      await assignSingle({
        projectId: project.id,
        data: {
          employee_id: employee.id,
          start_date: startDate,
          position: position,
        }
      });

      // Update state to show the employee as added
      setRecentlyAddedIds(prev => new Set(prev).add(employee.id));
      setEmployeePositions(prev => new Map(prev).set(employee.id, position));

      toast({
        title: "Thêm nhân viên thành công",
        description: `Đã thêm ${employee.fullname} vào dự án ${project.name}`,
      });

      // Notify parent
      onEmployeesAdded?.();
    } catch (error: unknown) {
      console.error('Assignment error:', error);
    } finally {
      setAssigningId(null);
    }
  };

  const handleConfirmAssign = async () => {
    if (!selectedEmployee || !selectedPosition?.trim()) return;

    try {
      setAssigningId(selectedEmployee.id);
      setShowConfirmDialog(false);

      await assignSingle({
        projectId: project.id,
        data: {
          employee_id: selectedEmployee.id,
          start_date: startDate,
          position: selectedPosition,
          payment_schedule: paymentSchedule,
        }
      });

      // Update state to show the employee as added
      setRecentlyAddedIds(prev => new Set(prev).add(selectedEmployee.id));
      setEmployeePositions(prev => new Map(prev).set(selectedEmployee.id, selectedPosition));

      toast({
        title: "Thêm nhân viên thành công",
        description: `Đã thêm ${selectedEmployee.fullname} vào dự án ${project.name}`,
      });

      // Notify parent
      onEmployeesAdded?.();
    } catch (error: unknown) {
      console.error('Assignment error:', error);
    } finally {
      setAssigningId(null);
      setSelectedEmployee(null);
    }
  };

  const handleCancelAssign = () => {
    setShowConfirmDialog(false);
    setSelectedEmployee(null);
  };

  const getRowClassName = (employee: Employee) => {
    return recentlyAddedIds.has(employee.id) ? "bg-green-50/50" : "";
  };

  // Column definitions for DataTable
  const columns: ColumnDef<Employee>[] = [
    {
      id: "fullname",
      header: "Tên nhân viên",
      accessorKey: "fullname",
      size: 200,
      cell: ({ row }) => (
        <div className="flex min-h-11 items-center">
          <span className="font-medium break-words">
            {row.original.fullname}
          </span>
        </div>
      ),
    },
    {
      id: "cccd",
      header: "CCCD",
      accessorKey: "cccd",
      size: 140,
      cell: ({ row }) => (
        <div className="flex min-h-11 items-center">
          <span className="break-all font-mono typography-body-medium">
            {row.original.cccd}
          </span>
        </div>
      ),
    },
    {
      id: "position",
      header: "Vị trí",
      enableSorting: true,
      size: 120,
      cell: ({ row }) => {
        const employeeId = row.original.id;
        const position = employeePositions.get(employeeId);
        return (
          <div className="flex min-h-11 items-center">
            <span className="typography-body-medium">
              {position || '-'}
            </span>
          </div>
        );
      },
    },
    {
      id: "actions",
      header: "Trạng thái",
      enableSorting: true,
      size: 80,
      cell: ({ row }) => {
        const employeeId = row.original.id;
        const isAdded = recentlyAddedIds.has(employeeId);

        return (
          <div className="flex min-h-11 items-center justify-center">
            {isAdded ? (
              <CheckCircle2 className="w-5 h-5 text-green-600" />
            ) : assigningId === employeeId ? (
              <div className="w-4 h-4 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
            ) : (
              <Button
                variant="ghost"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation();
                  handleEmployeeClick(row.original);
                }}
                disabled={isAssigning}
                className="h-11 w-11 p-0 text-primary hover:text-primary"
              >
                <Link className="w-4 h-4" />
              </Button>
            )}
          </div>
        );
      },
    },
  ];

  if (isLoadingEmployees || isLoadingAssignments) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="typography-body-medium text-muted-foreground">Đang tải danh sách nhân viên...</div>
      </div>
    );
  }

  const content = (
    <div className="h-full flex flex-col">
      {/* Employee List */}
      <div className="flex-1 overflow-hidden p-6">
        {availableEmployees.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 space-y-4 p-6">
            <div className="p-4 rounded-full bg-muted/50">
              <Users className="w-8 h-8 text-muted-foreground" />
            </div>
            <div className="text-center space-y-2">
              <h3 className="typography-title-large">Không có nhân viên khả dụng</h3>
              <p className="typography-body-medium text-muted-foreground max-w-sm">
                Tất cả nhân viên đã được giao vào dự án này.
              </p>
            </div>
          </div>
        ) : (
          <div className="h-full">
            <DataTable
              columns={columns}
              data={availableEmployees}
              sorting={sorting}
              onSortingChange={setSorting}
              showPagination={availableEmployees.length > 15}
              primaryColumns={["fullname", "cccd"]}
              caption="Danh sách nhân viên khả dụng để thêm vào dự án"
              className="h-full"
              onRowClick={handleRowClick}
              getRowClassName={getRowClassName}
            />
          </div>
        )}
      </div>

      {/* Footer with Close Button */}
      {(isOpen !== undefined && onClose) && (
        <div className="flex-shrink-0 w-full border-t">
          <div className="p-4 w-full" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
            <Button
              variant="outline"
              onClick={onClose}
              className="min-h-11 w-full bg-gray-800 border-border text-white hover:bg-gray-700 hover:border-border"
            >
              Đóng
            </Button>
          </div>
        </div>
      )}

      {/* Confirmation Dialog with Position Selection */}
      <Dialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <DialogContent className="max-h-[92dvh] overflow-y-auto sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Users className="h-5 w-5" />
              Xác nhận thêm nhân viên
            </DialogTitle>
            <DialogDescription>
              Thêm <strong>{selectedEmployee?.fullname}</strong> vào dự án <strong>{project?.name}</strong>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="position" className="text-sm font-medium">
                Vị trí *
              </Label>
              <div className="space-y-3">
                <div className="flex flex-col gap-2 min-[380px]:flex-row">
                  <Select
                    value={hasPayrates ? selectedPosition : ""}
                    onValueChange={setSelectedPosition}
                    disabled={!hasPayrates}
                  >
                    <SelectTrigger className="h-11 w-full">
                      <SelectValue
                        placeholder={hasPayrates ? "Chọn vị trí" : "Dự án chưa có bảng lương"}
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {availablePositions.map((position) => (
                        <SelectItem key={position} value={position}>
                          {position}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {!hasPayrates && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setShowConfirmDialog(false);
                        openProjectDetails(project.id.toString(), 'payrates');
                      }}
                      className="min-h-11 shrink-0"
                    >
                      <Plus className="h-4 w-4 mr-2" />
                      Thêm bảng lương
                    </Button>
                  )}
                </div>
                {!hasPayrates && (
                  <p className="text-xs text-muted-foreground">
                    Dự án này chưa có bảng lương. Vui lòng tạo bảng lương trước khi thêm nhân viên.
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="start-date" className="text-sm font-medium flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                Ngày bắt đầu
              </Label>
              <Input
                id="start-date"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="h-11 w-full"
              />
              <p className="text-xs text-muted-foreground">
                Để trống để có hiệu lực ngay lập tức
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="payment-schedule" className="text-sm font-medium flex items-center gap-2">
                <Clock className="h-4 w-4" />
                Chu kỳ thanh toán
              </Label>
              <Select
                value={paymentSchedule}
                onValueChange={(value) => setPaymentSchedule(value as 'weekly' | 'monthly')}
              >
                <SelectTrigger className="h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="weekly">
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4" />
                      Trả lương hàng tuần
                    </div>
                  </SelectItem>
                  <SelectItem value="monthly">
                    <div className="flex items-center gap-2">
                      <Calendar className="h-4 w-4" />
                      Trả lương hàng tháng
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Chu kỳ thanh toán mặc định là hàng tuần
              </p>
            </div>
          </div>

            <DialogFooter className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Button
              type="button"
              variant="outline"
              onClick={handleCancelAssign}
              disabled={assigningId === selectedEmployee?.id}
              className="min-h-11"
            >
              Đóng
            </Button>
            <Button
              type="button"
              onClick={handleConfirmAssign}
              disabled={assigningId === selectedEmployee?.id || !selectedPosition?.trim()}
              className="min-h-11"
            >
              {assigningId === selectedEmployee?.id ? "Đang thêm..." : "Thêm nhân viên"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );

  // If used as a sheet, wrap in Sheet component
  if (isOpen !== undefined && onClose) {
    return (
      <Sheet open={isOpen} onOpenChange={onClose}>
        <SheetContent
          side="right"
          className={`${isMobile ? "w-full" : "w-[70%] min-w-[600px] max-w-[800px]"} p-0 flex flex-col h-full`}
        >
          <SheetHeader className="space-y-0 p-6 pb-0 flex-shrink-0 border-b" style={{ paddingTop: "max(24px, calc(24px + env(safe-area-inset-top)))" }}>
            <div className="flex items-start gap-3">
              <div className="p-2 rounded-xl bg-primary/10">
                <Building2 className="w-5 h-5 text-primary" />
              </div>
              <div className="min-w-0 flex-1">
                <SheetTitle className="typography-headline-medium leading-tight tracking-tight">
                  Thêm nhân viên vào dự án
                </SheetTitle>
                <p className="typography-body-medium mt-1 break-words font-medium text-muted-foreground">
                  {project?.name}
                </p>
              </div>
              <SheetClose asChild>
                <Button variant="ghost" size="icon" className="h-11 w-11 shrink-0">
                  <X className="h-4 w-4" />
                  <span className="sr-only">Đóng</span>
                </Button>
              </SheetClose>
            </div>
          </SheetHeader>
          {content}
        </SheetContent>
      </Sheet>
    );
  }

  // If used as a component, return content directly
  return content;
}
