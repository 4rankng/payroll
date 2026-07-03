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
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loader2, ScanFace, Search, AlertCircle } from "lucide-react";
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

export function CheckInBulkDialog({ open, onOpenChange }: CheckInBulkDialogProps) {
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [search, setSearch] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [pendingEmployeeId, setPendingEmployeeId] = useState<number | null>(null);

  const { data: projectsData } = useProjects(
    { status: ["active"], pageSize: 200 },
    { enabled: open }
  );

  const employeeParams = useMemo(
    () => ({ pageSize: 200, status: "current" as const }),
    []
  );

  const { data: employeesData, isLoading: loadingEmployees } = useProjectEmployees(
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

  const allEmployees = useMemo(() => employeesData?.data ?? [], [employeesData?.data]);

  const filteredEmployees = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return allEmployees;
    return allEmployees.filter(
      (e) =>
        e.employee_name.toLowerCase().includes(q) ||
        e.employee_cccd.toLowerCase().includes(q)
    );
  }, [allEmployees, search]);

  useEffect(() => {
    if (!open || flexibleProjects.length === 0) return;
    const selectedProjectExists = flexibleProjects.some(
      (project) => project.id === selectedProjectId
    );
    if (selectedProjectExists) return;
    setSelectedProjectId(flexibleProjects[0].id);
    setSearch("");
    setErrorMsg(null);
  }, [flexibleProjects, open, selectedProjectId]);

  const handleProjectChange = (value: string) => {
    setSelectedProjectId(Number(value));
    setSearch("");
    setErrorMsg(null);
  };

  const allFilteredEnabled =
    filteredEmployees.length > 0 &&
    filteredEmployees.every((e) => e.check_in_enabled);

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

  const handleBulkToggleFiltered = () => {
    if (!selectedProjectId || filteredEmployees.length === 0 || isMutating) return;
    const enabled = !allFilteredEnabled;
    setErrorMsg(null);
    bulkToggle.mutate(
      {
        projectId: selectedProjectId,
        employeeIds: filteredEmployees.map((employee) => employee.employee_id),
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
                value={selectedProjectId ? String(selectedProjectId) : undefined}
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
                <label className="text-sm font-medium">Tìm kiếm nhân viên</label>
                <div className="relative">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
                  <Input
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder="Tên hoặc CCCD..."
                    className="h-11 pl-9"
                  />
                </div>
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
              {!loadingEmployees && (
                <div className="flex flex-col gap-2 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <p className="text-sm text-muted-foreground">
                    {search
                      ? `${filteredEmployees.length} / ${allEmployees.length} nhân viên`
                      : `${allEmployees.length} nhân viên`}
                  </p>
                  {filteredEmployees.length > 0 && (
                    <button
                      onClick={handleBulkToggleFiltered}
                      disabled={isMutating}
                      className="inline-flex min-h-11 w-fit items-center rounded-lg text-sm font-medium text-primary hover:underline disabled:pointer-events-none disabled:opacity-50"
                    >
                      {allFilteredEnabled ? "Tắt tất cả" : "Bật tất cả"}
                    </button>
                  )}
                </div>
              )}

              {/* Grid */}
              {loadingEmployees ? (
                <div className="flex justify-center py-16">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : filteredEmployees.length === 0 ? (
                <div className="rounded-lg border border-dashed py-12 text-center text-sm text-muted-foreground">
                  {search ? "Không tìm thấy nhân viên phù hợp" : "Dự án chưa có nhân viên"}
                </div>
              ) : (
                <div className="grid max-h-[52dvh] grid-cols-1 gap-2 overflow-y-auto pr-1 min-[380px]:grid-cols-2 sm:grid-cols-3">
                  {filteredEmployees.map((emp) => {
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
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
