import { useEffect, useMemo, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { SearchBar } from "@/components/shared/SearchBar";
import {
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Loader2,
  ScanFace,
} from "lucide-react";
import { useProjects } from "@/hooks/api/useProjects";
import {
  useProjectEmployees,
  useToggleCheckInEnabled,
  useBulkToggleCheckInEnabled,
} from "@/hooks/api/useProjectEmployees";

interface CheckInBulkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const EMPLOYEES_PER_PAGE = 50;

export function CheckInBulkDialog({ open, onOpenChange }: CheckInBulkDialogProps) {
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [search, setSearch] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [pendingEmployeeId, setPendingEmployeeId] = useState<number | null>(null);

  const { data: projectsData } = useProjects(
    { status: ["active"], pageSize: 200 },
    { enabled: open }
  );

  const employeeParams = useMemo(() => {
    const trimmedSearch = search.trim();
    return {
      page: currentPage,
      pageSize: EMPLOYEES_PER_PAGE,
      status: "current" as const,
      ...(trimmedSearch ? { search: trimmedSearch } : {}),
    };
  }, [currentPage, search]);

  const {
    data: employeesData,
    isLoading: loadingEmployees,
    isFetching: fetchingEmployees,
  } = useProjectEmployees(
    selectedProjectId ?? 0,
    employeeParams,
    open && !!selectedProjectId
  );

  const toggleCheckIn = useToggleCheckInEnabled();
  const bulkToggle = useBulkToggleCheckInEnabled();

  const flexibleProjects = useMemo(
    () => (projectsData?.data ?? []).filter((p) => p.is_flexible),
    [projectsData]
  );

  const pageEmployees = useMemo(() => employeesData?.data ?? [], [employeesData?.data]);
  const totalRecords = employeesData?.pagination.totalRecords ?? 0;
  const totalPages = employeesData?.pagination.totalPages ?? 0;
  const isLoadingEmployeePage = loadingEmployees || fetchingEmployees;

  useEffect(() => {
    if (!open || flexibleProjects.length === 0) return;
    const selectedProjectExists = flexibleProjects.some(
      (project) => project.id === selectedProjectId
    );
    if (selectedProjectExists) return;
    setSelectedProjectId(flexibleProjects[0].id);
    setSearch("");
    setCurrentPage(1);
    setErrorMsg(null);
  }, [flexibleProjects, open, selectedProjectId]);

  const handleProjectChange = (value: string) => {
    setSelectedProjectId(Number(value));
    setSearch("");
    setCurrentPage(1);
    setErrorMsg(null);
  };

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setCurrentPage(1);
  };

  const allPageEmployeesEnabled =
    pageEmployees.length > 0 &&
    pageEmployees.every((e) => e.check_in_enabled);

  const isMutating = toggleCheckIn.isPending || bulkToggle.isPending;

  const handleToggleEmployee = (employeeId: number, enabled: boolean) => {
    if (!selectedProjectId || isMutating) return;
    setErrorMsg(null);
    setPendingEmployeeId(employeeId);
    toggleCheckIn.mutate(
      { projectId: selectedProjectId, employeeId, enabled },
      {
        onError: (err: unknown) => {
          const e = err as { response?: { data?: { message?: string } }; message?: string };
          const msg =
            e?.response?.data?.message ||
            e?.message ||
            "Có lỗi xảy ra. Vui lòng thử lại.";
          setErrorMsg(msg);
        },
        onSettled: () => setPendingEmployeeId(null),
      }
    );
  };

  const handleBulkTogglePage = () => {
    if (!selectedProjectId || pageEmployees.length === 0 || isMutating) return;
    const enabled = !allPageEmployeesEnabled;
    setErrorMsg(null);
    bulkToggle.mutate(
      {
        projectId: selectedProjectId,
        employeeIds: pageEmployees.map((employee) => employee.employee_id),
        enabled,
      },
      {
        onError: (err: unknown) => {
          // Surface backend validation errors (e.g. no payrate) inside the dialog
          const e = err as { response?: { data?: { message?: string } }; message?: string };
          const msg =
            e?.response?.data?.message ||
            e?.message ||
            "Có lỗi xảy ra. Vui lòng thử lại.";
          setErrorMsg(msg);
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[92dvh] w-full max-w-3xl flex-col overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ScanFace className="h-5 w-5" />
            Cấu hình điểm danh
          </DialogTitle>
          <DialogDescription>
            Chọn dự án và nhân viên để bật / tắt tính năng điểm danh.
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto pt-1">
          {/* Top row: project selector + search */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:items-end">
            <div className="min-w-0 space-y-1.5">
              <label className="text-sm font-medium">Dự án</label>
              <Select
                value={selectedProjectId ? String(selectedProjectId) : ""}
                onValueChange={handleProjectChange}
              >
                <SelectTrigger className="h-11">
                  <SelectValue placeholder="Chọn dự án..." />
                </SelectTrigger>
                <SelectContent>
                  {flexibleProjects.length === 0 ? (
                    <SelectItem value="__none" disabled>
                      Không có dự án linh động
                    </SelectItem>
                  ) : (
                    flexibleProjects.map((p) => (
                      <SelectItem key={p.id} value={String(p.id)}>
                        {p.name}
                        {p.code && (
                          <span className="ml-1.5 text-muted-foreground text-xs">
                            ({p.code})
                          </span>
                        )}
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
            </div>

            {selectedProjectId && (
              <div className="min-w-0 space-y-1.5">
                <label
                  htmlFor="check-in-employee-search"
                  className="text-sm font-medium"
                >
                  Tìm kiếm nhân viên
                </label>
                <SearchBar
                  key={selectedProjectId}
                  inputId="check-in-employee-search"
                  searchTerm={search}
                  onSearchChange={handleSearchChange}
                  placeholder="Tên hoặc CCCD..."
                  className="w-full rounded-md"
                />
              </div>
            )}
          </div>

          {/* Error banner */}
          {errorMsg && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{errorMsg}</AlertDescription>
            </Alert>
          )}

          {/* Employee grid */}
          {selectedProjectId && (
            <>
              {/* Toolbar */}
              {!isLoadingEmployeePage && (
                <div className="flex flex-col gap-2 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <p className="text-sm text-muted-foreground">
                    {search ? `${totalRecords} kết quả` : `${totalRecords} nhân viên`}
                  </p>
                  {pageEmployees.length > 0 && (
                    <button
                      onClick={handleBulkTogglePage}
                      disabled={isMutating}
                      className="inline-flex min-h-11 w-fit items-center rounded-lg text-sm font-medium text-primary hover:underline disabled:pointer-events-none disabled:opacity-50"
                    >
                      {allPageEmployeesEnabled ? "Tắt trang này" : "Bật trang này"}
                    </button>
                  )}
                </div>
              )}

              {/* Grid */}
              {isLoadingEmployeePage ? (
                <div className="flex justify-center py-16">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : pageEmployees.length === 0 ? (
                <div className="rounded-lg border border-dashed py-12 text-center text-sm text-muted-foreground">
                  {search ? "Không tìm thấy nhân viên phù hợp" : "Dự án chưa có nhân viên"}
                </div>
              ) : (
                <div className="grid max-h-[52dvh] grid-cols-1 gap-2 overflow-y-auto pr-1 min-[380px]:grid-cols-2 sm:grid-cols-3">
                  {pageEmployees.map((emp) => {
                    const checked = Boolean(emp.check_in_enabled);
                    const isPending = pendingEmployeeId === emp.employee_id || bulkToggle.isPending;
                    return (
                      <div
                        role="switch"
                        tabIndex={isMutating ? -1 : 0}
                        key={emp.employee_id}
                        onClick={() => handleToggleEmployee(emp.employee_id, !checked)}
                        onKeyDown={(event) => {
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            handleToggleEmployee(emp.employee_id, !checked);
                          }
                        }}
                        aria-checked={checked}
                        aria-disabled={isMutating}
                        className={`
                          relative flex min-h-24 w-full items-start gap-2.5 rounded-lg border p-3 text-left select-none transition-colors
                          focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2
                          ${checked
                            ? "border-primary bg-primary/5"
                            : "border-border hover:bg-muted/40"}
                          ${isMutating ? "cursor-not-allowed opacity-70" : "cursor-pointer"}
                        `}
                      >
                        <Checkbox
                          checked={checked}
                          disabled={isMutating}
                          aria-hidden="true"
                          tabIndex={-1}
                          className="mt-0.5 shrink-0 pointer-events-none"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="break-words text-sm font-medium leading-tight">
                            {emp.employee_name}
                          </div>
                          <div className="mt-0.5 break-all text-xs text-muted-foreground">
                            {emp.employee_cccd}
                          </div>
                          <Badge
                            variant={emp.check_in_enabled ? "default" : "secondary"}
                            className="text-xs mt-1.5"
                          >
                            {emp.check_in_enabled ? "Đang bật" : "Đang tắt"}
                          </Badge>
                        </div>
                        {isPending && (
                          <Loader2 className="absolute right-3 top-3 h-4 w-4 animate-spin text-muted-foreground" />
                        )}
                      </div>
                    );
                  })}
                </div>
              )}

              {!isLoadingEmployeePage && totalPages > 1 && (
                <nav
                  aria-label="Phân trang nhân viên"
                  className="flex flex-wrap items-center justify-between gap-2 border-t pt-3"
                >
                  <p className="text-sm text-muted-foreground">
                    Trang {currentPage} / {totalPages}
                  </p>
                  <div className="flex items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      className="h-11 w-11"
                      aria-label="Trang trước"
                      onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
                      disabled={currentPage === 1 || isMutating}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      className="h-11 w-11"
                      aria-label="Trang sau"
                      onClick={() =>
                        setCurrentPage((page) => Math.min(totalPages, page + 1))
                      }
                      disabled={currentPage === totalPages || isMutating}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </nav>
              )}
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
